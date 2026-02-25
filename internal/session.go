package internal

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Session 表示一个 Claude Code 会话（使用 tmux）
type Session struct {
	ID              string
	ClaudeSessionID string // 真实的 Claude Code session ID
	CWD             string
	TmuxSessionName string
	Status          string // running, stopped, need_permission
	CreatedAt       time.Time
	LastActivity    time.Time
	mu              sync.Mutex
}

// SessionManager 管理所有会话
type SessionManager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

// NewSessionManager 创建新的会话管理器
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
	}
}

// findClaudeBinary 查找 claude 命令的路径
func findClaudeBinary() (string, error) {
	// 检查 PATH 中是否有 claude
	path, err := exec.LookPath("claude")
	if err == nil {
		return path, nil
	}

	// 检查常见安装位置
	possiblePaths := []string{
		"/usr/local/bin/claude",
		"/usr/bin/claude",
	}

	for _, p := range possiblePaths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", errors.New("claude command not found in PATH")
}

// tmuxCmd 创建一个使用独立 socket 的 tmux 命令
func tmuxCmd(args ...string) *exec.Cmd {
	fullArgs := []string{"-L", "claude-pty"}
	fullArgs = append(fullArgs, args...)
	return exec.Command("tmux", fullArgs...)
}

// runTmuxCommand 运行 tmux 命令
func runTmuxCommand(args ...string) error {
	cmd := tmuxCmd(args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tmux %v: %w: %s", args, err, string(out))
	}
	return nil
}

// CreateSession 创建一个新的 Claude Code 会话（使用 tmux）
func (sm *SessionManager) CreateSession(sessionID, cwd string) (*Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.sessions[sessionID]; exists {
		return nil, ErrSessionExists
	}

	// 查找 claude 命令
	claudePath, err := findClaudeBinary()
	if err != nil {
		return nil, err
	}

	// 确保目录存在
	if _, err := os.Stat(cwd); err != nil {
		return nil, err
	}

	// 生成 tmux 会话名称
	tmuxSessionName := "claude-" + sessionID[:8]

	// 获取 settings 文件路径（可选）
	// 获取 settings 文件路径
	// 优先使用环境变量，否则使用默认的 settings.example.json
	settingsPath := os.Getenv("CLAUDE_PTY_SETTINGS")
	if settingsPath == "" {
		// 尝试查找项目目录下的 settings.example.json
		execPath, err := os.Executable()
		if err == nil {
			// 假设 server 在 bin/ 目录，settings.example.json 在项目根目录
			projectDir := filepath.Dir(filepath.Dir(execPath))
			defaultSettings := filepath.Join(projectDir, "settings.example.json")
			if _, err := os.Stat(defaultSettings); err == nil {
				settingsPath = defaultSettings
			}
		}
	}

	// 转换为绝对路径（tmux 可能在不同目录运行）
	if settingsPath != "" && !filepath.IsAbs(settingsPath) {
		absPath, err := filepath.Abs(settingsPath)
		if err == nil {
			settingsPath = absPath
		}
	}

	// 构建启动命令
	// 使用 -e 设置环境变量 CLAUDE_PTY_SESSION_ID 传给 hook

	var tmuxArgs []string
	if settingsPath != "" {
		tmuxArgs = []string{"new-session", "-d", "-s", tmuxSessionName, "-x", "80", "-y", "40", "-c", cwd, "-e", "CLAUDE_PTY_SESSION_ID=" + sessionID, "--", claudePath, "--settings", settingsPath}
	} else {
		tmuxArgs = []string{"new-session", "-d", "-s", tmuxSessionName, "-x", "80", "-y", "40", "-c", cwd, "-e", "CLAUDE_PTY_SESSION_ID=" + sessionID, "--", claudePath}
	}

	// 使用 tmux new-session 创建会话
	// -d: 分离模式（后台运行）
	// -s: 会话名称
	// -c: 工作目录
	err = runTmuxCommand(tmuxArgs...)
	if err != nil {
		return nil, err
	}

	// 等待一下让 Claude Code 启动
	time.Sleep(3 * time.Second)

	// 查找 Claude Code 的真实 session ID
	realSessionID, err := findClaudeSessionIDFromTmux(tmuxSessionName)
	if err != nil {
		// 如果找不到，使用原来的 UUID
		fmt.Printf("Warning: could not find real session ID for tmux session %s: %v\n", tmuxSessionName, err)
		realSessionID = sessionID
	}

	session := &Session{
		ID:              sessionID,
		ClaudeSessionID: realSessionID,
		CWD:             cwd,
		TmuxSessionName: tmuxSessionName,
		Status:          "stopped",
		CreatedAt:       time.Now(),
		LastActivity:    time.Now(),
	}

	sm.sessions[sessionID] = session
	return session, nil
}

// GetSession 获取会话
func (sm *SessionManager) GetSession(sessionID string) (*Session, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, ok := sm.sessions[sessionID]
	if !ok {
		return nil, ErrSessionNotFound
	}
	return session, nil
}

// DeleteSession 删除会话
func (sm *SessionManager) DeleteSession(sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[sessionID]
	if !ok {
		return ErrSessionNotFound
	}

	// 使用 tmux kill-session 删除 tmux 会话
	runTmuxCommand("kill-session", "-t", session.TmuxSessionName)

	delete(sm.sessions, sessionID)
	return nil
}

// ListSessions 列出所有会话
func (sm *SessionManager) ListSessions() []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessions := make([]*Session, 0, len(sm.sessions))
	for _, s := range sm.sessions {
		sessions = append(sessions, s)
	}
	return sessions
}

// SetStatus 设置会话状态
func (sm *SessionManager) SetStatus(sessionID string, status string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[sessionID]
	if !ok {
		return ErrSessionNotFound
	}

	session.mu.Lock()
	defer session.mu.Unlock()
	if status != "" {
		session.Status = status
	}
	session.LastActivity = time.Now()
	return nil
}

// GetStatus 获取会话状态
func (sm *SessionManager) GetStatus(sessionID string) (string, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, ok := sm.sessions[sessionID]
	if !ok {
		return "", ErrSessionNotFound
	}

	session.mu.Lock()
	defer session.mu.Unlock()
	return session.Status, nil
}

// GetMessages 获取会话消息历史
func (sm *SessionManager) GetMessages(sessionID string, limit int) ([]*Message, error) {
	sm.mu.RLock()
	session, ok := sm.sessions[sessionID]
	sm.mu.RUnlock()

	if !ok {
		return nil, ErrSessionNotFound
	}

	realSessionID := session.ClaudeSessionID
	if realSessionID == "" {
		realSessionID = sessionID
	}

	// 查找 jsonl 文件
	home := os.Getenv("HOME")
	projectsDir := filepath.Join(home, ".claude", "projects")

	var jsonlPath string
	entries, _ := os.ReadDir(projectsDir)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		candidate := filepath.Join(projectsDir, entry.Name(), realSessionID+".jsonl")
		if _, err := os.Stat(candidate); err == nil {
			jsonlPath = candidate
			break
		}
	}

	if jsonlPath == "" {
		return nil, fmt.Errorf("session file not found for %s", realSessionID)
	}

	// 读取 jsonl 文件
	file, err := os.Open(jsonlPath)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	var messages []*Message
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Bytes()

		// 先解析外层
		var outer struct {
			Type    string          `json:"type"`
			Message json.RawMessage `json:"message"`
		}
		if err := json.Unmarshal(line, &outer); err != nil {
			continue
		}

		if outer.Type == "user" {
			// user 消息: content 可以是 string 或 list
			var msg struct {
				Content interface{} `json:"content"`
			}
			if err := json.Unmarshal(outer.Message, &msg); err != nil {
				continue
			}

			switch c := msg.Content.(type) {
			case string:
				if c != "" {
					messages = append(messages, &Message{Type: "user", Content: c})
				}
			case []interface{}:
				for _, item := range c {
					if itemMap, ok := item.(map[string]interface{}); ok {
						if itemMap["type"] == "text" {
							if text, ok := itemMap["text"].(string); ok && text != "" {
								messages = append(messages, &Message{Type: "user", Content: text})
							}
						}
						// tool_result 是工具调用返回值，不是真实用户消息，跳过
					}
				}
			}
		} else if outer.Type == "assistant" {
			// assistant 消息: content 是 list
			var msg struct {
				Content []struct {
					Type  string          `json:"type"`
					Text  string          `json:"text,omitempty"`
					Name  string          `json:"name,omitempty"`
					Input json.RawMessage `json:"input,omitempty"`
				} `json:"content"`
			}
			if err := json.Unmarshal(outer.Message, &msg); err != nil {
				continue
			}

			for _, item := range msg.Content {
				if item.Type == "text" && item.Text != "" {
					messages = append(messages, &Message{Type: "assistant", Content: item.Text})
				} else if item.Type == "tool_use" && item.Name != "" {
					inputBytes, _ := json.Marshal(item.Input)
					content := fmt.Sprintf("%s: %s", item.Name, string(inputBytes))
					messages = append(messages, &Message{Type: "tool", Content: content})
				}
			}
		}
	}

	// 限制返回数量
	if limit > 0 && len(messages) > limit {
		messages = messages[len(messages)-limit:]
	}

	return messages, nil
}

// WriteToSession 向会话发送输入
func (sm *SessionManager) WriteToSession(sessionID, text string) (int, error) {
	sm.mu.RLock()
	session, ok := sm.sessions[sessionID]
	sm.mu.RUnlock()

	if !ok {
		return 0, ErrSessionNotFound
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	// 使用 tmux send-keys 发送输入
	err := runTmuxCommand("send-keys", "-t", session.TmuxSessionName, text)
	if err != nil {
		return 0, err
	}

	// 发送换行符（如果文本中没有换行）
	// if text[len(text)-1] != '\n' {
	// 	err = runTmuxCommand("send-keys", "-t", session.TmuxSessionName, "Enter")
	// 	if err != nil {
	// 		return 0, err
	// 	}
	// }

	session.LastActivity = time.Now()

	return len(text), nil
}

// ReadFromSession 从会话读取输出
func (sm *SessionManager) ReadFromSession(sessionID string) (string, error) {
	sm.mu.RLock()
	session, ok := sm.sessions[sessionID]
	sm.mu.RUnlock()

	if !ok {
		return "", ErrSessionNotFound
	}

	// 使用 tmux capture-pane 获取输出
	// -p: 输出到 stdout
	// -t: 目标会话
	// -S -: 从最后一行开始（获取所有历史）
	cmd := tmuxCmd("capture-pane", "-p", "-t", session.TmuxSessionName, "-S", "-")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	session.mu.Lock()
	session.LastActivity = time.Now()
	session.mu.Unlock()

	return string(output), nil
}

// generateSessionID 生成会话 ID
func generateSessionID() string {
	return uuid.New().String()
}

// findClaudeSessionIDFromTmux 从 tmux 会话中获取 Claude Code 的真实 session ID
func findClaudeSessionIDFromTmux(tmuxSessionName string) (string, error) {
	// 发送 /status 命令
	cmd := tmuxCmd("send-keys", "-t", tmuxSessionName, "/status", "Enter")
	if err := cmd.Run(); err != nil {
		return "", err
	}

	time.Sleep(100 * time.Millisecond)
	tmuxCmd("send-keys", "-t", tmuxSessionName, "Enter").Run()

	// 等待输出
	time.Sleep(800 * time.Millisecond)

	// 捕获输出
	cmd = tmuxCmd("capture-pane", "-p", "-t", tmuxSessionName)
	output, err := cmd.Output()

	fmt.Printf("Debug: tmux capture-pane output:\n%s\n", string(output))

	if err != nil {
		return "", err
	}

	// 查找 "Session ID: xxx" 模式
	outputStr := string(output)
	for _, line := range strings.Split(outputStr, "\n") {
		if strings.Contains(line, "Session ID:") {
			// 提取 session ID
			parts := strings.Split(line, "Session ID:")
			if len(parts) > 1 {
				id := strings.TrimSpace(parts[1])
				// 去除可能的其他字符
				id = strings.Fields(id)[0]
				if len(id) == 36 { // UUID 长度
					// 发送 Escape 退出 status 模式
					tmuxCmd("send-keys", "-t", tmuxSessionName, "C-[").Run()
					fmt.Printf("Debug: Found real session ID: %s\n", id)
					return id, nil
				}
			}
		}
	}

	// 如果没找到，尝试发送 Enter 退出
	tmuxCmd("send-keys", "-t", tmuxSessionName, "Enter").Run()

	return "", fmt.Errorf("session ID not found in status output")
}

package internal

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Session 表示一个 Claude Code 会话（使用 tmux）
type Session struct {
	ID              string
	CWD             string
	TmuxSessionName string
	WaitingForInput bool
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

// runTmuxCommand 运行 tmux 命令
func runTmuxCommand(args ...string) error {
	// 使用 -L default 明确指定默认 socket
	fullArgs := []string{"-L", "default"}
	fullArgs = append(fullArgs, args...)

	cmd := exec.Command("tmux", fullArgs...)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tmux %v: %w: %s", fullArgs, err, string(out))
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
		// 使用 --settings 参数启动 claude
		tmuxArgs = []string{"new-session", "-d", "-s", tmuxSessionName, "-c", cwd, "-e", "CLAUDE_PTY_SESSION_ID=" + sessionID, "--", claudePath, "--settings", settingsPath}
	} else {
		tmuxArgs = []string{"new-session", "-d", "-s", tmuxSessionName, "-c", cwd, "-e", "CLAUDE_PTY_SESSION_ID=" + sessionID, "--", claudePath}
	}

	// 使用 tmux new-session 创建会话
	// -d: 分离模式（后台运行）
	// -s: 会话名称
	// -c: 工作目录
	err = runTmuxCommand(tmuxArgs...)
	if err != nil {
		return nil, err
	}

	session := &Session{
		ID:              sessionID,
		CWD:             cwd,
		TmuxSessionName: tmuxSessionName,
		WaitingForInput: false,
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
func (sm *SessionManager) SetStatus(sessionID string, waiting bool) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[sessionID]
	if !ok {
		return ErrSessionNotFound
	}

	session.mu.Lock()
	defer session.mu.Unlock()
	session.WaitingForInput = waiting
	session.LastActivity = time.Now()
	return nil
}

// GetStatus 获取会话状态
func (sm *SessionManager) GetStatus(sessionID string) (bool, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, ok := sm.sessions[sessionID]
	if !ok {
		return false, ErrSessionNotFound
	}

	session.mu.Lock()
	defer session.mu.Unlock()
	return session.WaitingForInput, nil
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
	if text[len(text)-1] != '\n' {
		err = runTmuxCommand("send-keys", "-t", session.TmuxSessionName, "Enter")
		if err != nil {
			return 0, err
		}
	}

	session.LastActivity = time.Now()
	session.WaitingForInput = false

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
	cmd := exec.Command("tmux", "capture-pane", "-p", "-t", session.TmuxSessionName, "-S", "-")
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

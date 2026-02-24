package internal

import (
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/creack/pty"
)

// Session 表示一个 Claude Code 会话
type Session struct {
	ID                string
	CWD               string
	Cmd               *exec.Cmd
	PTy               *os.File
	WaitingForInput   bool
	CreatedAt         time.Time
	LastActivity      time.Time
	mu                sync.Mutex
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

// CreateSession 创建一个新的 Claude Code 会话
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

	cmd := exec.Command(claudePath)
	cmd.Dir = cwd

	// 创建 PTY
	pt, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}

	session := &Session{
		ID:              sessionID,
		CWD:             cwd,
		Cmd:             cmd,
		PTy:             pt,
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

	// 关闭 PTY 和进程
	session.PTy.Close()
	if session.Cmd.Process != nil {
		session.Cmd.Process.Kill()
		session.Cmd.Wait()
	}

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

	n, err := session.PTy.WriteString(text)
	if err == nil {
		session.LastActivity = time.Now()
		session.WaitingForInput = false
	}
	return n, err
}

// ReadFromSession 从会话读取输出
func (sm *SessionManager) ReadFromSession(sessionID string) (string, error) {
	sm.mu.RLock()
	session, ok := sm.sessions[sessionID]
	sm.mu.RUnlock()

	if !ok {
		return "", ErrSessionNotFound
	}

	buf := make([]byte, 4096)
	n, err := session.PTy.Read(buf)
	if err != nil {
		return "", err
	}

	session.mu.Lock()
	session.LastActivity = time.Now()
	session.mu.Unlock()

	return string(buf[:n]), nil
}

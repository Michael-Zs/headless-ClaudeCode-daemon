package internal

import "errors"

// 错误定义
var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExists   = errors.New("session already exists")
	ErrInvalidRequest  = errors.New("invalid request")
)

// Request 表示客户端请求
type Request struct {
	Action    string `json:"action"`
	SessionID string `json:"session_id,omitempty"`
	CWD       string `json:"cwd,omitempty"`
	Text      string `json:"text,omitempty"`
	Status    string `json:"status,omitempty"`
}

// Response 表示服务端响应
type Response struct {
	Success  bool        `json:"success"`
	Error    string      `json:"error,omitempty"`
	Session  *SessionInfo `json:"session,omitempty"`
	Sessions []*SessionInfo `json:"sessions,omitempty"`
	Output   string      `json:"output,omitempty"`
	Status   string     `json:"status,omitempty"`
}

// SessionInfo 会话信息（用于 JSON 序列化）
type SessionInfo struct {
	ID               string `json:"id"`
	ClaudeSessionID string `json:"claude_session_id,omitempty"`
	CWD              string `json:"cwd"`
	Status           string `json:"status"`
	CreatedAt        string `json:"created_at"`
	LastActivity     string `json:"last_activity"`
}

// ToSessionInfo 将 Session 转换为 SessionInfo
func (s *Session) ToSessionInfo() *SessionInfo {
	return &SessionInfo{
		ID:               s.ID,
		ClaudeSessionID:  s.ClaudeSessionID,
		CWD:              s.CWD,
		Status:           s.Status,
		CreatedAt:        s.CreatedAt.Format("2006-01-02 15:04:05"),
		LastActivity:     s.LastActivity.Format("2006-01-02 15:04:05"),
	}
}

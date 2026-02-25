package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

const (
	SocketPath = "/tmp/claude-pty.sock"
)

// Server 表示 PTY Server
type Server struct {
	socketPath string
	sessionMgr *SessionManager
	httpServer *http.Server
	logger     *log.Logger
}

// NewServer 创建新的 Server
func NewServer(socketPath string) *Server {
	if socketPath == "" {
		socketPath = SocketPath
	}

	return &Server{
		socketPath: socketPath,
		sessionMgr: NewSessionManager(),
		logger:     log.New(os.Stdout, "[claude-pty] ", log.LstdFlags),
	}
}

// Start 启动 Server
func (s *Server) Start() error {
	// 移除已存在的 socket 文件
	if err := os.Remove(s.socketPath); err != nil && !os.IsNotExist(err) {
		s.logger.Printf("warning: remove old socket: %v", err)
	}

	// 创建 Unix socket listener
	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("listen on unix socket: %w", err)
	}

	// 设置 socket 权限
	if err := os.Chmod(s.socketPath, 0777); err != nil {
		s.logger.Printf("warning: chmod socket: %v", err)
	}

	// 创建 HTTP 服务器
	s.httpServer = &http.Server{
		Handler: s,
	}

	s.logger.Printf("Server listening on %s", s.socketPath)

	// 启动 HTTP 服务器
	return s.httpServer.Serve(listener)
}

// Stop 停止 Server
func (s *Server) Stop() error {
	// 先清理所有 tmux 会话
	s.logger.Println("Cleaning up all tmux sessions...")
	sessions := s.sessionMgr.ListSessions()
	for _, session := range sessions {
		if err := s.sessionMgr.DeleteSession(session.ID); err != nil {
			s.logger.Printf("warning: failed to delete session %s: %v", session.ID, err)
		} else {
			s.logger.Printf("Deleted tmux session: %s", session.TmuxSessionName)
		}
	}

	if s.httpServer != nil {
		return s.httpServer.Close()
	}
	return nil
}

// ServeHTTP 处理 HTTP 请求
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 只接受本地 socket 连接
	if r.RemoteAddr != "@" && r.RemoteAddr != "" {
		s.logger.Printf("rejecting request from %s", r.RemoteAddr)
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// 设置响应头
	w.Header().Set("Content-Type", "application/json")

	switch r.URL.Path {
	case "/":
		s.handleRequest(w, r)
	case "/list":
		s.handleList(w, r)
	default:
		s.sendError(w, http.StatusNotFound, "not found")
	}
}

// handleRequest 处理请求
func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.sendError(w, http.StatusBadRequest, "read request body")
		return
	}

	var req Request
	if err := json.Unmarshal(body, &req); err != nil {
		s.sendError(w, http.StatusBadRequest, "parse JSON")
		return
	}

	var resp Response

	switch req.Action {
	case "create":
		resp = s.handleCreate(req)
	case "delete":
		resp = s.handleDelete(req)
	case "get":
		resp = s.handleGet(req)
	case "input":
		resp = s.handleInput(req)
	case "set_status":
		resp = s.handleSetStatus(req)
	case "get_status":
		resp = s.handleGetStatus(req)
	case "get_info":
		resp = s.handleGetInfo(req)
	case "messages":
		resp = s.handleMessages(req)
	default:
		resp = Response{Success: false, Error: "unknown action: " + req.Action}
	}

	s.sendResponse(w, resp)
}

// handleCreate 处理创建会话请求
func (s *Server) handleCreate(req Request) Response {
	cwd := req.CWD
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	// 确保目录存在
	if _, err := os.Stat(cwd); err != nil {
		return Response{Success: false, Error: "cwd not found: " + err.Error()}
	}

	sessionID := uuid.New().String()
	session, err := s.sessionMgr.CreateSession(sessionID, cwd)
	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}

	return Response{
		Success: true,
		Session: session.ToSessionInfo(),
	}
}

// handleDelete 处理删除会话请求
func (s *Server) handleDelete(req Request) Response {
	if req.SessionID == "" {
		return Response{Success: false, Error: "session_id required"}
	}

	err := s.sessionMgr.DeleteSession(req.SessionID)
	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}

	return Response{Success: true}
}

// handleGet 处理获取输出请求
func (s *Server) handleGet(req Request) Response {
	if req.SessionID == "" {
		return Response{Success: false, Error: "session_id required"}
	}

	output, err := s.sessionMgr.ReadFromSession(req.SessionID, req.LimitStr)
	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}

	return Response{Success: true, Output: output}
}

// handleInput 处理发送输入请求
func (s *Server) handleInput(req Request) Response {
	if req.SessionID == "" {
		return Response{Success: false, Error: "session_id required"}
	}
	if req.Text == "" {
		return Response{Success: false, Error: "text required"}
	}

	_, err := s.sessionMgr.WriteToSession(req.SessionID, req.Text)
	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}

	return Response{Success: true}
}

// handleSetStatus 处理设置状态请求
func (s *Server) handleSetStatus(req Request) Response {
	if req.SessionID == "" {
		return Response{Success: false, Error: "session_id required"}
	}

	err := s.sessionMgr.SetStatus(req.SessionID, req.Status)
	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}

	fmt.Printf("Session %s status updated: status=%s\n", req.SessionID, req.Status)

	return Response{Success: true}
}

// handleGetStatus 处理获取状态请求
func (s *Server) handleGetStatus(req Request) Response {
	if req.SessionID == "" {
		return Response{Success: false, Error: "session_id required"}
	}

	status, err := s.sessionMgr.GetStatus(req.SessionID)
	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}

	return Response{Success: true, Status: status}
}

// handleGetInfo 处理获取 session 信息请求
func (s *Server) handleGetInfo(req Request) Response {
	if req.SessionID == "" {
		return Response{Success: false, Error: "session_id required"}
	}

	session, err := s.sessionMgr.GetSession(req.SessionID)
	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}

	return Response{Success: true, Session: session.ToSessionInfo()}
}

// handleMessages 处理获取消息历史请求
func (s *Server) handleMessages(req Request) Response {
	if req.SessionID == "" {
		return Response{Success: false, Error: "session_id required"}
	}

	messages, err := s.sessionMgr.GetMessages(req.SessionID, req.Limit)
	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}

	return Response{Success: true, Messages: messages}
}

// handleList 处理列表请求
func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	sessions := s.sessionMgr.ListSessions()

	sessionInfos := make([]*SessionInfo, len(sessions))
	for i, s := range sessions {
		sessionInfos[i] = s.ToSessionInfo()
	}

	resp := Response{
		Success:  true,
		Sessions: sessionInfos,
	}

	s.sendResponse(w, resp)
}

func (s *Server) sendResponse(w http.ResponseWriter, resp Response) {
	data, err := json.Marshal(resp)
	if err != nil {
		s.logger.Printf("marshal response: %v", err)
		return
	}

	w.Write(data)
}

func (s *Server) sendError(w http.ResponseWriter, code int, msg string) {
	resp := Response{Success: false, Error: msg}
	data, _ := json.Marshal(resp)
	http.Error(w, string(data), code)
}

// getSocketDir 获取 socket 文件所在的目录
func getSocketDir() string {
	return "/tmp"
}

// GetDefaultSocketPath 获取默认的 socket 路径
func GetDefaultSocketPath() string {
	if path := os.Getenv("CLAUDE_PTY_SOCKET"); path != "" {
		return path
	}
	return filepath.Join(getSocketDir(), "claude-pty.sock")
}

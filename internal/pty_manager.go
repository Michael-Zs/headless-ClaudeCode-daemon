package internal

// 注意: PTY 管理逻辑已经集成到 session.go 中
// 现在使用 tmux 来管理终端会话，而不是直接使用 pty
// 相关的功能包括:
// - CreateSession: 使用 tmux new-session 创建会话
// - DeleteSession: 使用 tmux kill-session 删除会话
// - WriteToSession: 使用 tmux send-keys 发送输入
// - ReadFromSession: 使用 tmux capture-pane 读取输出

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"claude-pty/internal"
	"golang.org/x/term"
)

var socketPath = flag.String("socket", internal.GetDefaultSocketPath(), "Unix socket path")

// unixClient HTTP 客户端，使用 Unix socket
type unixClient struct {
	socketPath string
}

func (c *unixClient) do(action, sessionID, cwd, text, status string, limit ...int) (*internal.Response, error) {
	lim := 0
	if len(limit) > 0 {
		lim = limit[0]
	}

	reqBody := internal.Request{
		Action:    action,
		SessionID: sessionID,
		CWD:       cwd,
		Text:      text,
		Status:    status,
		Limit:     lim,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// 使用 Unix socket 创建 HTTP 请求
	url := "http://localhost/"
	req, err := http.NewRequest("POST", url, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// 创建 Unix 传输
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			dialer := &net.Dialer{}
			return dialer.DialContext(context.Background(), "unix", c.socketPath)
		},
	}

	client := &http.Client{Transport: transport}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var result internal.Response
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}

func (c *unixClient) list() (*internal.Response, error) {
	// 使用 Unix socket 创建 HTTP 请求
	url := "http://localhost/list"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// 创建 Unix 传输
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			dialer := &net.Dialer{}
			return dialer.DialContext(context.Background(), "unix", c.socketPath)
		},
	}

	client := &http.Client{Transport: transport}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var result internal.Response
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}

func cmdCreate(client *unixClient, args []string) {
	cwd := ""
	if len(args) > 0 {
		cwd = args[0]
	}

	resp, err := client.do("create", "", cwd, "", "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if !resp.Success {
		fmt.Fprintf(os.Stderr, "Error: %s\n", resp.Error)
		os.Exit(1)
	}

	fmt.Printf("Session created: %s\n", resp.Session.ID)
	fmt.Printf("Working directory: %s\n", resp.Session.CWD)
}

func cmdList(client *unixClient) {
	resp, err := client.list()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if !resp.Success {
		fmt.Fprintf(os.Stderr, "Error: %s\n", resp.Error)
		os.Exit(1)
	}

	if len(resp.Sessions) == 0 {
		fmt.Println("No sessions")
		return
	}

	fmt.Printf("%-36s %-20s %-15s\n", "ID", "CWD", "Status")
	fmt.Println(strings.Repeat("-", 75))
	for _, s := range resp.Sessions {
		fmt.Printf("%-36s %-20s %-15s\n", s.ID, s.CWD, s.Status)
	}
}

func cmdGet(client *unixClient, sessionID string, limit int) {
	resp, err := client.do("get", sessionID, "", "", "", limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if !resp.Success {
		fmt.Fprintf(os.Stderr, "Error: %s\n", resp.Error)
		os.Exit(1)
	}

	if resp.Output != "" {
		fmt.Print(resp.Output)
	}
}

func cmdInput(client *unixClient, sessionID, text string) {
	resp, err := client.do("input", sessionID, "", text, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if !resp.Success {
		fmt.Fprintf(os.Stderr, "Error: %s\n", resp.Error)
		os.Exit(1)
	}

	fmt.Println("Input sent")
}

func cmdDelete(client *unixClient, sessionID string) {
	resp, err := client.do("delete", sessionID, "", "", "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if !resp.Success {
		fmt.Fprintf(os.Stderr, "Error: %s\n", resp.Error)
		os.Exit(1)
	}

	fmt.Println("Session deleted")
}

func cmdLog(client *unixClient, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: claude-pty log <session_id> [limit]")
		os.Exit(1)
	}

	sessionID := args[0]
	limit := 0
	if len(args) > 1 {
		fmt.Sscanf(args[1], "%d", &limit)
	}

	// 调用 server 的 messages API
	resp, err := client.do("messages", sessionID, "", "", "", limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if !resp.Success {
		fmt.Fprintf(os.Stderr, "Error: %s\n", resp.Error)
		os.Exit(1)
	}

	// 显示消息
	for _, msg := range resp.Messages {
		switch msg.Type {
		case "user":
			fmt.Printf("\n[User]\n%s\n\n", msg.Content)
		case "assistant":
			fmt.Printf("[Claude]\n%s\n\n", msg.Content)
		case "tool":
			fmt.Printf("[Tool]\n%s\n\n", msg.Content)
		}
	}
}

func cmdInfo(client *unixClient, sessionID string) {
	resp, err := client.do("get_info", sessionID, "", "", "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if !resp.Success {
		fmt.Fprintf(os.Stderr, "Error: %s\n", resp.Error)
		os.Exit(1)
	}

	if resp.Session != nil {
		fmt.Printf("ID:              %s\n", resp.Session.ID)
		if resp.Session.ClaudeSessionID != "" {
			fmt.Printf("Claude Session: %s\n", resp.Session.ClaudeSessionID)
		}
		fmt.Printf("CWD:             %s\n", resp.Session.CWD)
		fmt.Printf("Status:          %s\n", resp.Session.Status)
		fmt.Printf("Created:         %s\n", resp.Session.CreatedAt)
		fmt.Printf("Last Activity:   %s\n", resp.Session.LastActivity)
	}
}

func cmdStatus(client *unixClient, sessionID string) {
	resp, err := client.do("get_status", sessionID, "", "", "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if !resp.Success {
		fmt.Fprintf(os.Stderr, "Error: %s\n", resp.Error)
		os.Exit(1)
	}

	fmt.Printf("Session %s status: %s\n", sessionID, resp.Status)
}

func cmdConnect(client *unixClient, sessionID string) {
	fmt.Printf("Connecting to session %s...\n", sessionID)
	fmt.Println("Press Ctrl+Q to disconnect")

	done := make(chan bool)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 保存终端状态
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error setting raw mode: %v\n", err)
		return
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// 读取输出的 goroutine（只打印新增内容）
	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()

		lastOutput := ""
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				resp, err := client.do("get", sessionID, "", "", "")
				if err != nil {
					return
				}
				if !resp.Success || resp.Output == "" {
					continue
				}
				// 只打印新增的部分
				if resp.Output == lastOutput {
					continue
				}
				newContent := resp.Output
				if strings.HasPrefix(resp.Output, lastOutput) {
					newContent = resp.Output[len(lastOutput):]
				} else {
					// 内容发生了变化（如清屏），全量输出
					fmt.Print("\r\033[2J\033[H") // 清屏
				}
				// raw mode 下 \n 需要转为 \r\n
				newContent = strings.ReplaceAll(newContent, "\n", "\r\n")
				fmt.Print(newContent)
				lastOutput = resp.Output
			}
		}
	}()

	// 读取用户输入并发送（逐字符）
	buf := make([]byte, 1)
	for {
		select {
		case <-sigChan:
			fmt.Print("\r\nDisconnected\r\n")
			close(done)
			return
		default:
			n, err := os.Stdin.Read(buf)
			if err != nil || n == 0 {
				close(done)
				return
			}
			// Ctrl+Q (0x11) 退出
			if buf[0] == 0x11 {
				fmt.Print("\r\nDisconnected\r\n")
				close(done)
				return
			}
			_, err = client.do("input", sessionID, "", string(buf[:n]), "")
			if err != nil {
				fmt.Fprintf(os.Stderr, "\r\nError: %v\r\n", err)
			}
		}
	}
}

func main() {
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: claude-pty <command> [arguments]")
		fmt.Println("Commands:")
		fmt.Println("  create [cwd]          Create a new session")
		fmt.Println("  list                  List all sessions")
		fmt.Println("  connect <session_id>  Connect to a session interactively")
		fmt.Println("  get <session_id> [limit]  Get output from a session")
		fmt.Println("  input <session_id> <text>  Send input to a session")
		fmt.Println("  delete <session_id>  Delete a session")
		fmt.Println("  info <session_id>    Get session information")
		fmt.Println("  status <session_id>  Get session status")
		os.Exit(1)
	}

	client := &unixClient{socketPath: *socketPath}

	cmd := args[0]
	switch cmd {
	case "create":
		cmdCreate(client, args[1:])
	case "list":
		cmdList(client)
	case "connect":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: claude-pty connect <session_id>")
			os.Exit(1)
		}
		cmdConnect(client, args[1])
	case "get":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: claude-pty get <session_id> [limit]")
			os.Exit(1)
		}
		limit := 0
		if len(args) >= 3 {
			fmt.Sscanf(args[2], "%d", &limit)
		}
		cmdGet(client, args[1], limit)
	case "input":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: claude-pty input <session_id> <text>")
			os.Exit(1)
		}
		cmdInput(client, args[1], args[2])
	case "delete":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: claude-pty delete <session_id>")
			os.Exit(1)
		}
		cmdDelete(client, args[1])
	case "info":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: claude-pty info <session_id>")
			os.Exit(1)
		}
		cmdInfo(client, args[1])
	case "log":
		cmdLog(client, args[1:])
	case "status":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: claude-pty status <session_id>")
			os.Exit(1)
		}
		cmdStatus(client, args[1])
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		os.Exit(1)
	}
}

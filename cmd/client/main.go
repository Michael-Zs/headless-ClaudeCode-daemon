package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"

	"claude-pty/internal"
)

const (
	DefaultSocketPath = "/tmp/claude-pty.sock"
)

var socketPath = flag.String("socket", DefaultSocketPath, "Unix socket path")

// unixClient HTTP 客户端，使用 Unix socket
type unixClient struct {
	socketPath string
}

func (c *unixClient) do(action, sessionID, cwd, text string, waiting bool) (*internal.Response, error) {
	reqBody := internal.Request{
		Action:    action,
		SessionID: sessionID,
		CWD:       cwd,
		Text:      text,
		Waiting:   waiting,
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

	resp, err := client.do("create", "", cwd, "", false)
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

	fmt.Printf("%-36s %-20s %-10s\n", "ID", "CWD", "Waiting")
	fmt.Println(strings.Repeat("-", 70))
	for _, s := range resp.Sessions {
		fmt.Printf("%-36s %-20s %-10v\n", s.ID, s.CWD, s.WaitingForInput)
	}
}

func cmdGet(client *unixClient, sessionID string) {
	for {
		resp, err := client.do("get", sessionID, "", "", false)
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

		// 简单的轮询间隔
		// 在实际使用中可能需要更好的机制
	}
}

func cmdInput(client *unixClient, sessionID, text string) {
	resp, err := client.do("input", sessionID, "", text, false)
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
	resp, err := client.do("delete", sessionID, "", "", false)
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

func cmdStatus(client *unixClient, sessionID string) {
	resp, err := client.do("get_status", sessionID, "", "", false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if !resp.Success {
		fmt.Fprintf(os.Stderr, "Error: %s\n", resp.Error)
		os.Exit(1)
	}

	fmt.Printf("Session %s waiting for input: %v\n", sessionID, resp.Waiting)
}

func cmdConnect(client *unixClient, sessionID string) {
	fmt.Printf("Connecting to session %s...\n", sessionID)
	fmt.Println("Press Ctrl+C to disconnect")

	// 创建一个读取输出的 goroutine
	outputChan := make(chan string)
	go func() {
		for {
			resp, err := client.do("get", sessionID, "", "", false)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				close(outputChan)
				return
			}

			if !resp.Success {
				fmt.Fprintf(os.Stderr, "Error: %s\n", resp.Error)
				close(outputChan)
				return
			}

			if resp.Output != "" {
				outputChan <- resp.Output
			}
		}
	}()

	// 读取用户输入并发送
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := scanner.Text() + "\n"
		resp, err := client.do("input", sessionID, "", text, false)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}

		if !resp.Success {
			fmt.Fprintf(os.Stderr, "Error: %s\n", resp.Error)
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
		fmt.Println("  get <session_id>      Get output from a session")
		fmt.Println("  input <session_id> <text>  Send input to a session")
		fmt.Println("  delete <session_id>  Delete a session")
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
			fmt.Fprintln(os.Stderr, "Usage: claude-pty get <session_id>")
			os.Exit(1)
		}
		cmdGet(client, args[1])
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

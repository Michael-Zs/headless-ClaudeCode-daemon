package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"time"
)

func main() {
	sessionName := "claude-test"

	// 先杀掉可能存在的旧 session
	exec.Command("tmux", "kill-session", "-t", sessionName).Run()

	// 创建新的 tmux session，运行 claude
	cmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName, "claude")
	cmd.Env = removeCLAUDECODE(os.Environ())
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting tmux: %v\n", err)
		os.Exit(1)
	}

	// 等待一下让 claude 启动
	time.Sleep(2 * time.Second)

	fmt.Println("Claude in tmux Started")
	fmt.Println("Session name:", sessionName)
	fmt.Println("Type your message and press Enter to submit.")
	fmt.Println("Press Ctrl+C to exit.")
	fmt.Println("---")

	// 启动一个 goroutine 定期捕获输出
	go func() {
		for {
			captureAndPrint(sessionName)
			time.Sleep(500 * time.Millisecond)
		}
	}()

	// 主循环：读取用户输入并发送到 tmux
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := scanner.Text()
		// 发送文本到 tmux
		exec.Command("tmux", "send-keys", "-t", sessionName, text, "Enter").Run()
	}
}

func captureAndPrint(sessionName string) {
	cmd := exec.Command("tmux", "capture-pane", "-t", sessionName, "-p")
	out, err := cmd.Output()
	if err != nil {
		return
	}
	output := string(out)
	if output != "" {
		fmt.Print(output)
	}
}

func removeCLAUDECODE(env []string) []string {
	newEnv := []string{}
	for _, e := range env {
		if len(e) >= 13 && e[:13] == "CLAUDECODE=" {
			continue
		}
		newEnv = append(newEnv, e)
	}
	return newEnv
}

package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

func main() {
	sessionName := "claude-auto-test"

	// 先杀掉可能存在的旧 session
	exec.Command("tmux", "kill-session", "-t", sessionName).Run()

	// 创建新的 tmux session，运行 claude
	cmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName, "claude")
	cmd.Env = removeCLAUDECODE(os.Environ())
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting tmux: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Claude in tmux Started")
	fmt.Println("Waiting 20 seconds...")
	fmt.Println("---")

	// 等待 20 秒
	time.Sleep(20 * time.Second)

	// 发送消息
	fmt.Println("Sending: Hello!")
	exec.Command("tmux", "send-keys", "-t", sessionName, "Hello!").Run()
	time.Sleep(100 * time.Millisecond)
	exec.Command("tmux", "send-keys", "-t", sessionName, "Enter").Run()

	// 等待回复
	fmt.Println("Waiting for response...")
	time.Sleep(10 * time.Second)

	// 捕获并显示输出
	fmt.Println("---")
	fmt.Println("Final output:")
	captureAndPrint(sessionName)
}

func captureAndPrint(sessionName string) {
	cmd := exec.Command("tmux", "capture-pane", "-t", sessionName, "-p")
	out, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error capturing: %v\n", err)
		return
	}
	fmt.Print(string(out))
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

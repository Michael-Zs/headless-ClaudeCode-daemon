package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/creack/pty"
)

func main() {
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: claude not found\n")
		os.Exit(1)
	}

	// 移除 CLAUDECODE 环境变量
	newEnv := []string{}
	for _, e := range os.Environ() {
		if len(e) >= 13 && e[:13] == "CLAUDECODE=" {
			continue
		}
		newEnv = append(newEnv, e)
	}

	cmd := exec.Command(claudePath)
	cmd.Env = newEnv

	pt, err := pty.Start(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer pt.Close()

	fmt.Println("Claude PTY Session Started")
	fmt.Println("Type your message. Ctrl+C to exit.")
	fmt.Println("---")

	// 从 pty 复制到 stdout
	go func() {
		buf := make([]byte, 8192)
		for {
			n, err := pt.Read(buf)
			if err != nil {
				return
			}
			os.Stdout.Write(buf[:n])
		}
	}()

	// 从 stdin 复制到 pty
	buf := make([]byte, 1)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			break
		}
		if n > 0 {
			pt.Write(buf[:n])
		}
	}
}

package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"claude-pty/internal"
)

func main() {
	socketPath := flag.String("socket", internal.GetDefaultSocketPath(), "Unix socket path")
	flag.Parse()

	logger := log.New(os.Stdout, "[claude-pty-server] ", log.LstdFlags)
	logger.Printf("Starting Claude PTY Server on %s", *socketPath)

	server := internal.NewServer(*socketPath)

	// 等待信号以优雅关闭
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Println("Shutting down...")
		server.Stop()
		os.Exit(0)
	}()

	if err := server.Start(); err != nil {
		logger.Fatalf("Server error: %v", err)
	}
}

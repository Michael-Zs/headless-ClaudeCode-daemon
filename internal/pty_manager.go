package internal

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
)

// findClaudeBinary 查找 claude 命令的路径
func findClaudeBinary() (string, error) {
	// 检查 PATH 中是否有 claude
	path, err := exec.LookPath("claude")
	if err == nil {
		return path, nil
	}

	// 检查常见安装位置
	possiblePaths := []string{
		"/usr/local/bin/claude",
		"/usr/bin/claude",
		filepath.Join(os.Getenv("HOME"), ".local/bin/claude"),
		filepath.Join(os.Getenv("HOME"), "bin/claude"),
	}

	for _, p := range possiblePaths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", errors.New("claude command not found in PATH")
}

// generateSessionID 生成会话 ID
func generateSessionID() string {
	// 简单使用时间戳和随机数
	return ""
}

#!/bin/bash

set -e

echo "Building claude-pty..."

# 创建 bin 目录
mkdir -p bin

# 编译 server
echo "Building server..."
go build -o bin/claude-pty-server ./cmd/server

# 编译 client
echo "Building client..."
go build -o bin/claude-pty-client ./cmd/client

echo "Build complete!"
echo "  bin/claude-pty-server"
echo "  bin/claude-pty-client"

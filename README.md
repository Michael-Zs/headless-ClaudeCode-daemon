# Claude PTY Server

通过 PTY 创建和管理 Claude Code 子进程的服务器和客户端工具。

## 功能特性

- **Server**: 通过 Unix Socket 提供 API，管理 Claude Code 会话
- **CLI**: 命令行客户端，支持创建、连接、发送输入等操作
- **Hook 集成**: 支持 Claude Code 的 Notification hook

## 编译

```bash
cd claude-pty
go build -o bin/server ./cmd/server
go build -o bin/client ./cmd/client
```

## 使用方法

### 1. 启动 Server

```bash
./bin/claude-pty-server
```

默认 socket 路径: `/run/user/1000/claude-pty.sock` (使用 XDG_RUNTIME_DIR)

可通过环境变量或参数指定:
```bash
./bin/claude-pty-server -socket /tmp/claude-pty.sock
# 或
CLAUDE_PTY_SOCKET=/tmp/claude-pty.sock ./bin/claude-pty-server

# 终止 server
./bin/kill-server
# 或
pkill -f claude-pty-server
```

### 2. CLI 命令

```bash
# 创建新会话
./bin/claude-pty-client create [工作目录]

# 列出所有会话
./bin/claude-pty-client list

# 查看会话状态
./bin/claude-pty-client status <session_id>

# 发送输入
./bin/claude-pty-client input <session_id> "文本"

# 获取输出
./bin/claude-pty-client get <session_id>

# 删除会话
./bin/claude-pty-client delete <session_id>

# 交互式连接
./bin/claude-pty-client connect <session_id>
```

### 3. 使用 curl 直接调用 API

```bash
SOCKET="/run/user/1000/claude-pty.sock"

# 创建会话
curl -s -X POST \
  -H "Content-Type: application/json" \
  -d '{"action":"create","cwd":"/home/zsm/Prj"}' \
  --unix-socket "$SOCKET" http://localhost/

# 列表会话
curl -s --unix-socket "$SOCKET" http://localhost/list

# 获取状态
curl -s -X POST \
  -H "Content-Type: application/json" \
  -d '{"action":"get_status","session_id":"<id>"}' \
  --unix-socket "$SOCKET" http://localhost/

# 设置状态
curl -s -X POST \
  -H "Content-Type: application/json" \
  -d '{"action":"set_status","session_id":"<id>","waiting":true}' \
  --unix-socket "$SOCKET" http://localhost/

# 发送输入
curl -s -X POST \
  -H "Content-Type: application/json" \
  -d '{"action":"input","session_id":"<id>","text":"hello"}' \
  --unix-socket "$SOCKET" http://localhost/

# 获取输出
curl -s -X POST \
  -H "Content-Type: application/json" \
  -d '{"action":"get","session_id":"<id>"}' \
  --unix-socket "$SOCKET" http://localhost/

# 删除会话
curl -s -X POST \
  -H "Content-Type: application/json" \
  -d '{"action":"delete","session_id":"<id>"}' \
  --unix-socket "$SOCKET" http://localhost/
```

### 4. Hook 集成

#### 方法 A: 非侵入式（推荐）

设置环境变量让 server 自动加载 settings：

```bash
# 1. 复制示例配置
cp settings.example.json /path/to/your-claude-pty.json

# 2. 启动 server（指定 settings 文件）
CLAUDE_PTY_SETTINGS=/path/to/your-claude-pty.json ./bin/server
# 或
export CLAUDE_PTY_SETTINGS=/path/to/your-claude-pty.json
./bin/server

# 3. 创建的会话会自动使用该 settings 启动 claude
./bin/client create /path/to/dir
```

#### 方法 B: 全局配置

修改 `~/.claude/settings.json`:

```json
{
  "hooks": {
    "Notification": [
      {
        "matcher": "idle_prompt",
        "hooks": [
          {
            "type": "command",
            "command": "/path/to/claude-pty/cmd/hook/set-status set-status"
          }
        ]
      }
    ]
  }
}
```

Hook 脚本会设置 `waiting_for_input = true` 状态。

## 测试

运行 API 测试:

```bash
./tests/api_test.sh
```

或指定 socket 路径:

```bash
CLAUDE_PTY_SOCKET=/tmp/claude-pty.sock ./tests/api_test.sh
```

## 项目结构

```
claude-pty/
├── cmd/
│   ├── server/main.go      # Server 主程序
│   ├── client/main.go      # CLI 主程序
│   └── hook/set-status    # Hook 脚本
├── internal/
│   ├── server.go          # Unix Socket Server
│   ├── session.go         # 会话管理
│   ├── protocol.go        # 通信协议
│   └── pty_manager.go     # PTY 管理
├── tests/
│   └── api_test.sh       # API 测试脚本
├── bin/
│   ├── server             # 编译后的 server
│   └── client             # 编译后的 client
├── go.mod
└── go.sum
```

## 依赖

- Go 1.18+
- github.com/creack/pty
- github.com/google/uuid

# Claude PTY Server 文档

## 概述

Claude PTY Server 是一个通过 tmux 创建和管理 Claude Code 子进程的服务器和客户端工具。

## 架构

```
┌─────────────┐     Unix Socket      ┌─────────────┐
│   Client    │ ◄──────────────────► │   Server    │
└─────────────┘                      └──────┬──────┘
                                           │
                                      tmux │
                                           ▼
                                    ┌─────────────┐
                                    │ Claude Code │
                                    │  (tmux)     │
                                    └─────────────┘
```

## 核心组件

### 1. Server (cmd/server)

- 长期运行的守护进程
- 通过 Unix Socket 提供 HTTP API
- 管理 Claude Code tmux 会话

### 2. Client (cmd/client)

- 命令行客户端
- 调用 Server API 进行操作

### 3. Hook 脚本 (cmd/hook/set-status)

- Claude Code 钩子脚本
- 通知 Server 会话状态变化

## 状态管理

### 三种状态

| 状态 | 说明 | 触发 Hook |
|------|------|-----------|
| `running` | Claude 正在运行 | UserPromptSubmit |
| `stopped` | Claude 已停止 | Stop |
| `need_permission` | 等待用户授权 | PermissionRequest |

### 状态流程

```
创建会话 ──► stopped ──► UserPromptSubmit ──► running
                              ▲                    │
                              │                    │
                              └────────────────────┘
                              (用户提交输入)

running ──► PermissionRequest ──► need_permission
              ▲                         │
              │                         │
              └─────────────────────────┘
              (需要授权时，如执行命令)
```

## API

### 创建会话

```bash
curl -s -X POST \
  -d '{"action":"create","cwd":"/path/to/dir"}' \
  --unix-socket /tmp/claude-pty.sock http://localhost/
```

### 设置状态

```bash
curl -s -X POST \
  -d '{"action":"set_status","session_id":"<id>","status":"running"}' \
  --unix-socket /tmp/claude-pty.sock http://localhost/
```

### 获取状态

```bash
curl -s -X POST \
  -d '{"action":"get_status","session_id":"<id>"}' \
  --unix-socket /tmp/claude-pty.sock http://localhost/
```

### 列表会话

```bash
curl -s --unix-socket /tmp/claude-pty.sock http://localhost/list
```

### 发送输入

```bash
curl -s -X POST \
  -d '{"action":"input","session_id":"<id>","text":"hello"}' \
  --unix-socket /tmp/claude-pty.sock http://localhost/
```

### 删除会话

```bash
curl -s -X POST \
  -d '{"action":"delete","session_id":"<id>"}' \
  --unix-socket /tmp/claude-pty.sock http://localhost/
```

## 文件结构

```
claude-pty/
├── cmd/
│   ├── server/main.go           # Server 主程序
│   ├── client/main.go           # CLI 主程序
│   └── hook/
│       └── set-status           # Hook 脚本
├── internal/
│   ├── server.go                # Unix Socket Server
│   ├── session.go               # 会话管理 (tmux)
│   └── protocol.go              # 通信协议
├── scripts/
│   ├── build.sh                 # 编译脚本
│   ├── extract_conversation.py  # 提取对话历史
│   └── find_session.py         # 查找会话文件
├── settings.example.json        # 示例配置文件
└── docs/
    └── README.md               # 本文档
```

## 编译

```bash
./scripts/build.sh
```

或手动编译:

```bash
go build -o bin/claude-pty-server ./cmd/server
go build -o bin/claude-pty-client ./cmd/client
```

## 使用方法

### 启动 Server

```bash
./bin/claude-pty-server
```

### CLI 命令

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

# 查看对话历史
./bin/claude-pty-client log <session_id> [limit]
```

## Hook 配置

### settings.example.json

```json
{
  "hooks": {
    "PermissionRequest": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "/path/to/claude-pty/cmd/hook/set-status need_permission"
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "/path/to/claude-pty/cmd/hook/set-status stop"
          }
        ]
      }
    ],
    "UserPromptSubmit": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "/path/to/claude-pty/cmd/hook/set-status running"
          }
        ]
      }
    ]
  }
}
```

### 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| CLAUDE_PTY_SOCKET | Unix Socket 路径 | /tmp/claude-pty.sock |
| CLAUDE_PTY_SETTINGS | settings 文件路径 | (无) |
| CLAUDE_PTY_SESSION_ID | 当前会话 ID | (由 server 设置) |

## 版本历史

### v1.0

- 初始版本
- 支持 tmux 会话管理
- 支持三种状态: running, stopped, need_permission
- Server 退出时自动清理 tmux 会话
- 自动获取 Claude Code 真实 session ID

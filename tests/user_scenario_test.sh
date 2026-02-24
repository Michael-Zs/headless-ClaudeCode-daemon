#!/bin/bash
# Claude PTY Server 用户场景测试
# 模拟用户操作流程: create -> get (模拟 Ctrl+C) -> input

SOCKET_PATH="${CLAUDE_PTY_SOCKET:-/run/user/1000/claude-pty.sock}"

log_info() { echo "[INFO] $1"; }
log_pass() { echo "[PASS] $1"; }
log_fail() { echo "[FAIL] $1"; }

# 测试: create -> get(中断) -> input
test_user_scenario() {
    log_info "测试用户场景: create -> get(中断) -> input"

    # 1. 创建会话
    log_info "1. 创建会话"
    RESPONSE=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d '{"action":"create","cwd":"/home/zsm/Prj"}' \
        --unix-socket "$SOCKET_PATH" \
        http://localhost/)

    if ! echo "$RESPONSE" | grep -q '"success":true'; then
        log_fail "创建失败: $RESPONSE"
        return 1
    fi

    SESSION_ID=$(echo "$RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    log_pass "会话: $SESSION_ID"

    # 2. 获取输出 (模拟用户按 Ctrl+C)
    log_info "2. 获取输出 (然后 Ctrl+C)"
    OUTPUT=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "{\"action\":\"get\",\"session_id\":\"$SESSION_ID\"}" \
        --unix-socket "$SOCKET_PATH" \
        http://localhost/)
    log_pass "获取到输出"

    # 3. 验证会话仍然存在
    log_info "3. 验证会话存在"
    RESPONSE=$(curl -s --unix-socket "$SOCKET_PATH" http://localhost/list)
    if ! echo "$RESPONSE" | grep -q "$SESSION_ID"; then
        log_fail "会话消失了!"
        return 1
    fi
    log_pass "会话仍然存在"

    # 4. 发送输入
    log_info "4. 发送输入"
    RESPONSE=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "{\"action\":\"input\",\"session_id\":\"$SESSION_ID\",\"text\":\"hello\"}" \
        --unix-socket "$SOCKET_PATH" \
        http://localhost/)

    echo "响应: $RESPONSE"

    if ! echo "$RESPONSE" | grep -q '"success":true'; then
        log_fail "发送输入失败"
        return 1
    fi
    log_pass "发送输入成功"

    # 清理
    curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "{\"action\":\"delete\",\"session_id\":\"$SESSION_ID\"}" \
        --unix-socket "$SOCKET_PATH" \
        http://localhost/ > /dev/null

    log_pass "测试通过!"
}

# 检查 server
if [ ! -S "$SOCKET_PATH" ]; then
    log_fail "Server 未运行: $SOCKET_PATH"
    exit 1
fi

test_user_scenario

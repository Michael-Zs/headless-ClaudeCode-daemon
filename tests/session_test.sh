#!/bin/bash
# Claude PTY Server 会话持久化测试

SOCKET_PATH="${CLAUDE_PTY_SOCKET:-/run/user/1000/claude-pty.sock}"

log_info() {
    echo "[INFO] $1"
}

log_pass() {
    echo "[PASS] $1"
}

log_fail() {
    echo "[FAIL] $1"
}

# 测试: 创建会话 -> 获取输出 -> 发送输入 -> 获取状态
test_session_persistence() {
    log_info "测试: 会话持久化"

    # 1. 创建会话
    log_info "1. 创建会话"
    RESPONSE=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d '{"action":"create","cwd":"/home/zsm/Prj"}' \
        --unix-socket "$SOCKET_PATH" \
        http://localhost/)

    echo "创建响应: $RESPONSE"

    if ! echo "$RESPONSE" | grep -q '"success":true'; then
        log_fail "创建会话失败"
        return 1
    fi

    SESSION_ID=$(echo "$RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    log_pass "会话创建成功: $SESSION_ID"

    # 2. 列出会话，确认存在
    log_info "2. 列出会话"
    RESPONSE=$(curl -s --unix-socket "$SOCKET_PATH" http://localhost/list)
    echo "列表响应: $RESPONSE"

    if ! echo "$RESPONSE" | grep -q "$SESSION_ID"; then
        log_fail "会话不在列表中"
        return 1
    fi
    log_pass "会话在列表中"

    # 3. 获取输出
    log_info "3. 获取输出"
    RESPONSE=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "{\"action\":\"get\",\"session_id\":\"$SESSION_ID\"}" \
        --unix-socket "$SOCKET_PATH" \
        http://localhost/)

    echo "获取输出响应: $RESPONSE"

    if ! echo "$RESPONSE" | grep -q '"success":true'; then
        log_fail "获取输出失败"
        return 1
    fi
    log_pass "获取输出成功"

    # 4. 再次列出会话，确认仍然存在
    log_info "4. 再次列出会话"
    RESPONSE=$(curl -s --unix-socket "$SOCKET_PATH" http://localhost/list)
    echo "列表响应: $RESPONSE"

    if ! echo "$RESPONSE" | grep -q "$SESSION_ID"; then
        log_fail "会话在获取输出后消失了"
        return 1
    fi
    log_pass "会话仍然在列表中"

    # 5. 发送输入
    log_info "5. 发送输入"
    RESPONSE=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "{\"action\":\"input\",\"session_id\":\"$SESSION_ID\",\"text\":\"hello\"}" \
        --unix-socket "$SOCKET_PATH" \
        http://localhost/)

    echo "发送输入响应: $RESPONSE"

    if ! echo "$RESPONSE" | grep -q '"success":true'; then
        log_fail "发送输入失败: $RESPONSE"
        return 1
    fi
    log_pass "发送输入成功"

    # 6. 最后列出会话
    log_info "6. 最终列出会话"
    RESPONSE=$(curl -s --unix-socket "$SOCKET_PATH" http://localhost/list)
    echo "列表响应: $RESPONSE"

    if ! echo "$RESPONSE" | grep -q "$SESSION_ID"; then
        log_fail "会话在发送输入后消失了"
        return 1
    fi
    log_pass "会话仍然存在"

    # 7. 清理 - 删除会话
    log_info "7. 删除会话"
    RESPONSE=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "{\"action\":\"delete\",\"session_id\":\"$SESSION_ID\"}" \
        --unix-socket "$SOCKET_PATH" \
        http://localhost/)

    echo "删除响应: $RESPONSE"

    if ! echo "$RESPONSE" | grep -q '"success":true'; then
        log_fail "删除会话失败"
        return 1
    fi
    log_pass "删除会话成功"

    echo ""
    log_pass "所有测试通过!"
}

# 检查 server
if [ ! -S "$SOCKET_PATH" ]; then
    log_fail "Server socket 不存在: $SOCKET_PATH"
    exit 1
fi

test_session_persistence

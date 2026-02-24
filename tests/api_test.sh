#!/bin/bash
# Claude PTY Server API 测试脚本

SOCKET_PATH="${CLAUDE_PTY_SOCKET:-/run/user/1000/claude-pty.sock}"
BASE_URL="http://localhost/"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 测试计数
TESTS_PASSED=0
TESTS_FAILED=0

# 辅助函数
log_info() {
    echo -e "${YELLOW}[INFO]${NC} $1" >&2
}

log_pass() {
    echo -e "${GREEN}[PASS]${NC} $1" >&2
    ((TESTS_PASSED++))
}

log_fail() {
    echo -e "${RED}[FAIL]${NC} $1" >&2
    ((TESTS_FAILED++))
}

# 检查 server 是否运行
check_server() {
    log_info "检查 server 是否运行..."
    if [ -S "$SOCKET_PATH" ]; then
        log_pass "Server socket 存在: $SOCKET_PATH"
        return 0
    else
        log_fail "Server socket 不存在: $SOCKET_PATH"
        return 1
    fi
}

# 测试 1: 创建会话
test_create_session() {
    log_info "测试: 创建会话"

    RESPONSE=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d '{"action":"create","cwd":"/home/zsm/Prj/claude-server"}' \
        --unix-socket "$SOCKET_PATH" \
        http://localhost/)

    if echo "$RESPONSE" | grep -q '"success":true'; then
        SESSION_ID=$(echo "$RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
        log_pass "创建会话成功: $SESSION_ID"
        echo "$SESSION_ID"
    else
        log_fail "创建会话失败: $RESPONSE"
        echo ""
    fi
}

# 测试 2: 列出会话
test_list_sessions() {
    log_info "测试: 列出会话"

    RESPONSE=$(curl -s --unix-socket "$SOCKET_PATH" http://localhost/list)

    if echo "$RESPONSE" | grep -q '"success":true'; then
        log_pass "列出会话成功"
    else
        log_fail "列出会话失败: $RESPONSE"
    fi
}

# 测试 3: 获取会话状态
test_get_status() {
    local session_id=$1
    log_info "测试: 获取会话状态 (session_id: $session_id)"

    RESPONSE=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "{\"action\":\"get_status\",\"session_id\":\"$session_id\"}" \
        --unix-socket "$SOCKET_PATH" \
        http://localhost/)

    if echo "$RESPONSE" | grep -q '"success":true'; then
        log_pass "获取状态成功: $RESPONSE"
    else
        log_fail "获取状态失败: $RESPONSE"
    fi
}

# 测试 4: 设置会话状态
test_set_status() {
    local session_id=$1
    log_info "测试: 设置会话状态"

    RESPONSE=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "{\"action\":\"set_status\",\"session_id\":\"$session_id\",\"waiting\":true}" \
        --unix-socket "$SOCKET_PATH" \
        http://localhost/)

    if echo "$RESPONSE" | grep -q '"success":true'; then
        log_pass "设置状态成功"
    else
        log_fail "设置状态失败: $RESPONSE"
    fi
}

# 测试 5: 验证状态设置正确
test_verify_status() {
    local session_id=$1
    log_info "测试: 验证状态设置"

    RESPONSE=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "{\"action\":\"get_status\",\"session_id\":\"$session_id\"}" \
        --unix-socket "$SOCKET_PATH" \
        http://localhost/)

    if echo "$RESPONSE" | grep -q '"waiting":true'; then
        log_pass "状态验证成功: waiting=true"
    else
        log_fail "状态验证失败: $RESPONSE"
    fi
}

# 测试 6: 发送输入
test_send_input() {
    local session_id=$1
    log_info "测试: 发送输入"

    RESPONSE=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "{\"action\":\"input\",\"session_id\":\"$session_id\",\"text\":\"test input\"}" \
        --unix-socket "$SOCKET_PATH" \
        http://localhost/)

    if echo "$RESPONSE" | grep -q '"success":true'; then
        log_pass "发送输入成功"
    else
        log_fail "发送输入失败: $RESPONSE"
    fi
}

# 测试 7: 获取输出
test_get_output() {
    local session_id=$1
    log_info "测试: 获取输出"

    RESPONSE=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "{\"action\":\"get\",\"session_id\":\"$session_id\"}" \
        --unix-socket "$SOCKET_PATH" \
        http://localhost/)

    if echo "$RESPONSE" | grep -q '"success":true'; then
        log_pass "获取输出成功"
    else
        log_fail "获取输出失败: $RESPONSE"
    fi
}

# 测试 8: 删除会话
test_delete_session() {
    local session_id=$1
    log_info "测试: 删除会话"

    RESPONSE=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "{\"action\":\"delete\",\"session_id\":\"$session_id\"}" \
        --unix-socket "$SOCKET_PATH" \
        http://localhost/)

    if echo "$RESPONSE" | grep -q '"success":true'; then
        log_pass "删除会话成功"
    else
        log_fail "删除会话失败: $RESPONSE"
    fi
}

# 测试 9: 验证会话已删除
test_verify_delete() {
    local session_id=$1
    log_info "测试: 验证会话已删除"

    RESPONSE=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "{\"action\":\"get_status\",\"session_id\":\"$session_id\"}" \
        --unix-socket "$SOCKET_PATH" \
        http://localhost/)

    if echo "$RESPONSE" | grep -q '"success":false'; then
        log_pass "验证删除成功: 会话不存在"
    else
        log_fail "验证删除失败: $RESPONSE"
    fi
}

# 测试 10: 错误处理 - 不存在的会话
test_error_not_found() {
    log_info "测试: 错误处理 - 不存在的会话"

    RESPONSE=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d '{"action":"get_status","session_id":"non-existent-id"}' \
        --unix-socket "$SOCKET_PATH" \
        http://localhost/)

    if echo "$RESPONSE" | grep -q '"success":false'; then
        log_pass "错误处理正确: 返回失败"
    else
        log_fail "错误处理失败: 应返回失败但返回: $RESPONSE"
    fi
}

# 测试 11: 错误处理 - 缺少必要参数
test_error_missing_params() {
    log_info "测试: 错误处理 - 缺少 session_id"

    RESPONSE=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d '{"action":"get_status"}' \
        --unix-socket "$SOCKET_PATH" \
        http://localhost/)

    if echo "$RESPONSE" | grep -q '"error".*"session_id required"'; then
        log_pass "参数验证正确: 返回 session_id required"
    else
        log_fail "参数验证失败: $RESPONSE"
    fi
}

# 测试 12: 错误处理 - 未知 action
test_error_unknown_action() {
    log_info "测试: 错误处理 - 未知 action"

    RESPONSE=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d '{"action":"unknown_action"}' \
        --unix-socket "$SOCKET_PATH" \
        http://localhost/)

    if echo "$RESPONSE" | grep -q '"success":false'; then
        log_pass "未知 action 处理正确"
    else
        log_fail "未知 action 处理失败: $RESPONSE"
    fi
}

# 主测试流程
main() {
    echo "=========================================="
    echo "  Claude PTY Server API 测试"
    echo "  Socket: $SOCKET_PATH"
    echo "=========================================="
    echo ""

    # 检查 server
    if ! check_server; then
        echo ""
        log_fail "Server 未运行，请先启动 server"
        exit 1
    fi
    echo ""

    # 测试列出会话
    test_list_sessions
    echo ""

    # 创建会话并获取 session_id
    SESSION_ID=$(test_create_session)
    echo ""

    if [ -z "$SESSION_ID" ]; then
        log_fail "无法创建测试会话，终止测试"
        exit 1
    fi

    # 继续测试
    test_get_status "$SESSION_ID"
    echo ""

    test_set_status "$SESSION_ID"
    echo ""

    test_verify_status "$SESSION_ID"
    echo ""

    test_send_input "$SESSION_ID"
    echo ""

    test_get_output "$SESSION_ID"
    echo ""

    test_delete_session "$SESSION_ID"
    echo ""

    test_verify_delete "$SESSION_ID"
    echo ""

    # 错误处理测试
    test_error_not_found
    echo ""

    test_error_missing_params
    echo ""

    test_error_unknown_action
    echo ""

    # 输出总结
    echo "=========================================="
    echo "  测试总结"
    echo "=========================================="
    echo -e "${GREEN}通过: $TESTS_PASSED${NC}"
    echo -e "${RED}失败: $TESTS_FAILED${NC}"
    echo ""

    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "${GREEN}所有测试通过!${NC}"
        exit 0
    else
        echo -e "${RED}部分测试失败!${NC}"
        exit 1
    fi
}

main "$@"

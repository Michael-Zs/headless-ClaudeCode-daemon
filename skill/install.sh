#!/bin/bash
# 将 claude-pty skill 安装到各 AI 编程工具的 skills 目录

set -e

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SKILL_SRC="$REPO_ROOT/skill/claude-pty"
SKILL_NAME="claude-pty"
BIN_SERVER="$REPO_ROOT/bin/claude-pty-server"
BIN_CLIENT="$REPO_ROOT/bin/claude-pty-client"

# 确保二进制文件存在，否则先编译
ensure_built() {
    if [ ! -f "$BIN_SERVER" ] || [ ! -f "$BIN_CLIENT" ]; then
        echo "二进制文件不存在，先执行编译..."
        bash "$REPO_ROOT/scripts/build.sh"
    fi
}

# 复制 skill 到指定目录，若父级 config 目录不存在则跳过
install_skill() {
    local dst="$1"
    local label="$2"
    local config_dir="$3"  # 必须存在的 config 根目录

    if [ -n "$config_dir" ] && [ ! -d "$config_dir" ]; then
        echo "  [$label] 未安装，跳过。"
        return
    fi

    if [ -e "$dst" ]; then
        echo "  [$label] 警告: $dst 已存在，备份为 ${dst}.bak"
        mv "$dst" "${dst}.bak"
    fi

    # 复制 skill 目录（-L 展开 symlink，得到独立副本，含 bin/ 内的二进制文件）
    mkdir -p "$(dirname "$dst")"
    cp -rL "$SKILL_SRC" "$dst"

    echo "  [$label] 安装完成: $dst"
}

echo "Installing claude-pty skill..."
echo "  源目录: $SKILL_SRC"
echo ""

ensure_built

# Claude Code
install_skill "$HOME/.claude/skills/$SKILL_NAME" "claude-code" "$HOME/.claude"

# OpenCode（XDG 路径）
OPENCODE_XDG="${XDG_CONFIG_HOME:-$HOME/.config}/opencode"
install_skill "$OPENCODE_XDG/skills/$SKILL_NAME" "opencode(xdg)" "$OPENCODE_XDG"

# OpenCode（~/.opencode 路径）
install_skill "$HOME/.opencode/skills/$SKILL_NAME" "opencode" "$HOME/.opencode"

# OpenClaw
install_skill "$HOME/.openclaw/skills/$SKILL_NAME" "openclaw" "$HOME/.openclaw"

echo ""
echo "安装完成。"

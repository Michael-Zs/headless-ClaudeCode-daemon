#!/bin/bash
# 将 claude-pty skill 安装到各 AI 编程工具的 skills 目录

set -e

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SKILL_SRC="$REPO_ROOT/skill/claude-pty"
SKILL_NAME="claude-pty"

# 安装到指定目录，若父级 config 目录不存在则跳过
install_skill() {
    local dst="$1"
    local label="$2"
    local config_dir="$3"  # 必须存在的 config 根目录

    if [ -n "$config_dir" ] && [ ! -d "$config_dir" ]; then
        echo "  [$label] 未安装，跳过。"
        return
    fi

    if [ -e "$dst" ] || [ -L "$dst" ]; then
        if [ -L "$dst" ] && [ "$(readlink -f "$dst")" = "$SKILL_SRC" ]; then
            echo "  [$label] 已安装，跳过。"
            return
        fi
        echo "  [$label] 警告: $dst 已存在，备份为 ${dst}.bak"
        mv "$dst" "${dst}.bak"
    fi

    mkdir -p "$(dirname "$dst")"
    ln -s "$SKILL_SRC" "$dst"
    echo "  [$label] 安装完成: $dst"
}

echo "Installing claude-pty skill..."
echo "  源目录: $SKILL_SRC"
echo ""

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
echo "注意: 请确保已编译可执行文件："
echo "  cd $REPO_ROOT && bash scripts/build.sh"

---
name: claude-pty
description: Use when you need to run Claude Code in a managed PTY session — creating sessions, sending prompts, reading output, handling permission requests, and cleaning up. Covers the full lifecycle of a claude-pty session.
---

# Claude PTY Session Management

This skill teaches you how to operate Claude Code sessions via the `claude-pty` server using the binaries located in the `bin/` folder inside this skill folder.

**Binaries:** `./bin/client` and `./bin/server` (relative to this skill folder)

---

## Overview

The `client` tool talks to a running `claude-pty-server` over a Unix socket. It lets you:

- Create and list Claude sessions
- Send text input and keystrokes
- Read terminal output
- Monitor session status
- Delete sessions when done

---

## Step 0 — Make sure the server is running

Before using any command, the `claude-pty-server` must be running. If you see this error from any `client` call:

```
Error: do request: Post "http://localhost/": dial unix /tmp/claude-pty.sock: connect: no such file or directory
```

The server is not running. Start it in the background:

```bash
./bin/server &
sleep 1
```

Then retry your command. You only need to do this once — the server stays running until the machine reboots or it is explicitly killed.

---

## Step 1 — Orient yourself with `create` and `list`

Before doing anything, get context about what sessions exist.

```bash
# List all existing sessions
./bin/client list
```

Output columns: `ID`, `CWD` (working directory), `Status`.

```bash
# Create a new session (optionally pass a working directory)
./bin/client create /path/to/workdir
```

This prints the new session ID. Save it — you will use it for every subsequent command.

---

## Step 2 — Check session info with `info`

```bash
./bin/client info <session_id>
```

This shows the session's status, working directory, creation time, and last activity. The key field is **Status**, which tells you what Claude is doing right now.

### Status meanings

| Status | Meaning | What to do |
|---|---|---|
| `running` | Claude is actively processing | **Wait.** Do not send input yet. Poll with `info` or `get` until the status changes. |
| `stopped` | Claude has finished and is idle | Safe to read output with `get`, then send the next prompt with `input`. |
| `need_permission` | Claude is waiting for you to approve or deny a tool-use permission | Read the output with `get` to see what it is asking. If you want to allow it, send the selection keys. |

---

## Step 3 — Read output with `get`

```bash
# Get full terminal output
./bin/client get <session_id>

# Get only the last N lines (useful to avoid huge output)
./bin/client get <session_id> 100
```

Always use a limit when you just want recent context. Use `get` to:

- Understand what Claude has done so far
- See what permission it is requesting (when status is `need_permission`)
- Confirm a prompt was received and is being processed

---

## Step 4 — Send a prompt with `input`

Send the prompt text first, then send `Enter` as a separate call to submit it.

```bash
# Send the prompt text
./bin/client input <session_id> "your prompt here"

# Submit it (press Enter)
./bin/client input <session_id> "Enter"
```

**Important:** Always send these as two separate calls. The first call types the text; the second call submits it.

---

## Step 5 — Handle `need_permission` status

When `info` shows `need_permission`, Claude is showing an interactive permission dialog. Run `get` to read what permission is being requested.

Navigate and confirm using individual key presses — each is a separate `input` call:

```bash
# Move selection up
./bin/client input <session_id> "Up"

# Move selection down
./bin/client input <session_id> "Down"

# Confirm the current selection
./bin/client input <session_id> "Enter"
```

Typical dialog options are "Yes" / "No" / "Always allow". Use `Up`/`Down` to highlight your choice, then `Enter` to confirm.

After confirming, Claude's status should return to `running`, then eventually `stopped`.

---

## Step 6 — Wait for completion

When status is `running`, poll periodically:

```bash
./bin/client info <session_id>
```

Do not send more input while Claude is running. Wait until status becomes `stopped` before proceeding.

---

## Step 7 — Clean up with `delete`

When the task is fully complete, delete the session to free resources:

```bash
./bin/client delete <session_id>
```

---

## Full workflow example

```bash
SKILL_DIR="$(dirname "$0")"   # or hardcode the path to the skill folder
CLIENT="$SKILL_DIR/bin/client"
SERVER="$SKILL_DIR/bin/server"

# 0. Ensure server is running
if ! $CLIENT list >/dev/null 2>&1; then
  echo "Server not running, starting..."
  $SERVER &
  sleep 1
fi

# 1. Create session
SESSION=$($CLIENT create /my/project | grep "Session created:" | awk '{print $3}')
echo "session: $SESSION"
sleep 1

# 2. Send a prompt
$CLIENT input "$SESSION" "Summarize the README file"
sleep 1
$CLIENT input "$SESSION" "Enter"
sleep 1

# 3. Wait until done (poll every 2s)
while true; do
  STATUS=$($CLIENT info "$SESSION" | grep "^Status:" | awk '{print $2}')
  sleep 1
  case "$STATUS" in
    stopped) break ;;
    running) sleep 2 ;;
    need_permission)
      $CLIENT get "$SESSION" 30   # see what it wants
      sleep 1
      $CLIENT input "$SESSION" "Enter"  # approve default selection
      sleep 1
      ;;
  esac
done

# 4. Read the result
$CLIENT get "$SESSION" 50
sleep 1

# 5. Clean up
$CLIENT delete "$SESSION"
```

---

## Quick reference

| Command | Usage | Purpose |
|---|---|---|
| `list` | `./bin/client list` | See all sessions and their statuses |
| `create` | `./bin/client create [cwd]` | Start a new Claude session |
| `info` | `./bin/client info <id>` | Check status and metadata |
| `get` | `./bin/client get <id> [limit]` | Read terminal output |
| `input` | `./bin/client input <id> <text>` | Send a keystroke or text |
| `delete` | `./bin/client delete <id>` | Remove session when finished |

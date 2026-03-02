---
name: claude-pty
description: Use when you need to run Claude Code in a managed PTY session — creating sessions, sending prompts, reading output, handling permission requests, and cleaning up. Covers the full lifecycle of a claude-pty session.
---

# Orchestrating Claude Sub-agents

This skill lets you (the orchestrator) spawn Claude Code sub-agents, read their output, and **decide what to do next based on what you learn**. You are not running a fixed script — you are an intelligent controller that observes, reasons, and adapts.

**Binaries:** `./bin/client` and `./bin/server` (relative to this skill folder)

---

## The orchestration loop

```
┌─────────────────────────────────────────────┐
│                                             │
│   spawn / send prompt                       │
│         │                                   │
│         ▼                                   │
│     [sub-agent works]                       │
│         │                                   │
│         ▼                                   │
│   read output with get                      │
│         │                                   │
│         ▼                                   │
│   YOU decide what to do next  ◄─────────┐  │
│         │                               │  │
│    ┌────┴──────────────────────┐        │  │
│    │                           │        │  │
│    ▼                           ▼        │  │
│  report to user      send follow-up ───┘  │
│  wait for next       spawn more agents     │
│  instruction         escalate to user      │
└─────────────────────────────────────────────┘
```

**Sessions are never deleted by you.** Only delete a session when the user explicitly asks for it.

The intelligence lives in **you reading the output and deciding**. The commands are just the mechanism.

---

## Step 0 — Ensure server is running

```bash
./bin/client list
```

If that fails with a socket error, start the server:

```bash
nohup ./bin/server > /tmp/claude-pty-server.log 2>&1 &
sleep 1
```

---

## Step 1 — Spawn a sub-agent

```bash
SESSION=$(./bin/client create /path/to/workdir | grep "Session created:" | awk '{print $3}')
```

Wait ~1 second before sending the first prompt — Claude needs a moment to initialize.

---

## Step 2 — Delegate a task

```bash
./bin/client input "$SESSION" "Your task here"
./bin/client input "$SESSION" "Enter"
```

**Tip:** Ask sub-agents to produce structured output (JSON, a summary line, a yes/no answer) so you can parse and reason about their response more easily.

```bash
# Good: structured output makes your decision easy
./bin/client input "$SESSION" "Check if all tests pass. Reply with exactly: PASS or FAIL, then explain why."
./bin/client input "$SESSION" "Enter"

# Good: ask for a specific artifact
./bin/client input "$SESSION" "List every TODO in the codebase as JSON: [{\"file\": ..., \"line\": ..., \"text\": ...}]"
./bin/client input "$SESSION" "Enter"
```

---

## Step 3 — Wait for the sub-agent

Poll until it stops:

```bash
while true; do
  STATUS=$(./bin/client status "$SESSION" | awk '{print $NF}')
  case "$STATUS" in
    stopped)        break ;;
    running)        sleep 3 ;;
    need_permission)
      ./bin/client get "$SESSION" ".1"        # see what it's asking
      ./bin/client input "$SESSION" "Enter"   # approve default
      sleep 1 ;;
  esac
done
```

---

## Step 4 — Read the output and decide

This is the most important step. **Read carefully, then reason.**

```bash
RESULT=$(./bin/client get "$SESSION" ">1")
echo "$RESULT"
```

**`get` limit formats:**

| Limit | Returns |
|---|---|
| (none) | Full terminal output |
| `100` | Last 100 lines |
| `>1` | Last complete user turn (your prompt + sub-agent's full response) |
| `>2` | Last 2 user turns — useful to see recent context |
| `.1` | Last output block — useful during `need_permission` |

After reading, ask yourself:

- **Did the sub-agent succeed?** → Report results to the user and wait for their next instruction.
- **Did it fail or get stuck?** → Send a corrective follow-up, or spawn a fresh agent on a different approach.
- **Is the result incomplete?** → Ask it to continue or clarify.
- **Did it surface new information?** → Adjust your overall plan. Spawn additional agents if needed.
- **Is there an unexpected situation?** → Escalate to the user. Don't guess.

You do not need to follow a fixed plan. If the sub-agent's output changes what makes sense to do, change course.

---

## Step 5 — Act on your decision

### Continue with the same sub-agent

The sub-agent remembers all prior context. Use it for follow-up tasks in the same project:

```bash
./bin/client input "$SESSION" "The test at line 42 is failing — fix it"
./bin/client input "$SESSION" "Enter"
# ... wait and read again ...
```

### Spawn a new sub-agent for a different concern

```bash
SESSION_B=$(./bin/client create /other/project | grep "Session created:" | awk '{print $3}')
sleep 1
./bin/client input "$SESSION_B" "..."
./bin/client input "$SESSION_B" "Enter"
```

### Report and wait

When the task is done, report the result to the user and stop. **Do not delete the session.** The user may want to ask a follow-up, request changes, or continue from where the sub-agent left off.

---

## Step 6 — Iterate

Repeat steps 3–5 as many times as needed. A single task might require many read-decide-act cycles before it's complete.

---

## Example: adaptive orchestration

```bash
CLIENT="./bin/client"

# Spawn sub-agent
SESSION=$($CLIENT create /my/project | grep "Session created:" | awk '{print $3}')
sleep 1

# --- Round 1: investigate ---
$CLIENT input "$SESSION" "Run the test suite and report: PASS or FAIL on the first line, then list any failures."
$CLIENT input "$SESSION" "Enter"

while true; do
  STATUS=$($CLIENT status "$SESSION" | awk '{print $NF}')
  [ "$STATUS" = "stopped" ] && break
  [ "$STATUS" = "need_permission" ] && $CLIENT input "$SESSION" "Enter" && sleep 1 && continue
  sleep 3
done

RESULT=$($CLIENT get "$SESSION" ">1")

# --- Decision: what did we learn? ---
if echo "$RESULT" | grep -q "^PASS"; then
  echo "All tests pass."
  # Report to user. Session stays alive for follow-up.

elif echo "$RESULT" | grep -q "^FAIL"; then
  echo "Tests failing. Asking sub-agent to fix."

  # --- Round 2: fix based on what we read ---
  $CLIENT input "$SESSION" "Fix the failing tests you listed. Do not change any test expectations."
  $CLIENT input "$SESSION" "Enter"

  while true; do
    STATUS=$($CLIENT status "$SESSION" | awk '{print $NF}')
    [ "$STATUS" = "stopped" ] && break
    [ "$STATUS" = "need_permission" ] && $CLIENT input "$SESSION" "Enter" && sleep 1 && continue
    sleep 3
  done

  RESULT2=$($CLIENT get "$SESSION" ">1")

  if echo "$RESULT2" | grep -q "^PASS"; then
    echo "Fixed. All tests now pass."
    # Report to user. Session stays alive for follow-up.
  else
    echo "Could not fix automatically. Escalating to user."
    echo "$RESULT2"
    # Stop and let the user decide what to do next.
  fi

else
  echo "Unexpected output — could not parse. Full result:"
  echo "$RESULT"
  # Report to user. Session stays alive for investigation.
fi
```

---

## Parallel sub-agents with independent decisions

```bash
# Spawn agents for independent tasks
SESSION_A=$($CLIENT create /project | grep "Session created:" | awk '{print $3}')
SESSION_B=$($CLIENT create /project | grep "Session created:" | awk '{print $3}')
sleep 1

$CLIENT input "$SESSION_A" "Audit security vulnerabilities. Output JSON: [{\"severity\": ..., \"location\": ..., \"description\": ...}]"
$CLIENT input "$SESSION_A" "Enter"
$CLIENT input "$SESSION_B" "Profile performance bottlenecks. Output JSON: [{\"hotspot\": ..., \"impact\": ...}]"
$CLIENT input "$SESSION_B" "Enter"

# Wait for both
for SID in "$SESSION_A" "$SESSION_B"; do
  while true; do
    STATUS=$($CLIENT status "$SID" | awk '{print $NF}')
    [ "$STATUS" = "stopped" ] && break
    [ "$STATUS" = "need_permission" ] && $CLIENT input "$SID" "Enter" && sleep 1 && continue
    sleep 3
  done
done

# Read both results and reason about priority
SECURITY=$($CLIENT get "$SESSION_A" ">1")
PERF=$($CLIENT get "$SESSION_B" ">1")

# Report findings to the user. Both sessions stay alive for follow-up questions.
```

---

## Quick reference

| Command | Usage | Purpose |
|---|---|---|
| `list` | `./bin/client list` | List all active sub-agent sessions |
| `create` | `./bin/client create [cwd]` | Spawn a new sub-agent |
| `status` | `./bin/client status <id>` | Poll state: `running` / `stopped` / `need_permission` |
| `get` | `./bin/client get <id> [limit]` | **Read output to inform your decision** (`>N` turns, `.N` blocks, line count) |
| `input` | `./bin/client input <id> <text>` | Send prompt text or keystroke (`Enter`, `Up`, `Down`) |
| `info` | `./bin/client info <id>` | Full metadata (CWD, timestamps, Claude session ID) |
| `log` | `./bin/client log <id> [limit]` | Structured conversation history (User / Claude / Tool) |
| `delete` | `./bin/client delete <id>` | **Only when the user explicitly asks** |
| `connect` | `./bin/client connect <id>` | Interactive terminal access (Ctrl+Q to exit) |

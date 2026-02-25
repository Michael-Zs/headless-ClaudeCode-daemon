#!/bin/bash

SKILL_DIR="$(dirname "$0")" # or hardcode the path to the skill folder
CLIENT="$SKILL_DIR/bin/client"

# 1. Create session
SESSION=$($CLIENT create /home/zsm/Prj/find_my_director/ | grep "Session created:" | awk '{print $3}')
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
    $CLIENT get "$SESSION" 30 # see what it wants
    sleep 1
    $CLIENT input "$SESSION" "Enter" # approve default selection
    sleep 1
    ;;
  esac
done

# 4. Read the result
$CLIENT get "$SESSION" 50
sleep 1

# 5. Clean up
$CLIENT delete "$SESSION"

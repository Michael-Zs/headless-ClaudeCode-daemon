#!/usr/bin/env python3
"""Extract conversation from Claude Code session jsonl file."""
import json
import sys

def extract_conversation(jsonl_file, limit=None):
    """Extract conversation and return list of messages."""
    messages = []

    with open(jsonl_file) as f:
        lines = f.readlines()

    if limit:
        lines = lines[-limit:]

    for line in lines:
        try:
            d = json.loads(line)
            msg_type = d.get('type', '')

            if msg_type == 'user':
                msg = d.get('message', {})
                content = msg.get('content', '')

                # Content can be string or list
                if isinstance(content, str) and content:
                    messages.append(('User', content))
                elif isinstance(content, list):
                    for c in content:
                        if c.get('type') == 'text':
                            text = c.get('text', '')
                            if text:
                                messages.append(('User', text))
                        elif c.get('type') == 'tool_result':
                            text = c.get('content', '')
                            if text:
                                messages.append(('User', text))

            elif msg_type == 'assistant':
                msg = d.get('message', {})
                content = msg.get('content', [])

                for c in content:
                    if c.get('type') == 'text':
                        text = c.get('text', '')
                        if text:
                            messages.append(('Assistant', text))
                    elif c.get('type') == 'tool_use':
                        tool_name = c.get('name', '')
                        tool_input = c.get('input', '')
                        messages.append(('Assistant Tool', f"{tool_name}: {tool_input}"))

        except Exception as e:
            pass

    return messages

def print_conversation(messages):
    """Print messages in a readable format."""
    for msg_type, content in messages:
        # Truncate long content
        display = content[:300] + '...' if len(content) > 300 else content
        print(f"\n[{msg_type}]:")
        print(display)

def main():
    if len(sys.argv) < 2:
        print("Usage: extract_conversation.py <jsonl_file> [limit]")
        sys.exit(1)

    jsonl_file = sys.argv[1]
    limit = int(sys.argv[2]) if len(sys.argv) > 2 else None

    messages = extract_conversation(jsonl_file, limit)
    print_conversation(messages)

if __name__ == '__main__':
    main()

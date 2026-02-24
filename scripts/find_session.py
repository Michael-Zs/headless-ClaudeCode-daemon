#!/usr/bin/env python3
"""
Find the relationship between workspace folder, session ID, and jsonl file.

Claude Code stores sessions in:
~/.claude/projects/{slug}/{session_id}.jsonl

Where slug is derived from the working directory path.
"""

import os
import json
import sys
from pathlib import Path

def get_project_slug(cwd):
    """Generate project slug from working directory."""
    # Remove leading/trailing slashes and normalize
    cwd = os.path.normpath(cwd)

    # Get the home directory
    home = os.path.expanduser("~")

    # Try to find the relative path from home
    if cwd.startswith(home):
        rel_path = cwd[len(home):].lstrip("/")
    else:
        rel_path = cwd

    # Convert to slug format (replace / with -)
    slug = rel_path.replace("/", "-")

    if not slug:
        slug = "home"

    return slug

def find_jsonl_file(cwd, session_id):
    """Find the jsonl file for a given working directory and session ID."""
    home = os.path.expanduser("~")
    projects_dir = os.path.join(home, ".claude", "projects")

    # Get the project slug
    slug = get_project_slug(cwd)

    # Try different possible locations
    possible_paths = [
        # Direct match
        os.path.join(projects_dir, f"{slug}", f"{session_id}.jsonl"),
        # With -home-zsm prefix for home directory
        os.path.join(projects_dir, f"-home-zsm-{slug}", f"{session_id}.jsonl"),
    ]

    for path in possible_paths:
        if os.path.exists(path):
            return path

    # Fallback: search all projects directories
    if os.path.exists(projects_dir):
        for entry in os.listdir(projects_dir):
            entry_path = os.path.join(projects_dir, entry)
            if os.path.isdir(entry_path):
                jsonl_path = os.path.join(entry_path, f"{session_id}.jsonl")
                if os.path.exists(jsonl_path):
                    return jsonl_path

    return None

def list_sessions_for_workspace(cwd):
    """List all sessions for a given workspace."""
    home = os.path.expanduser("~")
    projects_dir = os.path.join(home, ".claude", "projects")

    slug = get_project_slug(cwd)

    sessions = []

    # Search in project directories
    if os.path.exists(projects_dir):
        for entry in os.listdir(projects_dir):
            entry_path = os.path.join(projects_dir, entry)
            if os.path.isdir(entry_path):
                # Check if this directory matches our slug
                if slug in entry or entry in slug:
                    for file in os.listdir(entry_path):
                        if file.endswith('.jsonl'):
                            sessions.append({
                                'path': os.path.join(entry_path, file),
                                'session_id': file.replace('.jsonl', ''),
                                'project': entry
                            })

    return sessions

def get_session_info(jsonl_path):
    """Get basic info about a session from jsonl file."""
    info = {
        'cwd': None,
        'version': None,
        'message_count': 0,
        'first_timestamp': None,
        'last_timestamp': None,
    }

    if not os.path.exists(jsonl_path):
        return info

    try:
        with open(jsonl_path) as f:
            lines = f.readlines()
            info['message_count'] = len(lines)

            if lines:
                first = json.loads(lines[0])
                last = json.loads(lines[-1])
                info['cwd'] = first.get('cwd')
                info['version'] = first.get('version')
                info['first_timestamp'] = first.get('timestamp')
                info['last_timestamp'] = last.get('timestamp')
    except Exception as e:
        print(f"Error reading {jsonl_path}: {e}")

    return info

def main():
    # Test with known values
    test_cwd = "/home/zsm/Prj/claude-server"
    test_session = "22d0d43f-a089-4712-be62-4d27d49932f4"

    print("=== Claude Code Session File Locator ===\n")

    # Test 1: Get slug for a path
    print(f"1. Slug for '{test_cwd}':")
    print(f"   -> {get_project_slug(test_cwd)}\n")

    # Test 2: Find jsonl file
    print(f"2. Finding jsonl for session '{test_session}':")
    jsonl_path = find_jsonl_file(test_cwd, test_session)
    if jsonl_path:
        print(f"   -> {jsonl_path}")
        print(f"   -> exists: {os.path.exists(jsonl_path)}")
    else:
        print("   -> Not found")
    print()

    # Test 3: List all sessions for workspace
    print(f"3. All sessions for '{test_cwd}':")
    sessions = list_sessions_for_workspace(test_cwd)
    for s in sessions[:5]:
        info = get_session_info(s['path'])
        print(f"   - {s['session_id']}")
        print(f"     project: {s['project']}")
        print(f"     cwd: {info['cwd']}")
        print(f"     messages: {info['message_count']}")
    print()

    # Test 4: Interactive search
    if len(sys.argv) > 2:
        cwd = sys.argv[1]
        session_id = sys.argv[2]
        print(f"4. Looking up: cwd={cwd}, session={session_id}")
        jsonl_path = find_jsonl_file(cwd, session_id)
        if jsonl_path:
            print(f"   -> {jsonl_path}")
            info = get_session_info(jsonl_path)
            print(f"   -> cwd: {info['cwd']}")
            print(f"   -> version: {info['version']}")
            print(f"   -> messages: {info['message_count']}")

if __name__ == '__main__':
    main()

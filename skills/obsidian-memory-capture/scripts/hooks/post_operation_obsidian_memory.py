#!/usr/bin/env python3
"""Codex notify hook: capture assistant turn outputs into Obsidian inbox."""

from __future__ import annotations

import datetime as dt
import json
import shutil
import subprocess
import sys
from pathlib import Path

HOOK_DIR = Path(__file__).resolve().parent
LOG_FILE = HOOK_DIR / "post_operation_obsidian_memory.log"
LOCK_FILE = HOOK_DIR / "post_operation_obsidian_memory.lock"
CAPTURE_SCRIPT = (
    Path.home()
    / ".codex"
    / "skills"
    / "obsidian-memory-capture"
    / "scripts"
    / "capture_session_answers.py"
)


def log(message: str) -> None:
    timestamp = dt.datetime.now(dt.timezone.utc).replace(microsecond=0).isoformat()
    LOG_FILE.parent.mkdir(parents=True, exist_ok=True)
    with LOG_FILE.open("a", encoding="utf-8") as handle:
        handle.write(f"[{timestamp}] {message}\n")


def run_capture(thread_id: str) -> subprocess.CompletedProcess[str]:
    base_cmd = [
        "/usr/bin/env",
        "python3",
        str(CAPTURE_SCRIPT),
        "--session-id",
        thread_id,
    ]

    lockf = shutil.which("lockf")
    if lockf:
        cmd = [lockf, "-k", str(LOCK_FILE), *base_cmd]
    else:
        cmd = base_cmd

    return subprocess.run(cmd, capture_output=True, text=True)


def main() -> int:
    payload_raw = sys.argv[1] if len(sys.argv) > 1 else ""
    if not payload_raw:
        payload_raw = sys.stdin.read().strip()
    if not payload_raw:
        return 0

    try:
        payload = json.loads(payload_raw)
    except json.JSONDecodeError:
        log("skip: invalid JSON payload")
        return 0

    event_type = str(payload.get("type", ""))
    thread_id = str(payload.get("thread-id", ""))
    last_message = str(payload.get("last-assistant-message", "")).strip()

    log(f"event={event_type or 'unknown'} thread={thread_id or 'unknown'}")

    if event_type != "agent-turn-complete":
        log("skip: unsupported event type")
        return 0
    if not thread_id or not last_message:
        log("skip: missing thread id or assistant message")
        return 0
    if not CAPTURE_SCRIPT.exists():
        log(f"skip: missing capture script {CAPTURE_SCRIPT}")
        return 0

    log(f"capture start thread={thread_id}")
    result = run_capture(thread_id)

    if result.stdout.strip():
        for line in result.stdout.strip().splitlines():
            log(line)
    if result.stderr.strip():
        for line in result.stderr.strip().splitlines():
            log(f"stderr: {line}")

    log(f"capture end thread={thread_id} rc={result.returncode}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

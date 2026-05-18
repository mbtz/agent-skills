#!/usr/bin/env python3
"""User feedback file helpers.

Commands:
- check_user_feedback
- init_user_feedback
- update_user_feedback
"""
from __future__ import annotations

import argparse
import os
import re
import sys
import time
from datetime import datetime

DEFAULT_NOTE = (
    "Do not delete the timestamp above; it records the last time this file was read by an agent."
)


def _default_path() -> str:
    return os.path.join(os.getcwd(), "USER_FEEDBACK.md")


def _parse_timestamp(line: str) -> float:
    line = line.strip()
    if not line:
        raise ValueError("First line is empty")

    if re.fullmatch(r"\d+(?:\.\d+)?", line):
        return float(line)

    m = re.search(
        r"(\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?)",
        line,
    )
    if m:
        token = m.group(1).replace(" ", "T")
        if token.endswith("Z"):
            token = token[:-1] + "+00:00"
        if re.search(r"[+-]\d{4}$", token):
            token = token[:-5] + token[-5:-2] + ":" + token[-2:]
        dt = datetime.fromisoformat(token)
        if dt.tzinfo is None:
            return time.mktime(dt.timetuple()) + dt.microsecond / 1_000_000
        return dt.timestamp()

    m = re.search(r"(\d{4}-\d{2}-\d{2})", line)
    if m:
        dt = datetime.fromisoformat(m.group(1))
        return time.mktime(dt.timetuple())

    raise ValueError(f"Unrecognized timestamp: {line!r}")


def _now_timestamp() -> str:
    return datetime.now().strftime("%Y-%m-%dT%H:%M:%S")


def check_user_feedback(path: str, threshold: float) -> int:
    try:
        with open(path, "r", encoding="utf-8", errors="replace") as f:
            first_line = f.readline().strip()
    except FileNotFoundError:
        sys.stderr.write(f"File not found: {path}\n")
        return 2

    try:
        ts = _parse_timestamp(first_line)
    except Exception as exc:
        sys.stderr.write(f"Error: {exc}\n")
        return 2

    mtime = os.path.getmtime(path)
    is_newer = mtime > (ts + threshold)
    print("true" if is_newer else "false", end="")
    return 0 if is_newer else 1


def init_user_feedback(path: str, force: bool) -> int:
    if os.path.exists(path) and not force:
        sys.stderr.write(f"File already exists: {path}\n")
        return 2

    os.makedirs(os.path.dirname(path) or ".", exist_ok=True)
    with open(path, "w", encoding="utf-8") as f:
        f.write(_now_timestamp() + "\n")
        f.write(DEFAULT_NOTE + "\n")
    return 0


def update_user_feedback(path: str) -> int:
    try:
        with open(path, "r", encoding="utf-8", errors="replace") as f:
            lines = f.readlines()
    except FileNotFoundError:
        sys.stderr.write(f"File not found: {path}\n")
        return 2

    timestamp = _now_timestamp()
    if not lines:
        lines = [timestamp + "\n"]
    else:
        newline = "\n" if lines[0].endswith("\n") else ""
        lines[0] = timestamp + newline

    with open(path, "w", encoding="utf-8") as f:
        f.writelines(lines)
    return 0


def _build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="User feedback file helpers")
    sub = parser.add_subparsers(dest="command", required=True)

    check = sub.add_parser("check_user_feedback")
    check.add_argument("--path", default=_default_path())
    check.add_argument("--threshold", type=float, default=1.0)

    init = sub.add_parser("init_user_feedback")
    init.add_argument("--path", default=_default_path())
    init.add_argument("--force", action="store_true")

    update = sub.add_parser("update_user_feedback")
    update.add_argument("--path", default=_default_path())

    return parser


def main() -> int:
    parser = _build_parser()
    args = parser.parse_args()

    if args.command == "check_user_feedback":
        return check_user_feedback(args.path, args.threshold)
    if args.command == "init_user_feedback":
        return init_user_feedback(args.path, args.force)
    if args.command == "update_user_feedback":
        return update_user_feedback(args.path)

    parser.error("Unknown command")
    return 2


if __name__ == "__main__":
    raise SystemExit(main())

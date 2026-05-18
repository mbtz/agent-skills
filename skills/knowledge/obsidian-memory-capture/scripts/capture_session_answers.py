#!/usr/bin/env python3
"""Append new Codex assistant answers from session JSONL into Obsidian raw inbox."""

from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import json
import os
import re
import socket
from pathlib import Path
from typing import Iterable


CAPTURE_FORMAT = "codex-session-jsonl-v1"
UNKNOWN_MACHINE_TAG = "unknown-machine"


def parse_args() -> argparse.Namespace:
    script_dir = Path(__file__).resolve().parent
    parser = argparse.ArgumentParser(
        description="Capture new assistant answers from Codex session JSONL."
    )
    parser.add_argument(
        "--config",
        type=Path,
        default=script_dir / "config.json",
        help="Config file path.",
    )
    parser.add_argument(
        "--session-id",
        type=str,
        default=os.environ.get("CODEX_THREAD_ID", ""),
        help="Session id (default: CODEX_THREAD_ID).",
    )
    parser.add_argument(
        "--session-file",
        type=Path,
        help="Explicit rollout JSONL path.",
    )
    parser.add_argument(
        "--project",
        action="append",
        default=[],
        help="Project slug (repeatable).",
    )
    parser.add_argument(
        "--include-phase",
        action="append",
        default=[],
        help="Extra assistant phases to include.",
    )
    parser.add_argument(
        "--machine-tag",
        type=str,
        default=os.environ.get("AGENT_MACHINE_TAG", ""),
        help="Machine tag to stamp raw capture metadata (default: AGENT_MACHINE_TAG or hostname).",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Show what would be appended without writing files.",
    )
    return parser.parse_args()


def load_config(path: Path) -> dict:
    if not path.exists():
        raise SystemExit(f"Config not found: {path}")
    payload = json.loads(path.read_text(encoding="utf-8"))
    required = ["vault_root", "inbox_root", "sessions_root", "source_agent"]
    missing = [key for key in required if key not in payload]
    if missing:
        raise SystemExit(f"Config missing required keys: {', '.join(missing)}")
    return payload


def expand_path(value: str) -> Path:
    return Path(os.path.expandvars(os.path.expanduser(value))).resolve()


def find_session_file(session_root: Path, session_id: str) -> Path:
    if not session_id:
        raise SystemExit("--session-id is required when --session-file is omitted.")
    matches = sorted(
        session_root.glob(f"**/rollout-*-{session_id}.jsonl"),
        key=lambda p: p.stat().st_mtime,
    )
    if not matches:
        raise SystemExit(f"No session file found for session id: {session_id}")
    return matches[-1]


def parse_session_path_date(session_file: Path) -> dt.date:
    parts = session_file.parts
    for idx in range(len(parts) - 2):
        year, month, day = parts[idx : idx + 3]
        if re.match(r"^\d{4}$", year) and re.match(r"^\d{2}$", month) and re.match(r"^\d{2}$", day):
            return dt.date(int(year), int(month), int(day))
    return dt.date.today()


def load_state(path: Path) -> dict:
    if not path.exists():
        return {}
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError:
        return {}


def save_state(path: Path, payload: dict) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(payload, indent=2, sort_keys=True) + "\n", encoding="utf-8")


def extract_text(content: list[dict]) -> str:
    chunks: list[str] = []
    for item in content:
        if item.get("type") != "output_text":
            continue
        text = item.get("text", "")
        if isinstance(text, str) and text.strip():
            chunks.append(text.rstrip())
    return "\n\n".join(chunks).strip()


def event_id(timestamp: str, phase: str, text: str) -> str:
    payload = f"{timestamp}|{phase}|{text}".encode("utf-8")
    return hashlib.sha1(payload).hexdigest()[:16]


def iter_new_events(
    *,
    lines: list[str],
    start_line: int,
    include_phases: set[str],
) -> Iterable[dict]:
    for line_number in range(start_line + 1, len(lines) + 1):
        raw = lines[line_number - 1]
        if not raw.strip():
            continue
        try:
            item = json.loads(raw)
        except json.JSONDecodeError:
            continue

        if item.get("type") != "response_item":
            continue
        payload = item.get("payload", {})
        if payload.get("type") != "message" or payload.get("role") != "assistant":
            continue

        phase = str(payload.get("phase", ""))
        if include_phases and phase not in include_phases:
            continue

        text = extract_text(payload.get("content", []))
        if not text:
            continue

        timestamp = str(item.get("timestamp", ""))
        eid = event_id(timestamp, phase, text)
        yield {
            "event_id": eid,
            "timestamp": timestamp,
            "phase": phase or "unknown",
            "text": text,
            "line_number": line_number,
        }


def split_frontmatter(text: str) -> tuple[dict, str]:
    if not text.startswith("---\n"):
        return {}, text
    marker = "\n---\n"
    end = text.find(marker, 4)
    if end == -1:
        return {}, text

    block = text[4:end]
    body = text[end + len(marker) :]
    metadata: dict[str, object] = {}
    current_key: str | None = None

    for raw in block.splitlines():
        line = raw.rstrip()
        if not line:
            continue
        key_match = re.match(r"^([A-Za-z0-9_-]+):\s*(.*)$", line)
        if key_match:
            key = key_match.group(1)
            value = key_match.group(2).strip()
            if value == "":
                metadata[key] = []
                current_key = key
            elif value.startswith("[") and value.endswith("]"):
                inner = value[1:-1].strip()
                metadata[key] = [
                    item.strip().strip("'\"")
                    for item in inner.split(",")
                    if item.strip()
                ]
                current_key = None
            else:
                metadata[key] = value.strip("'\"")
                current_key = None
            continue

        list_match = re.match(r"^\s*-\s+(.*)$", line)
        if list_match and current_key:
            value = list_match.group(1).strip().strip("'\"")
            existing = metadata.setdefault(current_key, [])
            if isinstance(existing, list):
                existing.append(value)

    return metadata, body


def normalize_slug(value: str) -> str:
    return re.sub(r"[^a-zA-Z0-9_-]+", "-", value.strip().lower()).strip("-")


def normalize_machine_tag(value: str) -> str:
    normalized = normalize_slug(value)
    if normalized:
        return normalized
    return UNKNOWN_MACHINE_TAG


def resolve_machine_tag(argument_value: str, config: dict) -> str:
    if argument_value.strip():
        return normalize_machine_tag(argument_value)

    config_value = config.get("machine_tag", "")
    if isinstance(config_value, str) and config_value.strip():
        return normalize_machine_tag(config_value)

    for env_name in ("AGENT_MACHINE_TAG", "HOSTNAME", "HOST"):
        candidate = os.environ.get(env_name, "").strip()
        if candidate:
            return normalize_machine_tag(candidate)

    return normalize_machine_tag(socket.gethostname())


def normalize_projects(projects: list[str], aliases: dict[str, str]) -> list[str]:
    output: set[str] = set()
    for project in projects:
        slug = normalize_slug(project)
        if not slug:
            continue
        if slug in aliases:
            slug = normalize_slug(aliases[slug])
        if slug:
            output.add(slug)
    return sorted(output)


def render_frontmatter(metadata: dict) -> str:
    preferred = [
        "source_agent",
        "machine_tag",
        "run_id",
        "status",
        "projects",
        "session_file",
        "capture_format",
        "captured_phases",
    ]
    keys = [key for key in preferred if key in metadata]
    keys.extend(sorted(key for key in metadata if key not in set(keys)))

    lines = ["---"]
    for key in keys:
        value = metadata[key]
        if isinstance(value, list):
            lines.append(f"{key}:")
            for item in value:
                lines.append(f"  - {item}")
        else:
            lines.append(f"{key}: {value}")
    lines.extend(["---", ""])
    return "\n".join(lines)


def ensure_note_and_frontmatter(
    *,
    note_path: Path,
    capture_date: dt.date,
    source_agent: str,
    machine_tag: str,
    session_id: str,
    projects: list[str],
    include_phases: list[str],
    session_file: Path,
    project_aliases: dict[str, str],
    dry_run: bool,
) -> None:
    note_path.parent.mkdir(parents=True, exist_ok=True)
    if note_path.exists():
        existing = note_path.read_text(encoding="utf-8")
        metadata, body = split_frontmatter(existing)
    else:
        metadata = {}
        body = f"# Raw Capture - {capture_date.isoformat()}\n\n"

    existing_projects = metadata.get("projects", [])
    if isinstance(existing_projects, str):
        existing_projects = [existing_projects]
    if not isinstance(existing_projects, list):
        existing_projects = []

    merged_projects = normalize_projects(
        [*existing_projects, *projects],
        project_aliases,
    )

    existing_phases = metadata.get("captured_phases", [])
    if isinstance(existing_phases, str):
        existing_phases = [existing_phases]
    if not isinstance(existing_phases, list):
        existing_phases = []

    merged_phases = sorted({*existing_phases, *include_phases})

    metadata.update(
        {
            "source_agent": metadata.get("source_agent", source_agent),
            "machine_tag": metadata.get("machine_tag", machine_tag),
            "run_id": metadata.get("run_id", session_id),
            "status": metadata.get("status", "raw"),
            "projects": merged_projects,
            "session_file": str(session_file),
            "capture_format": CAPTURE_FORMAT,
            "captured_phases": merged_phases,
        }
    )

    if dry_run:
        return

    content = render_frontmatter(metadata) + body.lstrip("\n")
    note_path.write_text(content, encoding="utf-8")


def load_logged_event_ids(note_path: Path) -> set[str]:
    if not note_path.exists():
        return set()
    text = note_path.read_text(encoding="utf-8")
    return set(re.findall(r"^event_id:\s*([a-f0-9]{16,40})\s*$", text, flags=re.MULTILINE))


def append_events(note_path: Path, events: list[dict], dry_run: bool) -> None:
    if not events or dry_run:
        return
    with note_path.open("a", encoding="utf-8") as handle:
        for event in events:
            handle.write(f"## {event['timestamp']} [{event['phase']}]\n")
            handle.write(f"event_id: {event['event_id']}\n")
            handle.write(f"session_line: {event['line_number']}\n\n")
            handle.write(event["text"].rstrip() + "\n\n")


def scan_label(start_line: int, total_lines: int) -> str:
    if total_lines == start_line:
        return "no new lines"
    return f"lines {start_line + 1}-{total_lines}"


def main() -> None:
    args = parse_args()
    config = load_config(args.config)
    machine_tag = resolve_machine_tag(args.machine_tag, config)

    vault_root = expand_path(str(config["vault_root"]))
    inbox_root = Path(str(config.get("inbox_root", "00-Inbox")))
    sessions_root = expand_path(str(config.get("sessions_root", "~/.codex/sessions")))
    project_aliases = {
        normalize_slug(key): normalize_slug(value)
        for key, value in dict(config.get("project_aliases", {})).items()
    }

    include_phases = list(config.get("include_phases", ["final_answer"]))
    include_phases.extend(args.include_phase)
    include_phase_set = {phase for phase in include_phases if phase}

    projects = [value for value in args.project if value]
    if not projects:
        projects = [str(p) for p in config.get("default_projects", ["obsidian-memory"]) if str(p)]

    session_id = args.session_id.strip()
    session_file = args.session_file.resolve() if args.session_file else find_session_file(sessions_root, session_id)
    session_file_match = re.search(r"([0-9a-f]{8}-[0-9a-f-]{27,})\.jsonl$", session_file.name)
    if args.session_file and session_file_match:
        session_id = session_file_match.group(1)
    elif not session_id and session_file_match:
        session_id = session_file_match.group(1)
    if not session_id:
        raise SystemExit("Unable to resolve session id.")

    capture_date = parse_session_path_date(session_file)
    note_dir = vault_root / inbox_root / f"{capture_date:%Y}" / f"{capture_date:%m}" / f"{capture_date:%d}" / session_id
    note_filename = str(config.get("note_filename", "raw-capture.md"))
    note_path = note_dir / note_filename
    state_filename = str(config.get("state_filename", ".capture-state.json"))
    state_path = note_dir / state_filename

    lines = session_file.read_text(encoding="utf-8").splitlines()

    state = load_state(state_path)
    same_session_file = state.get("session_file") == str(session_file)
    start_line = int(state.get("last_line_processed", 0)) if same_session_file else 0
    if start_line < 0 or start_line > len(lines):
        start_line = 0

    candidate_events = list(
        iter_new_events(lines=lines, start_line=start_line, include_phases=include_phase_set)
    )
    if not candidate_events and not note_path.exists():
        print(
            f"No captureable entries in {session_file.name} "
            f"(scanned {scan_label(start_line, len(lines))})."
        )
        return

    ensure_note_and_frontmatter(
        note_path=note_path,
        capture_date=capture_date,
        source_agent=str(config.get("source_agent", "codex")),
        machine_tag=machine_tag,
        session_id=session_id,
        projects=projects,
        include_phases=sorted(include_phase_set),
        session_file=session_file,
        project_aliases=project_aliases,
        dry_run=args.dry_run,
    )

    logged_ids = load_logged_event_ids(note_path)
    new_events = [event for event in candidate_events if event["event_id"] not in logged_ids]

    if not new_events:
        print(
            f"No new entries in {session_file.name} "
            f"(scanned {scan_label(start_line, len(lines))} of {session_file.name})."
        )
        return

    append_events(note_path, new_events, args.dry_run)

    state_payload = {
        "session_file": str(session_file),
        "last_line_processed": len(lines),
        "last_event_count": len(new_events),
        "updated_at": dt.datetime.now(dt.timezone.utc).replace(microsecond=0).isoformat(),
    }
    if not args.dry_run:
        save_state(state_path, state_payload)

    rel = note_path.relative_to(vault_root)
    action = "Would append" if args.dry_run else "Appended"
    print(
        f"{action} {len(new_events)} new entries to {rel} "
        f"(scanned {scan_label(start_line, len(lines))} of {session_file.name})."
    )


if __name__ == "__main__":
    main()

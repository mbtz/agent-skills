---
name: obsidian-memory-capture
description: Capture Codex assistant answers from session JSONL into Obsidian raw inbox notes for manager distillation. Use when maintaining long-term memory artifacts in `00-Inbox` and append only new, unlogged answers.
---

# Obsidian Memory Capture

Use this skill to ingest Codex session outputs into raw inbox notes with deduplicated, append-only capture.

## Bundle layout

- Script: `scripts/capture_session_answers.py`
- Config: `scripts/config.json`
- Install wizard: `INSTALL_WIZARD.json`

When installed via `askill`, the installer prompts for machine-specific paths and writes `scripts/config.json`. You can still edit the file manually later.

## Standard usage

From this skill directory:

```bash
python3 scripts/capture_session_answers.py
```

Optional explicit session:

```bash
python3 scripts/capture_session_answers.py --session-id "$CODEX_THREAD_ID"
```

Dry run preview:

```bash
python3 scripts/capture_session_answers.py --dry-run
```

## Post-Operation Hook (recommended)

Enable Codex post-operation capture in `~/.codex/config.toml`:

```toml
notify = ["/Users/vh/.codex/hooks/post_operation_obsidian_memory.sh"]
notifications = ["agent-turn-complete"]
```

Hook scripts:

- `/Users/vh/.codex/hooks/post_operation_obsidian_memory.sh`
- `/Users/vh/.codex/hooks/post_operation_obsidian_memory.py`

## Rules

- `00-Inbox` is raw capture only; do not normalize here.
- Default project slug is `obsidian-memory`.
- The script ingests assistant messages from Codex session JSONL and appends only unseen entries.
- The script must be called at the end of each answered turn when this memory workflow is active.
- In interactive use, prefer the post-operation hook so capture happens automatically after each answer.

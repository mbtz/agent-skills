#!/usr/bin/env python3
"""Install and enable the Obsidian memory post-operation hook for Codex."""

from __future__ import annotations

import argparse
import json
import re
import shutil
from pathlib import Path


HOOK_SCRIPT_NAMES = [
    "post_operation_obsidian_memory.sh",
    "post_operation_obsidian_memory.py",
]


def parse_args() -> argparse.Namespace:
    script_dir = Path(__file__).resolve().parent
    parser = argparse.ArgumentParser(
        description="Install post-operation hook files and update ~/.codex/config.toml",
    )
    parser.add_argument(
        "--skill-root",
        type=Path,
        default=script_dir.parent,
        help="Installed skill root path.",
    )
    parser.add_argument(
        "--hooks-dir",
        type=Path,
        default=Path.home() / ".codex" / "hooks",
        help="Codex hooks directory.",
    )
    parser.add_argument(
        "--codex-config",
        type=Path,
        default=Path.home() / ".codex" / "config.toml",
        help="Path to Codex config.toml.",
    )
    return parser.parse_args()


def parse_inline_string_array(inner: str) -> list[str]:
    values: list[str] = []
    for raw in inner.split(","):
        item = raw.strip()
        if not item:
            continue
        if (item.startswith('"') and item.endswith('"')) or (
            item.startswith("'") and item.endswith("'")
        ):
            item = item[1:-1]
        values.append(item)
    return values


def render_inline_string_array(values: list[str]) -> str:
    encoded = ", ".join(json.dumps(value) for value in values)
    return f"[{encoded}]"


def upsert_inline_array_value(content: str, key: str, value: str) -> str:
    pattern = re.compile(rf"(?m)^\s*{re.escape(key)}\s*=\s*\[(.*?)\]\s*$")
    match = pattern.search(content)

    if match:
        existing = parse_inline_string_array(match.group(1))
        if value not in existing:
            existing.append(value)
        replacement = f"{key} = {render_inline_string_array(existing)}"
        return content[: match.start()] + replacement + content[match.end() :]

    suffix = ""
    if content and not content.endswith("\n"):
        suffix = "\n"
    return content + suffix + f"{key} = {render_inline_string_array([value])}\n"


def install_hook_files(skill_root: Path, hooks_dir: Path) -> Path:
    hooks_source_dir = skill_root / "scripts" / "hooks"
    if not hooks_source_dir.exists():
        raise SystemExit(f"Hook source directory not found: {hooks_source_dir}")

    hooks_dir.mkdir(parents=True, exist_ok=True)
    for filename in HOOK_SCRIPT_NAMES:
        source = hooks_source_dir / filename
        if not source.exists():
            raise SystemExit(f"Hook source file not found: {source}")
        destination = hooks_dir / filename
        shutil.copy2(source, destination)
        destination.chmod(0o755)

    return (hooks_dir / HOOK_SCRIPT_NAMES[0]).resolve()


def update_codex_config(config_path: Path, hook_script_path: Path) -> None:
    config_path.parent.mkdir(parents=True, exist_ok=True)
    if config_path.exists():
        content = config_path.read_text(encoding="utf-8")
    else:
        content = ""

    content = upsert_inline_array_value(content, "notify", str(hook_script_path))
    content = upsert_inline_array_value(content, "notifications", "agent-turn-complete")

    config_path.write_text(content, encoding="utf-8")


def main() -> int:
    args = parse_args()
    skill_root = args.skill_root.expanduser().resolve()
    hooks_dir = args.hooks_dir.expanduser().resolve()
    codex_config = args.codex_config.expanduser().resolve()

    hook_script = install_hook_files(skill_root, hooks_dir)
    update_codex_config(codex_config, hook_script)

    print(f"Installed hooks into: {hooks_dir}")
    print(f"Updated Codex config: {codex_config}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

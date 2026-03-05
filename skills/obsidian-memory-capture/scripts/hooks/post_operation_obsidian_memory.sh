#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
/usr/bin/env python3 "$SCRIPT_DIR/post_operation_obsidian_memory.py" "$@"

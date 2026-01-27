# Agent Skills

This repo collects standardized `SKILL.md`-based skills and ships a small CLI
that installs them into supported harnesses.

## Installation

### Homebrew Homebrew (tap + install)

```bash
brew tap mbtz/agent-skills
brew install askill
```

### From Source

```bash
git clone https://github.com/mbtz/agent-skills.git

go build ./cmd/askill
```

## Upgrade

```bash
brew update
brew upgrade askill
```

## Usage

Interactive (default):

```bash
askill
```

Running without options opens the interactive TUI installer.

TUI controls:

- `j`/`k` or arrows to move up/down
- `a` to toggle all
- `space` to select/deselect
- `enter` to confirm
- `q` to cancel & quit

Flags (for non-interactive installation of all skills available):

- `-r`, `--repo`: path to skills repo (defaults to current directory)
- `-p`, `--project`: project path for project-local installs
- `-c`, `--copy`: copy files instead of symlink
- `-s`, `--symlink`: force symlink mode
- `-f`, `--from-config`: install all skills using config defaults
---
- `-v`, `--version`: print version and exit
- `-h`, `--help`: show help

### Config

```bash
askill config
askill config --init
askill config --edit
```

Config file path: `~/Library/Application Support/askill/config.toml`

Example:

```toml
skill-repo-path = "https://github.com/mbtz/agent-skills"
project-choice = "skip"
project-path = ""
install-mode = "symlink"
```

Release (updates version, tags, and Homebrew formula):

```bash
make release
```

### Supported harness paths

- Codex CLI: `~/.codex/skills/`
- Claude Code:
  - Global: `~/.claude/skills/`
  - Project: `/path/to/project/.claude/skills/`
- Cursor:
  - Global: `~/.cursor/skills/` (if present)
  - Project: `/path/to/project/.cursor/skills/`

The CLI detects available targets under `$HOME`, and uses `--project` for
project-local installs.

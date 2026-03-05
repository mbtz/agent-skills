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
- `-v`, `--version`: print version and exit
- `-h`, `--help`: show help

### Skill install wizard

Skills that require machine-specific setup can include `INSTALL_WIZARD.json` in the skill root.
When present, `askill` prompts for those values during installation and writes them into the installed skill files.

Supported wizard behavior:

- Writes JSON files
- Updates string values by dot-path keys (for example `vault_root` or `nested.path`)

Wizard example:

```json
{
  "version": 1,
  "title": "Skill setup",
  "files": [
    {
      "path": "scripts/config.json",
      "format": "json",
      "fields": [
        {
          "key": "vault_root",
          "prompt": "Path to your vault root",
          "default": "~/path/to/vault",
          "required": true
        }
      ]
    }
  ]
}
```

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
install-mode = "copy"
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

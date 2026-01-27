## Agent Skills

This repo collects standardized `SKILL.md`-based skills and ships a small CLI
that installs them into supported harnesses.

### CLI: askill

Homebrew (tap + install):

```bash
brew tap mbtz/agent-skills
brew install askill
```

Upgrade:

```bash
brew update
brew upgrade askill
```

Interactive (default):

```bash
askill
```

Running without options opens the interactive TUI installer.

TUI controls:

- `j`/`k` or arrows to move
- `space` to select/deselect
- `a` to toggle all
- `enter` to confirm
- `q` to quit

From source:

```bash
go build ./cmd/askill
```

Run from this repo (symlink mode by default):

```bash
./askill --project /path/to/your/project
```

Copy mode:

```bash
./askill --copy --project /path/to/your/project
```

Flags:

- `-r`, `--repo`: path to skills repo (defaults to current directory)
- `-p`, `--project`: project path for project-local installs
- `-c`, `--copy`: copy files instead of symlink
- `-s`, `--symlink`: force symlink mode
- `-v`, `--version`: print version and exit
- `-h`, `--help`: show help

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

## Agent Skills

This repo collects standardized `SKILL.md`-based skills and ships a small CLI
that installs them into supported harnesses.

### CLI: askill

Build:

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

Install all skills without the skills prompt:

```bash
./askill --all --project /path/to/your/project
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

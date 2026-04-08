---
name: tk-issue-tracking
description: "Use when creating, viewing, starting, or closing tickets via tk, or when managing the .tickets/ directory alongside code changes."
metadata:
  short-description: Ticket workflow via tk
---

# Ticket Tracking with tk

Use `tk` for issue tracking in this repo. Tickets live in `.tickets/` and must be committed alongside related code changes.

## Commands

| Command | Description |
|---------|-------------|
| `tk ready` | List tickets ready for work |
| `tk show <id>` | View ticket details |
| `tk start <id>` | Mark ticket as in_progress |
| `tk close <id>` | Close a completed ticket |
| `tk create "Title" -t <type> -p <priority> -d "Description"` | Create a new ticket |

**Types**: `bug`, `feature`, `task`
**Priority**: `0` (critical) through `4` (low)

## Typical Workflow

```bash
# 1. See what's ready
tk ready
# ID  Title                    Type     Priority
# 12  Fix auth token refresh   bug      1
# 15  Add dark mode toggle     feature  2

# 2. Pick and start a ticket
tk start 12

# 3. Implement the fix
git add src/auth.ts
git commit -m "fix: refresh auth token before expiry"

# 4. Close the ticket (commit .tickets/ with the code)
tk close 12
git add .tickets/
git commit -m "chore: close ticket 12"
git push
```

## Key Rules

- Always `tk start <id>` before working on a ticket.
- Commit `.tickets/` changes together with (or immediately after) related code changes so history stays linked.
- Check `tk ready` at the start of each session to pick up where you left off.

Reference: [tk-quickref.md](references/tk-quickref.md)

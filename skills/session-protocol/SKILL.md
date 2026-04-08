---
name: session-protocol
description: "Use when ending a work session, handing off to another agent, or ensuring all changes are committed and pushed before stopping."
metadata:
  short-description: End-of-session checklist
---

# Session Protocol

Run this checklist before ending any session to ensure no work is lost and the repo is in a clean state for the next agent or session.

## End-of-Session Checklist

1. **Check status**: `git status` — review all modified, staged, and untracked files.
2. **Review changes**: `git diff` — verify your changes are correct before staging.
3. **Handle untracked files**: decide whether to stage, `.gitignore`, or leave each untracked file.
4. **Stage changes**: `git add <files>` — stage only the files related to your task.
5. **Commit**: `git commit -m "descriptive message"` — use a conventional commit message.
6. **Push**: `git push` — push to the remote branch.

## Handling Unexpected State

- **Untracked files you don't recognize**: check `git log` and ask before deleting — they may be another agent's in-progress work.
- **Merge conflicts**: resolve conflicts, test, then commit the resolution.
- **Dirty working tree with no clear task**: stash changes with `git stash -m "session-end: uncommitted work"` and note it for the next session.

## Example

```bash
git status
# On branch feat/add-retry
# Changes not staged for commit:
#   modified:   src/handler.ts
# Untracked files:
#   src/handler.test.ts

git diff src/handler.ts        # Review the diff
git add src/handler.ts src/handler.test.ts
git commit -m "feat: add retry logic with tests"
git push
```

## Validation

After pushing, confirm:

- `git status` shows a clean working tree.
- `git log -1` shows your commit at HEAD.
- The remote branch is up to date (`git push` exits with "Everything up-to-date" or succeeds).

Reference: [end-session-checklist.md](references/end-session-checklist.md)

---
name: workflow-pattern
description: "Use when starting a new task, updating task status, or following team workflow conventions for issue tracking and code delivery."
metadata:
  short-description: Team workflow guidance
---

# Workflow Pattern

Follow the team workflow pattern for every task from start to finish. This ensures consistent status tracking and clean delivery.

## Workflow Steps

1. **Pick a task** — select from the ready queue using the repo's task system (`sv ready`, `tk ready`, or equivalent).
2. **Start work** — mark the task as `in_progress` before writing code.
3. **Implement** — make the code changes, committing incrementally with descriptive messages.
4. **Update status** — close the task when implementation is complete (`sv close <id>`, `tk close <id>`).
5. **Push** — push your branch and open a PR if needed.

## Task Metadata

When creating or updating tasks:

- **Title**: use a descriptive, imperative title (e.g., "Add retry logic to webhook handler").
- **Priority**: set `0` (critical) through `4` (low) based on impact.
- **Type**: choose `bug`, `feature`, or `task`.

## Example

```bash
# 1. Check what's ready
tk ready

# 2. Start the task
tk start 42

# 3. Implement and commit
git add src/handler.ts
git commit -m "feat: add retry logic to webhook handler"

# 4. Close the task
tk close 42

# 5. Push
git push
```

## Validation

After completing a task, verify:

- `git status` shows a clean working tree.
- The task status is `closed` in the task system.
- All related code changes are committed and pushed.

---
name: user-feedback
description: "Use when checking for new user feedback, triaging feedback into tasks, or initializing the USER_FEEDBACK.md file."
---

# User Feedback Protocol

Monitor and process `USER_FEEDBACK.md` to keep actionable tasks in sync with user input. This skill covers checking for new feedback, creating tasks, and updating the read timestamp.

## File Format

`USER_FEEDBACK.md` lives in the workspace root with this structure:

- **Line 1**: last-read timestamp (`YYYY-MM-DDTHH:MM:SS`).
- **Line 2**: note — "Do not delete the timestamp above; it records the last time this file was read by an agent."
- **Lines 3+**: free-form user feedback.

## Commands

Run the bundled script from the workspace root:

```bash
# Check if new feedback exists (exit 0 = new, 1 = no change, 2 = file missing)
python3 ~/.codex/skills/user-feedback/scripts/user_feedback.py check_user_feedback

# Initialize the feedback file (use --force to overwrite)
python3 ~/.codex/skills/user-feedback/scripts/user_feedback.py init_user_feedback

# Update the timestamp after processing
python3 ~/.codex/skills/user-feedback/scripts/user_feedback.py update_user_feedback
```

Use `--path <filepath>` on any command to override the default location.

## Workflow

1. **Check** — run `check_user_feedback`.
   - Exit code `2` with "File not found" → run `init_user_feedback`, then stop.
   - Exit code `1` → no new feedback, stop.
   - Exit code `0` → new feedback exists, continue.

2. **Process** — read lines 3+ and create one task per actionable item:
   - If `.tasks/` exists → use the `sv` task workflow.
   - If `.tickets/` exists → use the `tk` ticket workflow.
   - If neither exists → check `AGENTS.md` for instructions or ask the user.
   - Check existing tasks for duplicates before creating new ones.

3. **Update** — run `update_user_feedback` to set the timestamp and mark feedback as read.

## Example

```bash
# Step 1: Check for new feedback
python3 ~/.codex/skills/user-feedback/scripts/user_feedback.py check_user_feedback
# Output: true (exit 0)

# Step 2: Read and triage feedback, then create tasks
tk create "Fix login timeout on mobile" -t bug -p 1 -d "User reports 30s timeout on iOS Safari"

# Step 3: Mark as read
python3 ~/.codex/skills/user-feedback/scripts/user_feedback.py update_user_feedback
```

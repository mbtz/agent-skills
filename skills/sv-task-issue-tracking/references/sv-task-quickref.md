Purpose
  Task tracking for this repo. Tasks stored in:
  .tasks/tasks.jsonl (tracked)
  .git/sv/tasks.jsonl (shared across worktrees)

Quickstart
  sv task new "Ship CLI help"
  sv task list --status open
  sv task start 01HZ...
  sv task close 01HZ...

Commands
  sv task                           Open task TUI
  sv task new "<title>" [--status] [--priority P0-P4] [--body]
  sv task list [--status] [--priority] [--workspace] [--actor] [--updated-since] [--limit]
  sv task ready [--priority] [--workspace] [--actor] [--updated-since] [--limit]
  sv task show <id>
  sv task start <id>
  sv task status <id> <status>
  sv task priority <id> <P0-P4>
  sv task edit <id> [--title] [--body]
  sv task close <id> [--status]
  sv task delete <id>
  sv task comment <id> "<text>"
  sv task parent set <child> <parent>
  sv task parent clear <child>
  sv task block <blocker> <blocked>
  sv task unblock <blocker> <blocked>
  sv task relate <left> <right> --desc "<text>"
  sv task unrelate <left> <right>
  sv task relations <id>
  sv task sync
  sv task compact [--older-than] [--max-log-mb] [--dry-run]
  sv task prefix [<prefix>]

Notes
  list/ready sorted: status -> priority -> readiness -> updated_at -> id
  readiness: default_status and not blocked
  Use --json for machine output; use --events <path> with --json.
---
name: sv-workflow-pattern
description: Follow the team workflow expectations while working.
metadata:
  short-description: Team workflow guidance
---

Follow the team workflow pattern while working:

- Update status as you work (open → in_progress → closed).
- Keep Epics open until all child tasks are completed and closed.
- If an Epic is closed but have open child tasks, set the Epic's status to open
- Use descriptive titles and descriptiuons and use appropriate priority,
  status and parents/children when creating new tasks.
- Implement the tasks from highest to lowest priotity (P0 → P4) based on
  the furthest leaf task in the children tree.

To examplify how to properly select the right tasks, given a task tree as
follows:
```
.
├── Example task A (P3) - Single standalone task with priotity P3
├── EPIC: Example epic A (P1)
|   ├── Example child task AA (P2)
│   └── Example child task AB (P0)
│       ├── Example leaf task ABA (P1)
│       └── Example leaf task ABB (P0)
└── EPIC: Example Epic B (P0)
    ├── Example child task BA (P2)
    │   ├── Example leaf task BAA (P1)
    │   └── Example leaf task BAB (P0)
    └── Example child task BB (P2)
        ├── Example leaf task BBA (P1)
        └── Example leaf task BBB (P3)
```

The order would be:
- Example leaf task BAB (P0)
- Example leaf task BAA (P1)
- Example child task BA (P2)
- Example leaf task BBA (P1)
- Example leaf task BBB (P3)
- Example child task BB (P2)
- EPIC: Example Epic B (P0)
- Example leaf task ABB (P0)
- Example leaf task ABA (P1)
- Example child task AA (P2)
- EPIC: Example epic A (P1)
- Example task A (P3)

So in short, taking the highest priotity task and doing the highest priority
sub tasks with a deep first strategy.

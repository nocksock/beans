---
# beans-lw67
title: archive completed
status: completed
type: task
priority: normal
created_at: 2026-10-16T14:59:56Z
updated_at: 2026-10-16T15:05:00Z
---


archive completed todos
suggested key: y

in list view: archive all completed
in detail view: archive currently viewed task, prompt if not completed. (confirm with y, so yy will always archive)

## Work Log
2026-10-16 15:30: Implemented y archive in list/detail views with confirm modal (branch feat/tui-archive, worktree .worktrees/beans-lw67)
2026-10-16 15:31: Added TestArchive table tests; go test ./... passes
2026-10-16 15:45: Switched archive key from y to Y

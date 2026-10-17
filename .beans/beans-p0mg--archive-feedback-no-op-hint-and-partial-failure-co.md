---
# beans-p0mg
title: 'Archive feedback: no-op hint and partial-failure count'
status: todo
type: task
created_at: 2026-10-17T13:05:36Z
updated_at: 2026-10-17T13:05:36Z
---

Found in review of beans-ugfx.
- TUI: Y on an already-archived bean silently does nothing; show a hint.
- CLI: `beans archive` stops at the first UpdateBean error without saying how many beans were already archived (text and JSON output).
## Code review 2026-10-17 (beans-9pxj)

Extend: TUI bulk status/type/priority/tags/parent/blocking handlers (tui.go:296-483) also `continue` on error with no feedback; tui.go:509 archive returns silently. Land via applyToBeans helper (beans-u7nb).

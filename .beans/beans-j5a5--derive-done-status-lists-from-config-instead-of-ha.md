---
# beans-j5a5
title: Derive done-status lists from config instead of hardcoding names
status: todo
type: task
created_at: 2026-10-17T13:05:36Z
updated_at: 2026-10-17T13:05:36Z
---

Found in review of beans-ugfx. `hiddenByDefaultStatuses` (cmd/list.go) and `ui.RenderStatus`/`RenderStatusText` (internal/ui/styles.go) hardcode "completed", "scrapped", "archived". Derive them from `cfg.IsArchiveStatus` (statuses with Archive: true) so adding a done-like status needs one edit.
## Code review 2026-10-17 (beans-9pxj)

Still valid: hiddenByDefaultStatuses (cmd/list.go:46), ui/styles.go:175-194 hardcode; derive via cfg.IsArchiveStatus. Also createmodal.go:19 hardcodes 'draft'/'normal'.

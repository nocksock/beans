---
# beans-9p4x
title: TUI Y bulk-archive ignores tag filter and drill-down root
status: todo
type: bug
created_at: 2026-10-17T13:05:36Z
updated_at: 2026-10-17T13:05:36Z
---

Found in review of beans-ugfx. In the TUI list, Y collects archivable beans with a nil filter (internal/tui/tui.go, archiveCompletedMsg), so it archives every completed/scrapped bean in the project even when filtered by tag or drilled into a parent. Help text says 'Archive (all completed in list)'. Decide: scope to visible beans, or fix the help text.
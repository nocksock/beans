---
# beans-9pnc
title: Consolidate ~/projects/beans main commits into feat/launcher-result
status: completed
type: task
priority: normal
created_at: 2026-10-16T07:03:56Z
updated_at: 2026-10-16T07:07:02Z
---

Cherry-pick the 10 unpushed commits from ~/projects/beans main (tag editor, q-while-typing fix, ~ expansion, launcher beans-path fix) onto feat/launcher-result. Skip beans-data directory moves.
## Work Log
2026-10-16 09:07: Fetched ~/projects/beans main as home/main; cherry-picked 1603dde, 4432736, 371d3c1, 3a5caae -> 6c5719a, 1cf0641, ebdc7c2, 405db0d
2026-10-16 09:07: Resolved conflicts: kept HEAD launcher/TUI code (already uses ConfigDir), kept ConfigPath() + tests, merged q-quit fix into canQuitWithQ() incl. viewSessionDialog
2026-10-16 09:07: Skipped beans-data moves (3f5db13, 0b06a0d) and chore commits; go vet + go test ./... pass

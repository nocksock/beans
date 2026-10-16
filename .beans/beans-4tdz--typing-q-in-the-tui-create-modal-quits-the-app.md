---
# beans-4tdz
title: Typing q in the TUI create modal quits the app
status: completed
type: bug
created_at: 2026-10-16T07:07:02Z
updated_at: 2026-10-16T07:07:02Z
---

The global 'q' key handler in internal/tui/tui.go quit the app while typing in text inputs. Fixed by canQuitWithQ() helper (cherry-picked from ~/projects/beans main, commit ebdc7c2).
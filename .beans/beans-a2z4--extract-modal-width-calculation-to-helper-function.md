---
# beans-a2z4
title: modalSize helper for remaining inline modal size math
status: todo
type: task
priority: normal
tags:
    - technical-debt
    - refactoring
created_at: 2025-12-15T12:48:49Z
updated_at: 2026-10-17T14:02:53Z
parent: beans-gl8t
blocking:
    - beans-5c6x
---

Pattern `max(40, min(60, m.width*50/100))` appears 6+ times across modal files.

## Create helper
```go
func calculateModalWidth(screenWidth, percentWidth, minWidth, maxWidth int) int {
    return max(minWidth, min(maxWidth, screenWidth*percentWidth/100))
}
```

## Files affected
- internal/tui/launcherpicker.go (multiple locations)
- internal/tui/nolaunchersmulti.go
- internal/tui/modal.go

Add helper to modal.go and update all call sites.
## Code review 2026-10-17 (beans-9pxj)

Reshaped (R11): calculateModalWidth exists (modal.go:11). Add `modalSize(w,h,pct,maxW,maxH)` and replace ~12 inline copies (status/type/priority pickers, archiveconfirm, configurelaunchersdialog:171,241, help, parent/blocking pickers, launcherpicker:90,127 height). Detail ctor/resize duplicate layout; only resize clamps vpHeight.

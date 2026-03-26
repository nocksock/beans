---
# beans-a2z4
title: Extract modal width calculation to helper function
status: completed
type: task
priority: normal
tags:
    - technical-debt
    - refactoring
created_at: 2025-12-15T12:48:49Z
updated_at: 2025-12-15T17:55:18Z
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
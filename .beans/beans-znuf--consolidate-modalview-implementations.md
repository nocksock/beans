---
# beans-znuf
title: Consolidate ModalView implementations
status: todo
type: task
priority: normal
tags:
    - technical-debt
    - refactoring
created_at: 2025-12-15T12:48:57Z
updated_at: 2025-12-15T12:48:57Z
parent: beans-gl8t
---

Exact same 4-line `ModalView()` method duplicated 4 times across modal types.

## Files affected
- internal/tui/launcherpicker.go (3 models)
- internal/tui/nolaunchersmulti.go (1 model)

## Solution
Create helper function in modal.go or use interface with default implementation.

```go
func renderModalOverlay(view string, bgView string, fullWidth, fullHeight int) string {
    return overlayModal(bgView, view, fullWidth, fullHeight)
}
```
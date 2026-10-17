---
# beans-znuf
title: Consolidate ModalView implementations
status: archived
type: task
priority: normal
tags:
    - technical-debt
    - refactoring
created_at: 2025-12-15T12:48:57Z
updated_at: 2026-10-17T14:16:52Z
parent: beans-gl8t
blocking:
    - beans-mszj
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
## Code review 2026-10-17 (beans-9pxj)

Scrapped: folded into beans-603f (modal interface removes the 15 identical ModalView wrappers).

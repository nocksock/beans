---
# beans-5c6x
title: Shared message modal for info/error dialogs
status: todo
type: task
priority: normal
size: sm
tags:
    - technical-debt
    - refactoring
created_at: 2025-12-15T12:48:53Z
updated_at: 2026-10-17T14:10:28Z
parent: beans-gl8t
blocking:
    - beans-mszj
---

Two "no launchers" modals share 80% of the same rendering logic with only slight variations (title, message).

## Create shared renderer
```go
type errorModalConfig struct {
    Title       string
    Message     string
    Width       int
    Height      int
    TitleColor  lipgloss.Color
    BorderColor lipgloss.Color
}

func renderErrorModal(cfg errorModalConfig) string
```

## Files affected
- internal/tui/launcherpicker.go (noLaunchersModel.View)
- internal/tui/nolaunchersmulti.go (noLaunchersMultiModel.View)
- internal/tui/modal.go (add shared function)
## Code review 2026-10-17 (beans-9pxj)

Reshaped (R2): one `messageModal{title, msg, color}` replacing 4 copies: noLaunchersModel, noLaunchersMultiModel (nolaunchersmulti.go), launcherErrorModel (launcherpicker.go:220-264), configWriteErrorModel (configurelaunchersdialog.go:225-267). Absorbs beans-mszj (collapse the 4 view states, tui.go:38-42) and beans-x3um remainder. Fixes wrong 'Launcher Failed' title for archive/edit/create errors (tui.go:534,573,622). Use theme colors instead of #ff0000 / #555.

## Sequencing (review beans-9pxj)

Step 4 of 5: 5c6x → beans-u7nb → beans-r7wv → beans-603f.

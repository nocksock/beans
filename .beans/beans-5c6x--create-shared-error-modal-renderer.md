---
# beans-5c6x
title: Create shared error modal renderer
status: todo
type: task
priority: normal
tags:
    - technical-debt
    - refactoring
created_at: 2025-12-15T12:48:53Z
updated_at: 2025-12-15T12:50:36Z
parent: beans-gl8t
blocking:
    - beans-znuf
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
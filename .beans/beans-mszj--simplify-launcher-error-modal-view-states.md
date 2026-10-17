---
# beans-mszj
title: Simplify launcher error modal view states
status: archived
type: task
priority: low
tags:
    - enhancement
    - architecture
created_at: 2025-12-15T12:49:54Z
updated_at: 2026-10-17T14:16:52Z
parent: beans-gl8t
---

Currently 4 separate view states for launcher errors:
- viewNoLaunchers
- viewNoLaunchersMulti
- viewLauncherError
- viewConfigWriteError

## Proposal
Create single `viewModal` state with configurable genericModal/infoModal model.

## Benefits
- Less boilerplate
- Easier to add new modals
- Cleaner state machine

## Trade-offs
- Less type safety (modal content becomes dynamic)

## Implementation
```go
type genericModal struct {
    title       string
    message     string
    icon        string
    color       lipgloss.Color
    dismissKey  string
    onDismiss   func() tea.Msg
}
```

Replace all 4 states with single `viewModal` state, store modal config in app struct.
## Code review 2026-10-17 (beans-9pxj)

Scrapped: merged into beans-5c6x (shared message modal collapses these view states).

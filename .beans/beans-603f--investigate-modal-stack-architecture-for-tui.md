---
# beans-603f
title: Investigate modal stack architecture for TUI
status: todo
type: task
priority: low
tags:
    - research
    - architecture
created_at: 2025-12-15T12:50:06Z
updated_at: 2026-10-17T12:08:03Z
parent: beans-gl8t
blocking:
    - beans-mszj
---

Current flat enum of 17 view states doesn't reflect hierarchy (modals vs full-screen views).

## Research
- Hierarchical state structure
- Modal stack approach
- How other Bubble Tea apps handle this

## Potential approaches

### Option A: Hierarchical state
```go
type viewState struct {
    base  baseView  // list, detail, help
    modal *modalView // nil or active modal
}
```

### Option B: Modal stack
```go
type App struct {
    currentView view  // list or detail
    modalStack  []modal
}
```

## Potential benefits
- Clearer state ownership
- Easier to add new modals
- Background view handling becomes automatic (top of stack)

## Trade-offs
- More complex state structure

## Deliverable
Research document with recommendations and proof of concept.
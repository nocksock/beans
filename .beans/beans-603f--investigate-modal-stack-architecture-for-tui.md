---
# beans-603f
title: Modal interface instead of view-state enum
status: todo
type: task
priority: low
size: xl
tags:
    - research
    - architecture
created_at: 2025-12-15T12:50:06Z
updated_at: 2026-10-17T14:10:28Z
parent: beans-gl8t
blocking:
    - beans-mszj
blocked_by:
    - beans-5c6x
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
## Code review 2026-10-17 (beans-9pxj)

Reshaped (R5): `type modal interface{ Update; View; Typing() bool; Bindings() }` + base (list/detail) + modal (or stack, since error-over-create nests). One `closeModalMsg{refresh bool}` replaces ~10 close*Msg types and hand-managed previousState; removes 18-way switches in Update/View/canQuit/typingText/chordBindings. Absorbs beans-znuf. Structurally fixes previousState bugs (beans-et11 B9) and stale detail history (tui.go:533, tagpicker.go:125 reusing backToListMsg; detail rebuild losing scroll). Do after beans-5c6x.

## Sequencing (review beans-9pxj)

Step 4 of 5: last; after 5c6x, u7nb, r7wv have shrunk the state machine.

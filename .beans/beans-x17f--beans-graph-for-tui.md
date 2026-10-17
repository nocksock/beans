---
# beans-x17f
title: beans graph for tui
status: in-progress
type: task
priority: deferred
created_at: 2026-10-17T10:43:30Z
updated_at: 2026-10-17T14:02:50Z
parent: beans-7jm4
---

surely there's a cli tool out there that's good at rendering graphs, maybe even interactive ones for navigation?

## Plan (revised)

A 2D graph turned out to be a project of its own (see beans-bcss, code on branch `spike/graphlayout`). Instead: a **dependency navigator** showing transitive blocking chains as trees.

```
 Dependencies of beans-mszj  release 0.5
 ─ blocked by ─────────────────────────
   ├─ beans-5c6x  launcher picker   todo
   │  └─ beans-a2z4  tui launcher…  todo
   └─ beans-603f  archive state     todo
 ● beans-mszj  release 0.5          todo
 ─ blocks ─────────────────────────────
   └─ (nothing)
```

### Behavior
- `D` from list and detail opens it on the selected bean (no open relations → notice instead).
- Upstream tree (recursive blocked-by) above, focused bean, downstream tree (recursive blocking) below.
- Rows via `ui.RenderBeanRow` with tree prefixes.
- `j`/`k` move, `enter` opens detail (esc returns to navigator), `D` on a row re-focuses, `esc`/`q` back to origin.
- Completed/scrapped (archive statuses) hidden by default, `A` toggles; hidden beans also hide what's behind them.
- Repeats (diamonds): shown, dimmed, not expanded again. Cycles: cut and marked `↺`.

### Architecture
- `internal/tui/depnav.go`: pure tree builder (table-tested) + bubbletea model; beans loaded via GraphQL resolver.
- New `depnav` keymap context, help overlay entry, README keybinding row.

### Steps
- [x] Spike: mermaid-ascii vs hand-rolled 2D layout
  - mermaid-ascii: grid layout not by depth, muddled diamonds/cycles, no node positions. Hand-rolled: better but became its own project → beans-bcss.
- [ ] Tree builder + tests
- [ ] Navigator model/view, `D` binding in list + detail, keymap context
- [ ] enter→detail→back, re-focus, `A` toggle
- [ ] Help overlay + README

## Work Log
2026-10-17 14:00: Planned with user: blocking graph, focused+navigable, spike rendering engine first
2026-10-17 14:30: Spike done; chose hand-rolled layout over mermaid-ascii
2026-10-17 15:30: Hand-rolled layout too complex; parked on spike/graphlayout (beans-bcss); replanned as tree navigator

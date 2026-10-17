---
# beans-x17f
title: beans graph for tui
status: in-progress
type: task
priority: deferred
created_at: 2026-10-17T10:43:30Z
updated_at: 2026-10-17T13:29:54Z
parent: beans-7jm4
---

surely there's a cli tool out there that's good at rendering graphs, maybe even interactive ones for navigation?

## Plan

Interactive **blocking-dependency graph** view in the TUI (parent/child is already covered by the list tree).

### Decisions
- Scope: blocking / blocked-by edges only, not parent.
- Focused: opens on the selected bean, shows its connected blocking component (both directions).
- Hide completed/scrapped by default; `A` toggles them back in (dimmed).
- Layout left→right: blockers left, blocked right.
- Rendering engine: decided by a spike (mermaid-ascii embed vs hand-rolled layered layout).
- CLI output (`beans graph <id>`) is a separate follow-up bean, not in scope.

### Entry / exit
- New `viewGraph`, key `D` from list and detail view on the selected bean (`g` is taken).
- `esc`/`q` returns to origin view.
- No open blocking relations → status-bar message instead of opening.

### Rendering
- Node box: ID, truncated title, status (existing status colors); selected node highlighted border.
- Cycles: break for layout, mark the reversed edge visibly; never crash.
- Viewport scrolls to keep selection visible.

### Navigation
- `h`/`l`: move to neighbor in previous/next layer (prefer edge-connected).
- `j`/`k`: move within layer.
- `enter`: open detail; `esc` from detail returns to graph.
- Optional: `s`, `b`, `B` act on the selected node, graph refreshes.

### Architecture
- `internal/graphlayout`: pure `Layout(nodes, edges) → positions + edge routes`, no TUI deps, table-driven tests; reusable by future CLI.
- `internal/tui/graphview.go`: bubbletea model; loads via `graph.Resolver`, builds component, calls layout, renders with lipgloss.

### Phases
- [x] 1. Spike: render 2-3 real graphs from `.beans/` with mermaid-ascii and a minimal hand-rolled layered layout (longest-path layering + barycenter crossing reduction). Compare readability + node-position access. Record decision here.
  - **Decision: hand-rolled.** mermaid-ascii lays out on a grid not by depth (fan-in of 5 nodes = 25 lines tall), muddles diamonds/cycles, 5-line boxes, coords unexported (no node positions for nav). Hand-rolled ~200 LOC spike: depth columns, 3-line boxes, positions available, <0.1ms. Gaps to fix: edge junction glyphs (┬┼┤), skip-layer edges must route around boxes, draw+mark cycle edges.
- [ ] 2. Layout package + static rendering in `viewGraph` (with tests).
- [ ] 3. Navigation, enter→detail, completed toggle, help entries.
- [ ] 4. (optional) node actions s/b/B with refresh.

## Work Log
2026-10-17 14:00: Planned with user: blocking graph, focused+navigable, spike rendering engine first
2026-10-17 14:30: Spike done; chose hand-rolled layout over mermaid-ascii

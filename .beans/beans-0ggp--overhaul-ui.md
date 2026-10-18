---
# beans-0ggp
title: overhaul ui
status: in-progress
type: task
priority: normal
created_at: 2026-10-18T09:25:35Z
updated_at: 2026-10-18T09:49:47Z
---

Lazygit-style composable panes with `+`/`-` cycling through layout presets. Replaces the full-screen detail and depnav views.

## Scope

This bean ships the pane/layout framework and moves the existing panes into it: list, context (task view), log and dep-graph. Follow-ups are child beans that plug into the framework: navigable log with log detail, git task history with diff, and a second task view linked to the dep-graph.

## Decisions

- **One layout tree; presets are visibility sets.** A single declarative split tree contains every pane. A preset is `{visible set, focus}`. Hidden leaves collapse and their siblings take the space. The arrangement is fixed; only visibility changes.
- **Presets are snapshots.** Toggle keys flip a pane's visibility. `+`/`-` jumps to the next/previous preset in the focused pane's cycle and drops manual toggles.
- **Focus:** `tab`/`shift+tab` cycles the visible panes, `esc` pops a focus stack and enter on the list focuses the context. Focusing a pane keeps the current layout and only changes which preset cycle `+`/`-` uses.
- **Context pane** shows detail for the last-focused source pane. In this bean that is always a task view, of either the list selection or the dep-graph cursor.
- **Task view has no inner focus.** It shows the preview header and a scrollable body (j/k, u/d, pgup/pgdn). Links appear only in the static header; you navigate them in the dep-graph. Bean action keys (`s t P S T p b B e E ! Y`) act on its bean. `h`/`l` prev/next bean is dropped.
- **Dep-graph** is rooted at the context's bean when it becomes visible. Moving its cursor updates the context live. Enter re-roots it (replaces the old `D` refocus). `A` toggles done beans. The list keeps its own selection.
- **Enter on the list when the context is hidden** switches to the first preset in the cycle that shows the context.
- **Narrow terminals:** splits have minimum sizes. Below the threshold (120 cols) an HSplit drops its lowest-priority visible child (the context next to the list). Focusing the context then shows it at full width. The log needs at least 18 rows.
- Modals are unchanged; beans-603f covers them.

### Default tree and presets

```go
VSplit(
  HSplit(Leaf(List), VSplit(Leaf(DepGraph), Leaf(Context))),
  Leaf(Log, Fixed(8)),
)
```

- List focused: `{List, Context, Log}` → `{List, Log}` → `{List}` → wrap
- Context focused: `{List, Context, Log}` → `{DepGraph, Context, Log}` → `{Context}` → wrap
- `-` walks the cycle backwards.

### Keys (new `Layout` keymap group, overridable via config)

| key | action |
|---|---|
| `+` / `=` | next preset |
| `-` | previous preset |
| `tab` / `shift+tab` | focus next/previous visible pane |
| `esc` | focus back (pop stack) |
| `L` | toggle log |
| `D` | toggle dep-graph (was: open full-screen depnav) |
| `v` | toggle context (was: open detail; covers beans-rjk2) |

## Architecture

- **`internal/tui/layout/`** is a new package: a pure layout engine with no bubbletea models.
  - Nodes: `HSplit`, `VSplit`, `Leaf(id)`. Sizing: `Weight(n)`, `Fixed(n)`, `Min(n)`, plus a drop priority.
  - `Render(tree, visible, focus, w, h, draw func(id, w, h) string) string` collapses hidden leaves, distributes space and draws borders, titles and the focus highlight.
- **`internal/tui/presets.go`** holds the tree literal and the per-focus preset cycles. It is the only file to edit while iterating on layouts.
- **`internal/tui/workspace.go`** holds the layout state: visible set, focus, focus stack and preset index.
  - It owns the toggle and cycle logic, routes keys and messages to the focused pane, and forwards `selectionChangedMsg{bean}` from list and dep-graph to the context.
  - `App` delegates to it whenever no modal is open.
- **Pane interface:** `Update(msg) (Pane, tea.Cmd)`, `View(w, h int, focused bool) string`, `Title() string`, `Bindings() []key.Binding`. The layout draws the chrome, and a single global footer shows the focused pane's bindings plus the layout keys.
  - `listPane`: `list.go` minus its own border and footer.
  - `contextPane`: `preview.go` plus a viewport plus the action keys from `detail.go`. The rest of `detail.go` is deleted.
  - `depGraphPane`: `depnav.go` minus its title and footer. Emits selection changes.
  - `logPane`: `logpanel.go` with a height parameter. Still non-navigable (follow-up).
- **Removed:** `viewDetail`, `viewDepNav`, `App.history`, the `depNav*` fields, `twoColumnMinWidth`/`paneWidths`/`isTwoColumnMode`, and the `Detail`/`DepNav` keymap groups. Their bean actions merge into a shared `Bean` group, and the help overlay lists the `Layout` group.

## Implementation steps

- [x] `layout` package: node types, size distribution, collapsing, narrow-width drop, chrome rendering, with table-driven tests
- [x] Pane interface and `presets.go` (tree and cycles)
- [x] `workspace.go`: focus, focus stack, toggles, preset cycling, msg routing, with tests
- [x] `listPane` adapter (strip border/footer); App renders via workspace
- [x] `contextPane` (preview + viewport + bean actions); delete detail view and `history`
- [x] `depGraphPane` (strip chrome, selection drives context, enter re-roots); delete `viewDepNav`
- [x] `logPane` with height param; `L` via workspace
- [x] `Layout` keymap group, merge Detail/DepNav into `Bean` group, help overlay, config override names
- [x] Update existing tests (list, depnav, twocolumn, keys, tui)
- [x] Manual tmux check at 80/120/200 cols with a test `--beans-path`
- [x] README/docs for the new keys

## Follow-ups (blocked by this bean, under epic beans-7jm4)

- beans-8pxh navigable log pane with log-detail context
- beans-so3o git history backend (independent, can start now)
- beans-3srp task history pane with diff context
- beans-iuva second task view linked to dep-graph (idea)

## Testing

- Layout engine: sizes, collapsing hidden leaves, min sizes, narrow drop, focus highlight.
- Workspace: preset cycling both ways, a manual toggle followed by `+` resetting, enter switching preset to reveal the context, the esc focus stack, dep-graph cursor driving the context, the list selection restored on refocus.

## Original request


panes like lazygit
with key +/- zooming in/changing layouts.

default is


    [task-list] | task-view
    ---------------
    log

[] marks focus

there is no dedicated detail view
pressing enter moves focus to task-view

pressing + will change view to

    [task-list]
    ----------
    log

pressing + again will change view to

    [task-list]

pressing + again will change view back to

    [task-list] | task-view
    ---------------
    log


when focus is on task view we cycle
from

    task-list | [task-view]
    ---------------
    log

to

    dependecy-navigator
    -----
    [task-view]
    ------
    log

    [task-view]

log and dependecy-navigator (or rather anything) can be toggled on and off anywhere
+/- are basically cycling through presets of toggled states
however we might change these often while iterating, so we need a very modular and simple way to compose these view.


panes we'd like to have:
- context (detail view for whatever is currently selected in another pane)
- task list
- task view (displayed in context when task list is focused with selected task)
- log (navigatable, when focussed show log detail in context pane)
- log detail (output of a command)
- dependecy-graph (was: dependecy-navigator)
- task history (navigatable git log, when focussed show task diff in context )
- task diff (displayed in context when history is in focus)


i can also imagine that we have 2 task views open + dependecy-graph. 
where one task views the current task, and the other the one selected in dependency graph.

## Work Log
2026-10-18 11:00: Brainstormed design with user; plan written into bean
2026-10-18 11:05: Created follow-ups 8pxh, so3o, 3srp, iuva; status in-progress
2026-10-18 11:40: layout package (arrange, drop, frame) with tests committed
2026-10-18 12:20: workspace + presets committed (63d7f7c)
2026-10-18 13:10: panes wired into App, detail/depnav views removed, tests + README updated (f344467)
2026-10-18 13:12: deviation: Detail/DepNav keymaps became `context`/`depGraph` groups (not one shared `Bean` group); pane `back` bindings live per pane so esc in the list keeps its drill-out meaning
2026-10-18 13:13: log is skipped by tab (unfocusable) until beans-8pxh makes it navigable
2026-10-18 13:14: manual tmux check at 80x24, 100x40, 120x36, 200x45 ok
2026-10-18 13:30: enter in dependency graph now also focuses details (user feedback)
2026-10-18 13:50: pane interface (panes.go) replaces per-pane switches in App

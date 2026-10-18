---
# beans-8pxh
title: Navigable log pane with log-detail context
status: todo
type: task
created_at: 2026-10-18T09:49:29Z
updated_at: 2026-10-18T09:49:29Z
parent: beans-7jm4
blocked_by:
    - beans-0ggp
---

Follow-up of beans-0ggp. Make the log pane focusable and navigable (cursor over log entries). When the log pane is focused, the context pane shows a log-detail view for the selected entry: the full, scrollable output of that launcher run. Plugs into the pane framework (Pane interface, workspace, presets.go). Add preset cycle for log focus.

## Notes from beans-0ggp

- Add a pane by writing an adapter implementing `pane` (`internal/tui/panes.go`), registering it in `App.newPanes()`, and placing it in `layoutTree`/`presetCycles` (`presets.go`).
- The context pane currently always shows a bean (`contextModel`); a different context (log detail / diff) means letting the context pane render per source, e.g. via the focused `selectionSource`.
- The log is listed in `unfocusable` (`presets.go`); remove it there once it is navigable.

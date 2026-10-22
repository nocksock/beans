---
# beans-8pxh
title: Navigable log pane with log-detail context
status: archived
type: task
priority: normal
created_at: 2026-10-18T09:49:29Z
updated_at: 2026-10-19T20:46:15Z
parent: beans-7jm4
blocked_by:
    - beans-0ggp
---

Follow-up of beans-0ggp. Make the log pane focusable and navigable (cursor over log entries). When the log pane is focused, the context pane shows a log-detail view for the selected entry: the full, scrollable output of that launcher run. Plugs into the pane framework (Pane interface, workspace, presets.go). Add preset cycle for log focus.

## Notes from beans-0ggp

- Add a pane by writing an adapter implementing `pane` (`internal/tui/panes.go`), registering it in `App.newPanes()`, and placing it in `layoutTree`/`presetCycles` (`presets.go`).
- The context pane currently always shows a bean (`contextModel`); a different context (log detail / diff) means letting the context pane render per source, e.g. via the focused `selectionSource`.
- The log is listed in `unfocusable` (`presets.go`); remove it there once it is navigable.

## Work Log

2026-10-18 13:05: Extracted shared scroller from contextModel
2026-10-18 13:10: Log pane cursor (follows latest), LogKeyMap, removed unfocusable
2026-10-18 13:15: logDetailModel shows full run output in context pane via runSource; log preset cycle
2026-10-18 13:22: Tests + README; verified manually in tmux
2026-10-18 13:50: List/context split 50:50 (both min 60)
2026-10-18 14:20: Row columns reordered: status, title, tags, type, size, id; status/type right-aligned, labels active/done/feat (TUI, dep graph, CLI)
2026-10-18 14:25: Selected cut-off titles scroll to their end after 2s
2026-10-18 14:40: Epic rows show no status
2026-10-18 14:50: Type moved to the 2nd column
2026-10-18 15:00: Panes focusable via 1-4 (lazygit-style), key shown in pane titles

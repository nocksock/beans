---
# beans-ung3
title: filter and trigger keybinds
status: in-progress
type: task
priority: normal
created_at: 2026-10-17T12:59:24Z
updated_at: 2026-10-18T15:56:00Z
---

similar to lazygit. ? would open a keybinds dialog with all currently available keybinds grouped by global and context keybinds. / starts filtering. enter will run. without filter binds are selectable via j/k, enter runs selected.

## Plan

Design spec: `docs/specs/2026-10-18-keybinds-dialog-design.md` in the hub.

Decisions:
- Replaces the static help overlay on `?`; available from the panes only, not in pickers or modals.
- Three sections: Global, Layout, and the focused pane (headed by its title). Headers are skipped by the cursor.
- Rows use the bindings' own help labels; terse default labels get polished.
- `/` filters by key label or description (case-insensitive substring); empty sections disappear; arrow keys move while typing.
- `enter` closes the dialog and replays the binding's first key (chord message for chords) through the app's normal key handling. No per-action wiring.

Steps:
- [x] `keys.go`: polish terse help labels in `DefaultKeyMap` (filter, archive, open, refocus, open log, ...)
- [x] `keybinds.go`: `keybindRow`, pure `keybindRows(km, paneTitle, paneBindings)` and `filterRows(rows, query)` with table-driven tests
- [x] `keybinds.go`: `keybindsModel` with textinput filter, cursor that skips headers, `scroller` viewport, `runKeybindMsg`; Update/View tests
- [x] `tui.go`: host the new model in `viewHelpOverlay`, handle `runKeybindMsg` by replaying the key, report the filter input in `typingText`/`canQuit`; app-level replay test (plain key and chord)
- [x] Remove `help.go`/`help_test.go`, update README keybinding docs if the help overlay is mentioned
- [x] Manual check with `go run . --beans-path <tmp>`: open, filter, run `s`, run `g t`, close

## Work Log
2026-10-18 10:30: Planned the keybinds dialog with the user; spec in docs/specs/2026-10-18-keybinds-dialog-design.md
2026-10-18 11:05: Implemented keybinds dialog (keybinds.go) with TDD, replaced help overlay, wired replay into the app
2026-10-18 11:20: Polished binding labels, README keybinding docs, manual check in tmux: filter, run s, run g t

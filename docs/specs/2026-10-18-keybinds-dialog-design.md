# Keybinds dialog (beans-ung3)

A lazygit-style keybinds dialog for the TUI. It replaces the static help
overlay on `?`, lists the keys available right now, can be filtered, and runs
the selected binding.

## Behaviour

- `?` from any pane (list, details, dependencies, log) opens the dialog. It is
  not available inside pickers or text modals, which keep their own footers.
- The dialog lists every bound action in three sections, in this order:
  - **Global**: the `Global` key map (quit, help, reload config, open log).
  - **Layout**: the `Layout` key map (presets, pane focus, pane toggles).
  - **The focused pane**, headed by the pane's title (Beans, Details,
    Dependencies, Log): that pane's key map. The details pane shows the key
    map of its active view (bean or run output).
- A row shows the binding's key label (all keys, `esc/bksp`) and its help
  description. Actions without keys are skipped.
- `j`/`k`, `↓`/`↑` move the cursor over rows; headers are skipped. The cursor
  starts on the first row. The list scrolls to keep the cursor visible.
- `/` focuses a filter input at the top. Typing narrows the rows: a row stays
  when its key label or description contains the query (case-insensitive,
  substring). Sections without rows disappear. The cursor is reset to the
  first visible row on every change. While the input has focus, printable
  keys (including `j`, `k` and `?`) are typed; the arrow keys still move the
  cursor, so filter + `enter` runs the first match without leaving the input.
  `enter` runs the selected row; `esc` clears the filter and returns focus to
  the rows.
- `enter` on a row closes the dialog and runs the binding.
- `esc` or `?` (the `help.close` binding) closes the dialog when the rows
  have focus.
- Footer: `enter run  / filter  esc close`.

## Running a binding

Enter closes the dialog and replays the binding's first key through the app's
normal key handling, as if the user had pressed it in the pane: a key message
for plain keys, a chord message for chords (`g t`). This needs no per-action
wiring, honours rebinds, and works for any future binding. The help row
itself just closes the dialog.

## Components

- `internal/tui/keybinds.go` replaces `help.go`: the `keybindsModel` with a
  `textinput` filter, the rows, a cursor and a `scroller` viewport.
  - `keybindRow`: either a header (`section string`) or a binding (`keys`,
    `desc`, `binding key.Binding`).
  - `keybindRows(km KeyMap, pane string, paneBindings []key.Binding)
    []keybindRow`: pure; builds the three sections.
  - `filterRows(rows []keybindRow, query string) []keybindRow`: pure; keeps
    matching rows and the headers that still have rows.
  - `runKeybindMsg{key string}`: sent on enter.
- `keys.go`: `contextBindings` already pairs names with bindings; the row
  builder reuses it so order follows the struct fields. Help descriptions in
  `DefaultKeyMap` are polished where the old overlay said more, e.g.
  `filter (text, field:value, sort:)`, `archive completed`,
  `open children / details`, `root graph here`, `latest run output`.
- `tui.go`: the existing `viewHelpOverlay` state hosts the new model. On
  `runKeybindMsg` the app returns to the pane view and calls its own update
  with the replayed key (`chordMsg` when the key contains a space, except the
  bare space bar). `typingText` and `canQuit` report the dialog's filter
  input as typing. `chordBindings` keeps returning the `Help` key map for the
  dialog: chords are resolved later, when the key is replayed.

## Testing

Table-driven unit tests, no bubbletea program needed:

- `keybindRows`: sections and headers, pane title used, unbound actions
  skipped, order follows the key map.
- `filterRows`: matches key label or description, case-insensitive, drops
  empty sections, empty query returns everything.
- Cursor: skips headers up and down, clamps at the ends, resets on filter.
- Update: enter yields `runKeybindMsg` with the first key; chords yield the
  chord; `/` enters the filter; esc clears the filter, then closes.
- App: `runKeybindMsg` dispatches the key (e.g. opens the status picker) and
  the chord replay reaches the pane.

---
# beans-4hsk
title: e to edit in new popup E to edit in editor
status: todo
type: task
priority: normal
created_at: 2026-10-17T12:45:01Z
updated_at: 2026-10-17T12:47:28Z
---

Rebind the TUI edit keys: `e` opens an in-TUI edit popup (based on the create modal), and `E` opens `$EDITOR` (what `e` does today).

## Plan

### Keys
- List view and detail view: `e` → edit popup for the bean under the cursor (multi-selection is ignored); `E` → `$EDITOR` (existing `openEditorMsg` flow).
- The launcher result view keeps `e` = open log in editor (unchanged).
- Update the help bar (list.go, both variants), the detail help and the help overlay (help.go): `e edit`, `E $EDITOR`.

### Popup: generalize `createModalModel` into a create/edit form
- Add an edit mode: `beanID` (empty = create). The constructor prefills title, type/status/priority indices and body from the bean.
- Header reads "Edit Bean" plus a muted bean ID line; help labels say "save" instead of "create".
- Description textarea height in edit mode grows with the terminal height (min 5 lines) and recomputes on WindowSizeMsg. Create mode keeps 5 lines.
- Keys are the same as in create: tab/shift+tab, enter (outside body) / ctrl+s = save, esc = cancel, ctrl+e (and `e` on selectors) = save & open `$EDITOR`.
- In edit mode an empty title is allowed if the body is non-empty (same rule as create). If both are empty → validation hint, don't save.
- Parent/tags/blocking are not in the popup (they keep their own pickers).

### Saving
- New `beanEditedMsg{beanID, title, type, status, priority, body, openEditor}`. tui.go handles it via `resolver.Mutation().UpdateBean`, sending only fields that changed. If nothing changed, it's a no-op (no `updated_at` bump).
- After saving: reload the list, keep the cursor on the bean (`cursorTarget`), and return to the view the popup was opened from (list or detail, refresh detail). If `openEditor` is set, follow with `openEditorMsg`.
- Concurrent on-disk changes while the popup is open: last write wins (only changed fields are written, so the risk is small).

### Tests
- Table-driven tests for the modal in edit mode: prefill, diffing of changed fields, no-op on no change, empty title + body rule, `openEditor` flag.
- tui_test: `e` opens the popup and `E` emits `openEditorMsg`, in both list and detail.

## Work Log
2026-10-17: Planned with user (reuse create modal, cursor bean only, grow body, ctrl+e save & edit)

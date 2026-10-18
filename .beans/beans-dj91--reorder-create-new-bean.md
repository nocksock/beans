---
# beans-dj91
title: reorder create new bean
status: in-progress
type: task
priority: normal
created_at: 2026-10-18T15:34:11Z
updated_at: 2026-10-18T16:12:50Z
---

Reorder the create/edit bean form in the TUI (`internal/tui/createmodal.go`, shared by create and edit via `isEdit()`):

```
Create New Bean

╭──────────────────────────────────────────────╮
│ Enter bean title (optional)...               │
╰──────────────────────────────────────────────╯

╭──────────────────────────────────────────────╮
│ Description (markdown, optional)...          │
│                                              │
╰──────────────────────────────────────────────╯

Prefix    Type        Status        Priority  Size
  beans     task        draft         normal    md

enter create  ctrl+e create & edit  tab next  esc cancel
```

## Plan

- [x] Reorder `createField` constants to title, body, prefix, type, status, priority, size so focus cycling (tab/shift+tab) follows the visual order
- [x] View: header, title box, description box, selector row, error, help. Drop the "Title" and "Description" labels; placeholders and the focused border color identify the fields
- [x] Prefix joins the selector row as the first column, labelled "Prefix". Create form only. With one prefix it stays muted and unfocusable, as now. Column width fits the longest configured prefix plus arrow padding, with "(none)" as the floor
- [x] Modal minimum width grows so the five columns fit without wrapping; edit form keeps the four-column minimum
- [x] Edit form gets the same order without the prefix column (same component, no other change; `editModalChromeHeight` shrinks by the two removed label rows)
- [x] Tests: update the view test (labels removed, "Prefix" label present); focus-order test walking tab through all seven fields with two prefixes and skipping prefix with one; width test that the five columns fit at the minimum modal width

## Work Log
2026-10-18 18:12: Plan agreed with user, status set to in-progress
2026-10-18 18:23: Reordered the form, prefix joins the selector row, labels dropped, tests updated; go test ./... green

---
# beans-r7wv
title: Generic option picker for status/type/priority
status: todo
type: task
tags:
    - refactoring
    - code-review
created_at: 2026-10-17T14:02:23Z
updated_at: 2026-10-17T14:02:23Z
parent: beans-7jm4
---

Review beans-9pxj (`docs/reviews/2026-10-17-code-quality-review.md` in hub), R3.

statuspicker.go, typepicker.go, prioritypicker.go are ~95% identical. Replace with one `optionPickerModel{kind, items []option, onSelect}`. Extract `newPickerList(items, delegate, title, w, h)`, `renderCursor(selected)` (7 pickers) and `sortByTypeThenTitle` (parent+blocking pickers). Makes beans-jz5j a single implementation.
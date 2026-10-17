---
# beans-r7wv
title: Generic option picker for status/type/priority
status: todo
type: task
priority: normal
size: md
tags:
    - refactoring
    - code-review
created_at: 2026-10-17T14:02:23Z
updated_at: 2026-10-17T14:10:28Z
parent: beans-7jm4
---

Review beans-9pxj (`docs/reviews/2026-10-17-code-quality-review.md` in hub), R3.

statuspicker.go, typepicker.go, prioritypicker.go are ~95% identical. Replace with one `optionPickerModel{kind, items []option, onSelect}`. Extract `newPickerList(items, delegate, title, w, h)`, `renderCursor(selected)` (7 pickers) and `sortByTypeThenTitle` (parent+blocking pickers). Makes beans-jz5j a single implementation.
## Sequencing (review beans-9pxj)

Step 4 of 5: after beans-u7nb; unblocks beans-jz5j.

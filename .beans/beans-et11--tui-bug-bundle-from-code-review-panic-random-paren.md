---
# beans-et11
title: TUI bug bundle from code review (panic, random parent, truncation)
status: todo
type: bug
priority: high
size: sm
tags:
    - code-review
created_at: 2026-10-17T14:02:00Z
updated_at: 2026-10-17T14:10:28Z
parent: beans-7jm4
---

From review beans-9pxj (`docs/reviews/2026-10-17-code-quality-review.md` in hub), items B9, B11, B14 (ui/tui parts), B20 (tui parts). (B12 bulk errors → beans-p0mg, B18 → beans-5c6x, B19 → beans-603f.)

- [x] B9: panic — `launchersConfiguredMsg` hardcodes `previousState = viewDetail` (`tui.go:793`); opened from list, `a.detail.bean` is nil → next key or beansChangedMsg (`tui.go:227`) derefs nil. Keep existing previousState; add test.
- [ ] B11: multi-select CreateChild picks random parent via map iteration (`list.go:657`). Use cursor bean or refuse with >1 selected.
- [ ] B14: byte-based truncation splits UTF-8 and `len()` padding misaligns: `ui/styles.go:327-329,542-546,575-582`, `tui/modal.go:45-48`; `overlayLine` counts runes not cells (`modal.go:125-140`). Use ansi.Truncate / lipgloss.Width.
- [ ] Launch key with no selection uses `Items()[0]` even when filtered out (`list.go:724`)
- [ ] ChangeParent loses type of hidden/collapsed selected beans — use `m.beansByID` (`list.go:500`)
- [ ] `pickerModalConfig.Title` never rendered
- [ ] `ui.IsValidColor` accepts `#zzz` (`ui/styles.go:47-51`)
- [ ] Tests: status/type/priority/parent selected flows, back navigation, launchersConfiguredMsg error path
B9 fixed by beans-0ggp: the detail view is gone and `launchersConfiguredMsg` returns to `viewList`.

## Sequencing (review beans-9pxj)

Step 1 of 5: after beans-at6c, before beans-zs6d B2-B4.

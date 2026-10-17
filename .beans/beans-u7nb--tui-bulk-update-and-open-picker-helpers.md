---
# beans-u7nb
title: TUI bulk-update and open-picker helpers
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

Review beans-9pxj (`docs/reviews/2026-10-17-code-quality-review.md` in hub), R4.

- [ ] Extract `a.applyToBeans(ids, func(id) error) tea.Cmd` + `a.returnFromModal(refreshID)` replacing 5 near-identical blocks in App.Update (tui.go:294-449, 682-706); single place for failure feedback (beans-p0mg)
- [ ] list/detail: `m.targets() []*bean.Bean` + `openPickerFor(kind, beans)` replacing 6× ids + "%d selected beans" plumbing (list.go:491-737, detail.go:347-446)
## Sequencing (review beans-9pxj)

Step 4 of 5: after beans-5c6x; land TUI half of beans-p0mg here.

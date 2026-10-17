---
# beans-eywq
title: Simplify GraphQL filters and make relation order deterministic
status: todo
type: task
tags:
    - refactoring
    - code-review
created_at: 2026-10-17T14:02:24Z
updated_at: 2026-10-17T14:02:24Z
parent: beans-iaia
---

Review beans-9pxj (`docs/reviews/2026-10-17-code-quality-review.md` in hub), R8.

- [ ] filters.go: ~14 near-identical funcs → `keep(beans, preds)` with predicates built from the filter (~299→120 lines)
- [ ] Replace hardcoded "normal" priority (filters.go:126,145) with config default; drop dead nil-cfg check
- [ ] FindBlocked/FindBlockers O(n²) + map iteration → sort blockedBy/blocking/children results
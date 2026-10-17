---
# beans-ole9
title: Move bean validation from CLI into resolver/beancore
status: todo
type: task
priority: normal
size: lg
tags:
    - refactoring
    - code-review
created_at: 2026-10-17T14:02:24Z
updated_at: 2026-10-17T14:10:28Z
parent: beans-iaia
---

Review beans-9pxj (`docs/reviews/2026-10-17-code-quality-review.md` in hub), R9.

- [ ] Status/type/priority validation currently only in cmd/create.go:41-49 and cmd/update.go:144-166; GraphQL and TUI bypass it. Move into beancore (or resolver validateBean)
- [ ] CreateBean: existence/self/cycle checks for blocking/blockedBy like AddBlocking
- [ ] Single source for defaults (resolver uses config default type; see beans-at6c)
- [ ] Atomic update: one updateBean input carrying link changes (addTags/removeTags, addBlocking…) instead of N+2 mutations (update.go:64-121)
- [ ] config.validate() checks DefaultStatus/DefaultType (currently only cmd/check.go)
Do before beans-digu.
## Sequencing (review beans-9pxj)

Step 2 of 5: together with beans-rpmz (tag validation) and beans-zs6d B1 (Update API).

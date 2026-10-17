---
# beans-v6no
title: CLI and config helper cleanup
status: todo
type: task
priority: low
size: md
tags:
    - refactoring
    - code-review
created_at: 2026-10-17T14:02:24Z
updated_at: 2026-10-17T14:10:28Z
parent: beans-iaia
---

Review beans-9pxj (`docs/reviews/2026-10-17-code-quality-review.md` in hub), R10.

- [ ] `findBean(ctx, id, jsonMode)` helper replacing show/update/delete/graph lookups; remove unreachable ErrNotFound branch (content.go:108)
- [ ] `newResolver()` (12 sites build &graph.Resolver{Core: core})
- [ ] Flag help lists configured status/type/priority names via config `namesOf()` (create.go:118-130, update.go:200-212)
- [ ] config: generic name/find helpers for GetType/TypeNames/TypeList/IsValidType + priority twins; builtinStatusList nil receiver
- [ ] ReplaceConfig lock is misleading (readers don't lock) → atomic.Pointer or document
- [ ] roadmap JSON through output package; output takes io.Writer + tests
- [ ] search: RLock fast path, dedupe IndexBean/IndexBeans doc building
## Sequencing (review beans-9pxj)

Step 5 of 5: cleanup, any time.

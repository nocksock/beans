---
# beans-rpmz
title: CLI/GraphQL bug bundle from code review (partial-ID links, tags, misc)
status: todo
type: bug
priority: normal
size: md
tags:
    - code-review
created_at: 2026-10-17T14:02:00Z
updated_at: 2026-10-17T14:10:28Z
parent: beans-iaia
---

From review beans-9pxj (`docs/reviews/2026-10-17-code-quality-review.md` in hub), items B5, B6 + cmd/graph parts of B20. (B7 init overwrite → beans-9p2n, B8 default type → beans-at6c.)

- [ ] B5: link mutations resolve target via `Core.Get` (prefix) but store the typed partial ID → `check` later reports broken link. Store returned `.ID`. (`schema.resolvers.go:110-125,215,240,281`)
- [ ] B6: tags never validated/normalized in production (`applyTags` only used by tests, `cmd/content.go:46`); resolver assigns raw tags (`schema.resolvers.go:105,163`). `mergeTags` (`content.go:69`) reorders via map → git diff churn. Validate in resolver, keep order, drop applyTags.
- [ ] `check.go:72-98`: error counting by string prefix "invalid color" hides type-colors-ok line
- [ ] `roadmap.go:285` containsStatus ignores `ExpandStatuses`
- [ ] `SetParent`/`AddBlocking` format cycles with %v instead of formatCycle
- [ ] `err == beancore.ErrNotFound` → `errors.Is` (`schema.resolvers.go:56,326`)
- [ ] `Core.Delete` returns sentinel ErrAmbiguousID vs Get's AmbiguousIDError type
- [ ] `update` repeats "blocking" in changes once per target
## Sequencing (review beans-9pxj)

Step 2 of 5: one change to the mutation path together with beans-ole9 and beans-zs6d B1.

---
# beans-wg30
title: Link abstraction in bean package
status: todo
type: task
priority: low
size: md
tags:
    - refactoring
    - code-review
created_at: 2026-10-17T14:02:23Z
updated_at: 2026-10-17T14:10:28Z
parent: beans-c8p4
---

Review beans-9pxj (`docs/reviews/2026-10-17-code-quality-review.md` in hub), R6.

- [ ] `type LinkType` constants, `(b *Bean) Links() []Link`, `RemoveLinks(pred) int` in internal/bean
- [ ] Rewrite FindIncomingLinks/CheckAllLinks/FixBrokenLinks/RemoveLinksTo and FindBlockers/FindBlocked on top (~halves links.go)
- [ ] Move ValidParentTypes hierarchy (links.go:503) from beancore to config types
- [ ] Shared `resolveLocked(prefix)`; `Core.Delete` takes exact ID only
## Sequencing (review beans-9pxj)

Step 5 of 5: cleanup, any time.

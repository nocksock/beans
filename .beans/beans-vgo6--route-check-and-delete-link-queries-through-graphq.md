---
# beans-vgo6
title: Route check and delete link queries through GraphQL
status: todo
type: task
tags:
    - refactoring
    - code-review
created_at: 2026-10-17T14:02:24Z
updated_at: 2026-10-17T14:02:24Z
parent: beans-iaia
---

Review beans-9pxj (`docs/reviews/2026-10-17-code-quality-review.md` in hub), R12.

check.go:115,119 calls core.CheckAllLinks/FixBrokenLinks and delete.go:48 calls core.FindIncomingLinks directly, violating the "CLI uses GraphQL" rule. Add `incomingLinks` field and checkLinks query / fixLinks mutation (or document the exception).
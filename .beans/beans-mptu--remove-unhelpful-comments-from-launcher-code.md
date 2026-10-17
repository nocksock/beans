---
# beans-mptu
title: Remove unhelpful comments from launcher code
status: archived
type: task
priority: low
tags:
    - code-quality
    - quick-win
created_at: 2025-12-15T12:48:31Z
updated_at: 2026-10-17T14:16:52Z
parent: beans-gl8t
---

Remove 43 unhelpful comments that restate what the code already clearly shows. Keep only comments that explain WHY or provide critical context.

## Comments to remove
- internal/launcher/launcher.go: 8 comments
- internal/launcher/manager.go: 8 comments
- internal/tui/launchprogress.go: 14 comments
- internal/tui/launchconfirm.go: 7 comments
- internal/tui/launcher.go: 5 comments
- internal/tui/launcherpicker.go: 1 comment

## Comments to KEEP (they explain WHY)
- "Note: Stdout/Stderr will be set by caller"
- "must be called with lock held"
- "Create a copy to avoid race conditions"
- "will fail at execution with good error"
- "Availability is checked at execution time"

See code review analysis for specific line numbers.
## Code review 2026-10-17 (beans-9pxj)

Scrapped: launchprogress.go/launchconfirm.go gone; remaining launcher files covered by beans-re37.

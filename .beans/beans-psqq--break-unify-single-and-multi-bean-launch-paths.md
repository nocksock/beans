---
# beans-psqq
title: '[BREAK] Unify single and multi-bean launch paths'
status: scrapped
type: task
priority: low
tags:
    - enhancement
    - breaking
    - architecture
created_at: 2025-12-15T12:49:49Z
updated_at: 2026-10-17T14:02:52Z
parent: beans-gl8t
---

Currently two completely separate code paths:
- Single bean: Uses tea.ExecProcess directly
- Multi-bean: Goes through LaunchManager with progress view

## Proposal
Always use LaunchManager, even for single bean.

## Benefits
- Consistent error handling
- Duration tracking for all launches
- Cancellation support
- Simpler message flow

## Trade-offs
- Slight overhead for single-bean launches (creating manager, goroutine)
- Change in user experience (could show progress for single bean too)
- Breaking change in execution flow

## Implementation
- Remove the `if len(msg.beanIDs) == 1` branch in tui.go:537
- Simplify confirmation logic: check count threshold regardless of path
- Add `DisableProgressForSingle` config flag if you want to skip progress UI for single beans
## Code review 2026-10-17 (beans-9pxj)

Scrapped: multi-bean LaunchManager path no longer exists (foreground vs background split now). Remaining multi-foreground bug in beans-yxao.

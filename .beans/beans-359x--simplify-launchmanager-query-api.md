---
# beans-359x
title: Simplify LaunchManager query API
status: archived
type: task
priority: low
tags:
    - enhancement
    - architecture
created_at: 2025-12-15T12:49:59Z
updated_at: 2026-10-17T14:16:52Z
parent: beans-gl8t
blocking:
    - beans-psqq
---

Currently 5 different query methods:
- GetStatus()
- IsComplete()
- AllSuccessful()
- GetFirstError()
- GetCounts()

## Proposal
Single method returning summary struct:

```go
type LaunchSummary struct {
    Launches      []*BeanLaunch
    Complete      bool
    AllSuccessful bool
    FirstError    *BeanLaunch
    Counts        LaunchCounts
}

func (m *LaunchManager) GetSummary() LaunchSummary
```

## Benefits
- Single lock acquisition
- Impossible to have inconsistent state
- Clearer API

## Trade-offs
- More data computed per call (but you usually need most of it anyway)
- Slightly more memory allocation

## Implementation
Compute all values under single lock, return snapshot.
## Code review 2026-10-17 (beans-9pxj)

Scrapped: GetSummary() implemented, but LaunchManager is dead code; removal in beans-re37.

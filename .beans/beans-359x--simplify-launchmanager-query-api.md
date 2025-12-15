---
# beans-359x
title: Simplify LaunchManager query API
status: todo
type: task
priority: low
tags:
    - enhancement
    - architecture
created_at: 2025-12-15T12:49:59Z
updated_at: 2025-12-15T12:49:59Z
parent: beans-gl8t
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
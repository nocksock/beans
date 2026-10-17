---
# beans-fx99
title: Create comprehensive launcher manager tests
status: todo
type: task
priority: high
tags:
    - test
    - concurrency
created_at: 2025-12-15T12:49:25Z
updated_at: 2025-12-15T12:49:25Z
parent: beans-gl8t
---

Create `internal/launcher/manager_test.go` with tests for parallel execution and concurrency.

## Test scenarios
- Start() with 1, multiple, and many beans (20+)
- Stop() kills all processes, is idempotent
- GetStatus() returns consistent snapshot during active launches
- IsComplete() / AllSuccessful() state tracking
- GetFirstError() returns first failure
- GetCounts() accuracy
- monitorLaunch() stderr capture
- Race condition testing with `-race` flag
- Start after Stop() called (should error)
- Start with invalid beans root
- Stop when no processes running
- Stop when all processes already completed
- Thread-safe status updates

## Focus areas
- Concurrency correctness
- Resource cleanup
- Error handling and stderr capture
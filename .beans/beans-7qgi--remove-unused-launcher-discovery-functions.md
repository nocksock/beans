---
# beans-7qgi
title: Remove unused launcher discovery functions
status: completed
type: task
priority: normal
tags:
    - technical-debt
    - quick-win
created_at: 2025-12-15T12:48:32Z
updated_at: 2025-12-15T15:42:07Z
parent: beans-gl8t
---

Remove `isCommandAvailable()` and `resolveCommand()` functions from `internal/tui/launcher.go`. These are legacy functions from when discovery filtered by availability. Now all launchers are shown regardless of availability (checked at execution time).

## Files affected
- internal/tui/launcher.go

## Before removing
- Check if used in tests
- Verify no other code depends on these functions
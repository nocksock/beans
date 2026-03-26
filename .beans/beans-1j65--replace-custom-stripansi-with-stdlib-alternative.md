---
# beans-1j65
title: Replace custom stripAnsi with stdlib alternative
status: completed
type: task
priority: high
tags:
    - technical-debt
    - quick-win
created_at: 2025-12-15T12:48:29Z
updated_at: 2025-12-15T17:46:16Z
parent: beans-gl8t
---

Replace the custom `stripAnsi()` function in `internal/tui/modal.go:134-152` with `github.com/charmbracelet/x/ansi.Strip()` which is already in go.mod.

## Benefits
- More robust ANSI handling (handles all escape sequences)
- Reduces maintenance burden (19 lines of custom code removed)
- Uses tested, well-maintained library

## Files affected
- internal/tui/modal.go (2 call sites at lines 86, 117)

## Implementation
Replace all `stripAnsi(s)` calls with `ansi.Strip(s)` after importing `github.com/charmbracelet/x/ansi`
---
# beans-fjw5
title: '[BREAK] Consolidate duplicate command creation logic'
status: completed
type: task
priority: high
tags:
    - technical-debt
    - breaking
created_at: 2025-12-15T12:48:34Z
updated_at: 2025-12-15T17:45:44Z
parent: beans-gl8t
blocking:
    - beans-9mjd
---

Two identical implementations of exec command creation exist:
1. `internal/launcher/launcher.go:CreateExecCommand()` - Used by LaunchManager
2. `internal/tui/tui.go:createExecCommand()` - Used by single-bean launches

## Changes
Remove the TUI version and always use `launcher.CreateExecCommand()` for both single and multi-bean launches.

## Files affected
- internal/tui/tui.go (remove createExecCommand, update line 544)
- Ensure both code paths use the same environment variables

## Breaking change
- Minor behavioral change in single-bean execution
- Environment variables will be consistent between single/multi
- Update documentation if needed
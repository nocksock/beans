---
# beans-iweo
title: Expand launcher discovery tests with edge cases
status: todo
type: task
priority: normal
tags:
    - test
created_at: 2025-12-15T12:49:30Z
updated_at: 2025-12-15T12:49:30Z
parent: beans-gl8t
---

Expand existing `internal/tui/launcher_test.go` with missing edge cases.

## Test scenarios
- Launcher with empty name/exec (should skip)
- Launchers with duplicate names
- Very long exec commands (1000+ chars)
- Multi-line exec with embedded newlines
- discoverLaunchersForMultiple() filtering
  - Returns empty when no launchers have Multiple: true
  - Returns only launchers with Multiple: true
  - Filters out single-bean launchers
  - Handles mix of single and multi launchers
- appendLaunchersToConfig() edge cases
  - Append when launchers section already exists
  - Append with special characters in description
  - Append preserves existing config content
  - Append with multiline exec scripts
  - Error when config file doesn't exist
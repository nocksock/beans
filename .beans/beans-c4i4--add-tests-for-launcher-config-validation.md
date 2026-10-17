---
# beans-c4i4
title: Add tests for launcher config validation
status: completed
type: task
priority: high
tags:
    - test
    - config
created_at: 2025-12-15T12:49:36Z
updated_at: 2025-12-15T13:26:36Z
parent: beans-gl8t
---

Add tests to `internal/config/config_test.go` for launcher-related config fields.

## Test scenarios
- Launcher.Multiple field default/persistence
- TUIConfig.DisableLauncherWarning field default/persistence
- Multi-line launcher without shebang errors (already tested, expand)
- Launcher with shebang but empty body
- Launcher with Windows line endings (`\r\n`)
- Multiple launchers with same name
- Very long launcher scripts (10KB+)
- Launcher name with special characters
- Launcher exec with null bytes
- Shebang in middle of script (not at start)
- Default values are correct:
  - Multiple defaults to false
  - DisableLauncherWarning defaults to false
- Persists through save/load cycle
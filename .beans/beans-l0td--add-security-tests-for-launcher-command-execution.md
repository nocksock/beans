---
# beans-l0td
title: Add security tests for launcher command execution
status: completed
type: task
priority: critical
tags:
    - security
    - test
created_at: 2025-12-15T12:49:19Z
updated_at: 2025-12-15T17:40:42Z
parent: beans-gl8t
---

Create `internal/launcher/launcher_test.go` with comprehensive security tests for `CreateExecCommand()`.

## Test scenarios (CRITICAL)
- Shell injection via bean title/ID with metacharacters (`;`, `$()`, backticks)
- Path traversal in beans directory
- Malicious shebang lines
- Multi-line scripts with/without shebangs
- Environment variable injection
- Special characters in bean titles (Unicode, emoji)
- Script with null bytes
- Shebang with multiple arguments (`#!/usr/bin/env bash -e`)
- Empty shebang line (`#!/`)
- Single-line command with trailing newline

## Priority
CRITICAL - this is security-sensitive code that executes arbitrary commands. Must ensure bean titles and IDs cannot be used for shell injection.
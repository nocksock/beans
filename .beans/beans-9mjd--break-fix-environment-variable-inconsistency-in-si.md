---
# beans-9mjd
title: '[BREAK] Fix environment variable inconsistency in single-bean launchers'
status: completed
type: bug
priority: high
tags:
    - breaking
created_at: 2025-12-15T12:48:37Z
updated_at: 2025-12-15T12:57:02Z
parent: beans-gl8t
---

Single-bean launchers are missing the `BEANS_DIR` environment variable that multi-bean launchers have.

## Current state
- Multi-bean: BEANS_ROOT, BEANS_DIR, BEANS_ID, BEANS_TASK
- Single-bean: BEANS_ROOT, BEANS_ID, BEANS_TASK (missing BEANS_DIR)

## Fix
Add BEANS_DIR to single-bean launches for consistency.

## Breaking change
- Launchers might depend on current env var set
- Update documentation in README.md
- Consider this will be fixed automatically if we consolidate command creation
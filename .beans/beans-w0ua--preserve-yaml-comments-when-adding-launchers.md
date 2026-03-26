---
# beans-w0ua
title: Preserve YAML comments when adding launchers
status: todo
type: feature
created_at: 2025-12-14T20:17:32Z
updated_at: 2025-12-14T20:17:32Z
---

When adding default launchers to .beans.yml, the current implementation simply appends to the file. This works but doesn't preserve comments or formatting.

## Goal
Improve the appendLaunchersToConfig function to preserve existing comments and formatting when modifying .beans.yml.

## Potential Solutions
1. Use gopkg.in/yaml.v3 with custom marshaling that preserves comments
2. Use github.com/goccy/go-yaml (has built-in comment preservation)
3. Implement smart insertion that detects where to place the launchers section

## Acceptance Criteria
- Comments in .beans.yml are preserved when adding launchers
- Existing formatting is maintained
- Launchers section is inserted in a logical location (e.g., after beans section)

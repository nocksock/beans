---
# beans-agzc
title: 'CLI: create --prefix and update --body-append/--body-replace-*'
status: in-progress
type: feature
priority: normal
size: sm
created_at: 2026-10-23T12:00:40Z
updated_at: 2026-10-23T12:13:35Z
parent: beans-zkj9
---

Add these CLI flags:
- `beans create --prefix`. GraphQL `createBean(prefix)` already exists, so this only needs the flag.
- `beans update --body-append`, `--body-replace-old` and `--body-replace-new`.
- Accept `\n` escapes in body flags (upstream eb5ee19).

Reference: upstream internal/commands/{create,update}.go.
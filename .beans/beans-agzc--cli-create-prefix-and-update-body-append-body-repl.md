---
# beans-agzc
title: 'CLI: create --prefix and update --body-append/--body-replace-*'
status: completed
type: feature
priority: normal
size: sm
created_at: 2026-10-23T12:00:40Z
updated_at: 2026-10-23T12:19:56Z
parent: beans-zkj9
---

Add these CLI flags:
- `beans create --prefix`. GraphQL `createBean(prefix)` already exists, so this only needs the flag.
- `beans update --body-append`, `--body-replace-old` and `--body-replace-new`.
- Accept `\n` escapes in body flags (upstream eb5ee19).

Reference: upstream internal/commands/{create,update}.go.

## Work Log
2026-10-23 14:14: Started; studied upstream eb5ee19, update.go, bodyMod resolver
2026-10-23 14:17: Added bodyMod to GraphQL schema + resolver, UnescapeBody/ReplaceOnce/AppendWithSeparator, CLI flags
2026-10-23 14:17: Manual CLI test in temp project OK; resolver tests added
2026-10-23 14:18: Added cmd tests, updated README, prime prompt, docs/compare.md; go test + vet pass
2026-10-23 14:18: Committed 0f1d234 on task/beans-agzc
2026-10-23 10:15: Verified tests, merged into task/beans-agzc (985864f); dev merge pending

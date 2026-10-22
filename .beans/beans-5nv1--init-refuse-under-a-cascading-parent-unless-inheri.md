---
# beans-5nv1
title: 'init: refuse under a cascading parent unless --inherit'
status: archived
type: task
priority: normal
created_at: 2026-10-18T14:58:54Z
updated_at: 2026-10-19T20:46:15Z
parent: beans-xqrj
---

Running `beans init` inside a directory whose nearest ancestor config has `cascade: true` silently creates a full config that shadows the parent. Instead:

- Without a flag, fail with a validation error pointing at the parent config and the flag.
- With `--inherit`, write a child `.beans.yml` that contains only the settings passed as flags, so everything else keeps cascading from the parent. Validate statuses/types against the parent config. Only create a beans directory when `--path` is given; otherwise the parent's is used.
- `--inherit` without a cascading parent is an error.

## Checklist
- [x] config.CascadingParent(dir) and config.SaveSettings(dir, settings)
- [x] init: detect parent, --inherit flag, sparse write
- [x] tests (cmd + config)
- [x] README cascading section
## Work Log
2026-10-18: Implemented on dev as b0ef95b. Flag is named --inherit. Child beans dir is whatever the written child config resolves to, so a parent without beans.path still gives the child its own .beans (per README rules).

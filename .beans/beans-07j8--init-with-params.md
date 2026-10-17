---
# beans-07j8
title: init with params
status: archived
type: task
priority: normal
created_at: 2026-10-16T14:57:39Z
updated_at: 2026-10-17T14:16:52Z
parent: beans-xqrj
---

so we can

beans init --path ~/path/to/tasks

basically provide a parameter for every field we can set in beans.yml

## Work Log
2026-10-17 10:00: Blocker beans-4wwa not started; went with explicit init flags instead of generic key setter
2026-10-17 10:20: Added --path, --prefix, --id-length, --default-status, --default-type, --prompt-extra, --disable-launcher-warning to `beans init` (validated) + tests + README
2026-10-17 10:25: --path cannot combine with --beans-path; cascade left for follow-up beans-9p2n

## Note from beans-4wwa
`config.SetSetting(path, key, value)` validates dotted keys/values against the config schema and writes them into .beans.yml; init flags could be applied through it.

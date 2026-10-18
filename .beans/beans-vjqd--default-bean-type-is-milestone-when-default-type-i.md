---
# beans-vjqd
title: Default bean type is milestone when default_type is unset
status: todo
type: bug
created_at: 2026-10-18T09:22:28Z
updated_at: 2026-10-18T09:22:28Z
---

Found while testing beans-tbbz.

`Config.applyDefaults` (internal/config/config.go) sets `Beans.DefaultType = DefaultTypes[0].Name`, which is `milestone` since milestone moved to the front of DefaultTypes. `config.Default()` says `task`, but configs loaded from a file without `default_type` get milestone.

Repro: a .beans.yml with only `beans: {prefix: x-}`, then `beans create "t" --parent <id>` → 'milestone beans cannot have a parent'. `beans create "t"` silently creates a milestone.
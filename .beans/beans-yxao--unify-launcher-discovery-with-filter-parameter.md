---
# beans-yxao
title: Unify launcher discovery with filter parameter
status: todo
type: task
priority: normal
tags:
    - technical-debt
    - refactoring
created_at: 2025-12-15T12:49:05Z
updated_at: 2025-12-15T12:49:05Z
parent: beans-gl8t
---

`discoverLaunchers()` and `discoverLaunchersForMultiple()` share the same loop pattern with slight filtering variation.

## Solution
Add filter function parameter to `discoverLaunchers()`:

```go
func discoverLaunchers(cfg *config.Config, beansRoot string, filterFn func(*config.Launcher) bool) []launcher {
    var launchers []launcher
    for _, lc := range cfg.Launchers {
        if lc.Name == "" || lc.Exec == "" {
            continue
        }
        if filterFn != nil && !filterFn(&lc) {
            continue
        }
        launchers = append(launchers, launcher{...})
    }
    return launchers
}
```

## Usage
```go
discoverLaunchersForMultiple(cfg, beansRoot) {
    return discoverLaunchers(cfg, beansRoot, func(lc *config.Launcher) bool {
        return lc.Multiple
    })
}
```
## Code review 2026-10-17 (beans-9pxj)

Still valid. Add `multiple bool` to launcher struct to drop name re-lookup (tui/launcher.go:47-61, O(n·m), wrong with duplicate names); remove unused beansRoot param. Absorbs beans-iweo (test discoverLaunchersForMultiple). Also fix: foreground `multiple:true` launcher runs only beanIDs[0] silently (tui.go:762-765) — require background or notify.

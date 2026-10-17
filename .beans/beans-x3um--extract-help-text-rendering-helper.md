---
# beans-x3um
title: Extract help text rendering helper
status: scrapped
type: task
priority: low
tags:
    - technical-debt
    - refactoring
created_at: 2025-12-15T12:49:01Z
updated_at: 2026-10-17T14:02:52Z
parent: beans-gl8t
---

Pattern `helpKeyStyle.Render(key) + " " + helpStyle.Render(action)` appears 4+ times.

## Create helper
```go
func renderHelpText(keys ...struct{ key, action string }) string {
    var parts []string
    for _, k := range keys {
        parts = append(parts, helpKeyStyle.Render(k.key)+" "+helpStyle.Render(k.action))
    }
    return strings.Join(parts, "  ")
}
```

## Usage
```go
help := renderHelpText(
    {"enter", "select"},
    {"/", "filter"},
    {"esc", "cancel"},
)
```
## Code review 2026-10-17 (beans-9pxj)

Scrapped: hint()/hints() exist (keys.go:577-591); remaining hardcoded 'any key dismiss' disappears with beans-5c6x.

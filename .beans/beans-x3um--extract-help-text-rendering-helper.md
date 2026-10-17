---
# beans-x3um
title: Extract help text rendering helper
status: todo
type: task
priority: low
tags:
    - technical-debt
    - refactoring
created_at: 2025-12-15T12:49:01Z
updated_at: 2025-12-15T12:49:01Z
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
---
# beans-u7ao
title: Debug launcher not launching issue
status: completed
type: bug
priority: normal
created_at: 2026-03-26T08:51:48Z
updated_at: 2026-03-26T13:46:21Z
---

---
# beans-u7ao
title: Debug launcher not launching issue
status: completed
type: bug
priority: normal
created_at: 2026-03-26T08:51:48Z
updated_at: 2026-03-26T14:48:00Z
---

## Context

When pressing `!` to launch a bean, nothing happened when no bean was explicitly selected in the list view.

## Root Cause

The `!` key handler in list.go only worked if:
1. Beans were in multi-select mode (space bar selected), OR
2. The cursor was on a bean item

If the list was empty or cursor wasn't on a bean, `!` silently did nothing.

## Fix

Added fallback in `list.go` to use the first bean in the list when no selection:

```go
} else {
    // No bean selected - use first bean in list if available
    items := m.list.Items()
    if len(items) > 0 {
        if item, ok := items[0].(beanItem); ok {
            return m, func() tea.Msg {
                return openLauncherPickerMsg{...}
            }
        }
    }
}
```

## Work Log

2026-03-26 09:45: Added unit tests for launcher flow
2026-03-26 14:48: Fixed list.go to fallback to first bean when no selection
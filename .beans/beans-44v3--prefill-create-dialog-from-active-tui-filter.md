---
# beans-44v3
title: Prefill create dialog from active TUI filter
status: archived
type: task
priority: normal
created_at: 2026-10-18T09:24:27Z
updated_at: 2026-10-19T20:46:15Z
blocked_by:
    - beans-kumc
---

While a filter query (beans-kumc) is active in the TUI, creating a new bean prefills its fields from the filter, e.g. `size:sm` prefills size=sm.

- Applies to include tokens (`status`, `type`, `priority`, `size`, `tag`). Excluded (`-field:x`) and `sort:` tokens are ignored.
- If the same field appears more than once (`size:sm size:lg`), the last value wins.
- Tags: all `tag:` values are added to the new bean and shown read-only in the form header (decided with user).

## Work Log

2026-10-18 11:41: Implemented in task/beans-kumc branch (query.prefill, createModal.withPrefill, tags in CreateBeanInput)

---
# beans-6a8o
title: filter items based on tags
status: todo
type: task
priority: normal
created_at: 2025-12-15T13:10:32Z
updated_at: 2026-10-17T12:13:39Z
parent: beans-7jm4
blocking:
    - beans-4sfw
---

currently it only filters based on title.
set filter either by explicitly selecting (accessible via TAB in search mode)
or using #tag syntax.

when search with tag is active, those tags will automatically be added any task created.

## Update (beans-kumc, beans-44v3)

Mostly covered now: the TUI `/` filter supports `tag:x` (and `-tag:x`) alongside other `field:value` tokens, and creating a bean while a filter is active adds its `tag:` values automatically (beans-44v3). Not done: `#tag` shorthand and TAB-to-pick-tag inside the filter input. Decide whether those are still wanted or scrap this bean.

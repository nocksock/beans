---
# beans-kumc
title: make the filter more usable
status: archived
type: task
priority: normal
created_at: 2026-10-17T14:50:12Z
updated_at: 2026-10-19T20:46:15Z
---

Make the TUI `/` filter understand `field:value` tokens and sorting, instead of only fuzzy-matching title + ID.

## Query syntax

Whitespace-separated tokens typed after `/`:

- `field:value` for `status`, `type`, `priority`, `size`, `tag`
  - same field repeated = OR (`size:sm size:lg`), different fields = AND
  - `-` prefix excludes (`-status:completed`)
- `sort:<field>[:asc|desc]` with fields `status`, `priority`, `type`, `size`, `created`, `updated`, `title`, `id`
  - siblings are sorted per tree level
  - default direction: desc for `created`/`updated`, asc otherwise
- all other words: fuzzy match on title + ID (as today)
- unknown fields/values are ignored and shown as a warning hint in the filter bar (no hard error)

## Design

- `internal/tui/query.go`: pure `parseQuery(input, cfg) -> Query{Filter *model.BeanFilter, Text, Sort, Warnings}`, with table-driven tests
- `list.go`: replace bubbles' built-in filtering with our own `textinput`; re-parse on each keystroke; `loadBeans` merges `Query.Filter` with the archived exclusion via GraphQL, then applies text match + sort before `ui.BuildTree`, so ancestors stay visible as context
- move sort comparators from `cmd/list.go` into `internal/bean` so CLI and TUI share them
- the tag picker (`FilterByTag`) inserts `tag:x` into the query instead of keeping separate `tagFilter` state; `Esc` clears the query
- update help/footer hints

## Checklist

- [x] shared sort helpers in `internal/bean` (+ tests), CLI uses them
- [x] `parseQuery` + tests
- [x] custom filter input in list view, wired into `loadBeans`
- [x] text match + sort applied to tree
- [x] tag picker writes `tag:x` into query; remove `tagFilter`
- [x] warnings + help hints

## Work Log

2026-10-18 09:19: Planned with user: TUI-only, field:value (OR same / AND different / - exclude), sort:field[:dir] token, own input -> GraphQL BeanFilter
2026-10-18 11:24: Added bean.SortBy shared by CLI; follow-up beans-44v3 for create prefill
2026-10-18 11:29: Implemented parseQuery, query input in list, sort, tag picker token, README/help; all tests pass
2026-10-18 11:41: Also implemented follow-up beans-44v3 (create prefill from filter) in this branch

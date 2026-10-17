---
# beans-c05a
title: 'additional status: review -> completed -> merged'
status: in-progress
type: task
priority: normal
created_at: 2025-12-15T15:44:08Z
updated_at: 2026-10-17T12:52:04Z
parent: beans-xqrj
---

we should probably allow defining status via .beans.yml
but grouped by semantic

draft
todo
in-progress
completed
archived

are required builtins 

draft: planning
in-progress: drafting, planning, implementing, reviewing
archived: scrapped

so user can define custom states, but they must fit into one of the og 5
filter in-progress would show all substates.

## Decisions

- Map form in `.beans.yml`, keyed by builtin status (the parent/category):
  ```yaml
  statuses:
    in-progress:
      - name: review
        color: magenta      # optional, defaults to parent's color
        description: ...
    completed:
      - name: merged
  ```
- Categories are whatever is in `DefaultStatuses` (data-driven, so beans-ugfx's `archived` builtin works too)
- Custom statuses sort right after their parent; archive flag inherited from parent
- Filtering by a builtin (`status`/`excludeStatus`) includes its custom substatuses; filtering by a custom status is exact
- GraphQL `Bean.statusCategory: String!`

## Plan

- [ ] config: parse/validate `statuses` map, resolved status list, `StatusCategory`, `ExpandStatuses`
- [ ] replace `config.DefaultStatuses` usages with `cfg.Statuses()` (cmd create/update/check/prime, TUI picker/create modal)
- [ ] GraphQL: expand status filters; add `statusCategory` field; codegen
- [ ] `beans list` hidden-by-default expands to substatuses
- [ ] prime prompt lists custom statuses with their category
- [ ] tests + docs

## Work Log

2026-10-17 14:00: Design agreed with user (map form, statusCategory field), plan written

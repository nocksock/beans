---
# beans-h0g0
title: Aggregated view across cascading .beans.yml subtrees
status: todo
type: feature
priority: low
created_at: 2026-10-17T10:41:37Z
updated_at: 2026-10-17T12:13:39Z
parent: beans-c8p4
---

Phase 2 of beans-jzty. Running beans at a root also discovers descendant .beans.yml files; each subtree's beans are loaded with that subtree's merged (cascaded) config (prefix, path).

Related: beans-bgih (recurse into subfolders).

## Open questions
- ID uniqueness across prefixes/subtrees?
- Where does `create` put a new bean when run at the root?
- TUI grouping/filtering by subtree?
- Interaction with beans-bgih recursion within a single beans path
## Update from beans-tbbz (2026-10-18)
Partly addressed for configs sharing one beans dir:
- Lists in a nested cascading config are scoped to its prefix (`Config.Scope()`, GraphQL `BeanFilter.idPrefix`, `beans list --all`, TUI `g s`); the cascade root lists everything.
- Create at the root: the TUI create dialog offers the prefixes of descendant configs sharing the beans dir (`Config.AvailablePrefixes()`, `CreateBeanInput.prefix`), with minimal downward config discovery.
- Still open here: subtrees with their *own* beans dirs (loading several dirs at once).

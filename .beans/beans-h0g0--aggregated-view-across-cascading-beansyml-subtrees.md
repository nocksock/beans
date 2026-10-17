---
# beans-h0g0
title: Aggregated view across cascading .beans.yml subtrees
status: todo
type: feature
priority: low
created_at: 2026-10-17T10:41:37Z
updated_at: 2026-10-17T12:08:03Z
---

Phase 2 of beans-jzty. Running beans at a root also discovers descendant .beans.yml files; each subtree's beans are loaded with that subtree's merged (cascaded) config (prefix, path).

Related: beans-bgih (recurse into subfolders).

## Open questions
- ID uniqueness across prefixes/subtrees?
- Where does `create` put a new bean when run at the root?
- TUI grouping/filtering by subtree?
- Interaction with beans-bgih recursion within a single beans path
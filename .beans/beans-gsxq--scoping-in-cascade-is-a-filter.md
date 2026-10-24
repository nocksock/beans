---
# beans-gsxq
title: scoping in cascade is a filter
status: completed
type: task
priority: normal
created_at: 2026-10-24T11:46:31Z
updated_at: 2026-10-24T11:52:54Z
---

we need to be able to filter by prefix and then when running beans tui have it start with that filter set if it's part of a cascade. we can then also clear the filter to see all tasks in that cascade.

## Work Log
2026-10-24 13:52: Added `prefix:` TUI query token; TUI starts with `prefix:<scope>` in a cascade; `esc` clears it and `g s` toggles it. Docs and tests updated.
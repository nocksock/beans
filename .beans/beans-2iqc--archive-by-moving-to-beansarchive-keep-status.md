---
# beans-2iqc
title: Archive by moving to .beans/archive/, keep status
status: todo
type: feature
size: lg
created_at: 2026-10-23T12:00:40Z
updated_at: 2026-10-23T12:00:40Z
parent: beans-zkj9
blocked_by:
    - beans-zq5t
---

Breaking: this replaces the status: archived approach (9b537f6). Moving archived beans instead keeps their status, so we still know whether one was completed or scrapped.

- Move archived beans to .beans/archive/ and keep their status. Unarchive moves them back.
- Remove the built-in `archived` status. Keep a way to show archived beans (`A` in the TUI, a list flag).
- Migration: for beans with status: archived, restore the last status before `archived` from git history (internal/githistory). Fall back to `completed`.
- Reference: upstream pkg/beancore/core.go Archive/Unarchive/isArchivedPath (284bf3b).
- Related open beans: beans-p0mg (archive feedback) and beans-9p4x (Y bulk-archive filter bug).
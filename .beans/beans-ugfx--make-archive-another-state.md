---
# beans-ugfx
title: make archive another state
status: in-progress
type: task
priority: normal
created_at: 2026-10-17T12:39:16Z
updated_at: 2026-10-17T12:53:04Z
---

Archiving currently **deletes** completed/scrapped beans (and strips incoming links). Instead, keep them and move them to a new `archived` status that is hidden in list views by default.

## Decisions

- New hardcoded status `archived` (gray, `Archive: true`). Archiving overwrites completed/scrapped → archived (outcome is not preserved).
- Archive scope unchanged: `beans archive` + list `Y` archive all completed/scrapped; detail `Y` archives current bean (confirm if unfinished).
- Links (parent/blocking/blocked-by) are **kept**; archived blockers don't block (via `Archive: true`).
- Hidden by default in `beans list` and the TUI list; TUI gets a toggle to show them.
- `beans delete <id>` remains the way to actually delete.

## Plan

- [x] `internal/config/config.go`: add `{Name: "archived", Color: "gray", Archive: true, ...}` to `DefaultStatuses` (last, so it sorts after done states)
- [x] `internal/ui/styles.go`: include `archived` in the completed/scrapped style cases
- [x] `cmd/archive.go`: set status to `archived` via GraphQL `updateBean` mutation instead of deleting; skip beans already archived; drop link-removal + external-link warning; update Short/Long help text
- [x] `internal/tui/tui.go`: `archiveCompletedMsg` collects completed/scrapped only (not already archived); `archiveConfirmedMsg` updates status instead of `DeleteBean`; adjust confirm messages ("will be archived" not "deleted"); return to the list afterwards (archived bean is hidden there)
- [x] `cmd/list.go`: add `archived` to `hiddenByDefaultStatuses` (`--include archived` / `--status archived` show them); update help
- [x] TUI list: hide `archived` by default (excludeStatus in `loadBeans`); add `A` key to toggle showing archived, show indicator in title/footer, document in `help.go`
- [x] Tree context: decide that archived ancestors still render as context for non-archived children (BuildTree uses allBeans – keep that)
- [x] `cmd/prompt.tmpl`: update `beans archive` description (no longer deletes) and example queries' `excludeStatus` to include `archived`
- [x] `internal/graph/schema.graphqls`: update docs mentioning "(completed, scrapped)" → include archived; run `mise codegen`
- [x] `cmd/roadmap.go`: verify archived handled by `IsArchiveStatus` (should be automatic)
- [x] Tests: config status list, `defaultExcludedStatuses`, archive command (status change, links preserved, idempotent), TUI archive flow, TUI hide/toggle
- [x] README: no status list there, nothing to change

## Work Log
2026-10-17 12:45: Plan agreed with user, written to bean
2026-10-17 12:55: Added archived status (config, styles) + IsArchivable helper
2026-10-17 13:00: beans archive sets status archived via GraphQL, keeps links; test added
2026-10-17 13:05: beans list hides archived by default (--include archived)
2026-10-17 13:10: TUI hides archived, A toggles, Y/YY archive via status update; tests updated
2026-10-17 13:15: Updated prompt.tmpl + schema docs, codegen, full test suite green, manual CLI check ok
2026-10-17 13:30: Review fixes: hide archived parents, child counts, footer label, stay in detail
2026-10-17 13:32: Follow-ups: beans-j5a5, beans-9p4x, beans-p0mg

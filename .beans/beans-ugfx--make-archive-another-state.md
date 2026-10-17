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

- [ ] `internal/config/config.go`: add `{Name: "archived", Color: "gray", Archive: true, ...}` to `DefaultStatuses` (last, so it sorts after done states)
- [ ] `internal/ui/styles.go`: include `archived` in the completed/scrapped style cases
- [ ] `cmd/archive.go`: set status to `archived` via GraphQL `updateBean` mutation instead of deleting; skip beans already archived; drop link-removal + external-link warning; update Short/Long help text
- [ ] `internal/tui/tui.go`: `archiveCompletedMsg` collects completed/scrapped only (not already archived); `archiveConfirmedMsg` updates status instead of `DeleteBean`; adjust confirm messages ("will be archived" not "deleted"); stay in detail view / keep history since beans still exist
- [ ] `cmd/list.go`: add `archived` to `hiddenByDefaultStatuses` (`--include archived` / `--status archived` show them); update help
- [ ] TUI list: hide `archived` by default (excludeStatus in `loadBeans`); add `A` key to toggle showing archived, show indicator in title/footer, document in `help.go`
- [ ] Tree context: decide that archived ancestors still render as context for non-archived children (BuildTree uses allBeans – keep that)
- [ ] `cmd/prompt.tmpl`: update `beans archive` description (no longer deletes) and example queries' `excludeStatus` to include `archived`
- [ ] `internal/graph/schema.graphqls`: update docs mentioning "(completed, scrapped)" → include archived; run `mise codegen`
- [ ] `cmd/roadmap.go`: verify archived handled by `IsArchiveStatus` (should be automatic)
- [ ] Tests: config status list, `defaultExcludedStatuses`, archive command (status change, links preserved, idempotent), TUI archive flow, TUI hide/toggle
- [ ] README: mention archived status

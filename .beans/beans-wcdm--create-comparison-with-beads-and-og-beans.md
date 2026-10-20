---
# beans-wcdm
title: create comparison with beads and og beans
status: in-progress
type: task
priority: normal
created_at: 2026-10-20T08:54:39Z
updated_at: 2026-10-20T09:57:25Z
---

compare.md

compare contrast with beads, hmans/beans and my beans fork.
use table

for things that have be rehauled etc compared to hmans/beans create a separate section to compare just with hmans/beans
## Plan

Deliverable: `docs/compare.md` in the code repo (dev branch, own worktree in `.worktrees/`). Neutral, factual tone with tables plus short "when to pick which" notes.

### Research
- [ ] steveyegge/beads: get the current feature set from the web (repo README/docs), and note the date/version checked
- [ ] hmans/beans: read `upstream/main` (README, CLI commands, config, TUI)
- [ ] nocksock/beans fork: read `git log upstream/main..dev` and the docs for each reworked area

### Section 1: three-way table (beads, hmans/beans, fork)
Focus areas, each its own table or row group:
- [ ] Storage & data model: markdown files vs SQLite+JSONL, ID scheme, relationships (parent, blocking, blocked-by), git/merge behaviour
- [ ] Human UX: TUI, CLI ergonomics, filtering/sorting, graph view, editor integration
- [ ] Config & workflow: statuses (custom too), types, size/priority, cascading config, archiving, multi-repo/hierarchy
- Agent integration is not a focus area. It gets at most one or two summary rows.

### Section 2: fork vs hmans/beans
For the areas that were reworked, not just extended, a table with columns *hmans/beans | fork | why/impact*:
- [ ] Launchers: replace session management; `beans launch <name> [id] [-- args]`, builtins (worktree/prompt/agent/tmux/kitty), nested launches
- [ ] TUI: lazygit-style panes/layout engine replace the detail/depnav views; configurable keybinds; keybinds dialog; log pane
- [ ] Config: cascading `.beans.yml`, user-global layer, `beans settings`, `init --cascade/--inherit`
- [ ] Archive by status instead of deleting; list hides completed/scrapped by default
- [ ] Data model additions: blocked-by, size, optional titles, custom statuses
- [ ] Mark breaking changes (`!` commits) explicitly

### Finish
- [ ] Link docs/compare.md from the README
- [ ] Check every claim against the code or sources; give the date checked for beads
- [ ] Commit (`docs: ...`) together with the bean file

## Work Log
2026-10-20 09:00: Agreed scope with user (docs/compare.md, neutral, focus storage/UX/config, web research on beads)

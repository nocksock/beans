---
# beans-zq5t
title: Public pkg/ Go API and multi-binary layout (beans, beans-tui)
status: todo
type: feature
size: xl
created_at: 2026-10-23T12:00:40Z
updated_at: 2026-10-23T12:00:40Z
parent: beans-zkj9
---

Breaking: Go import paths change.

- Move internal/{bean,beancore,config,graph,...} to a public pkg/ API.
- Split the TUI into its own beans-tui binary.
- Leave beans-serve out for now.

Reference: upstream 4900811 (the pkg move) and 84b4afc (the binary split).

Do this early: it touches the files every other bean in this epic touches.
## Planning notes (2026-10-23, paused until usage reset)

How the fork imports its own packages today:
- `internal/bean` imports `config`.
- `internal/beancore` imports `bean`, `config` and `search`.
- `internal/graph` imports `bean`, `beancore`, `config`, `githistory` and `graph/model`.
- `internal/tui` imports `bean`, `beancore`, `config`, `graph`, `launcher`, `tui/layout` and `ui`.
- `cmd/` is a single cobra package, and `main.go` just calls `cmd.Execute()`.

What upstream did, for reference:
- 4900811 moved `bean`, `beancore` and `config` out of `internal/`, and they later ended up under `pkg/`. `search` stayed internal.
- Upstream's `pkg/` now holds `bean`, `beancore`, `beangraph` (the GraphQL resolver, made public), `config`, `forge` and `safepath`. Its `internal/` holds `commands`, `graph`, `tui`, `ui`, `output`, `search` and others.
- Its commands live in `internal/commands`, with `NewRootCmd()`, `RegisterCoreCommands(root)`, `RegisterTuiCmd(root)` and `Execute(root)`.
- `cmd/beans/main.go` registers the core commands. `cmd/beans-tui/main.go` sets `root.Use = "beans-tui"` and runs `tui` when no subcommand is given.
- `beans tui` in the main binary prints a hint to use the new binary instead.

Questions for the next session:
- Should `graph` be public, the way upstream's `pkg/beangraph` is?
- Should `githistory` and `launcher` go into `pkg/`?
- Should `beans tui` keep working, or only print a hint to use beans-tui?
- What about mise build tasks and release/goreleaser for the second binary?

## Work Log
2026-10-23 11:00: Mapped fork import graph and upstream split; paused planning

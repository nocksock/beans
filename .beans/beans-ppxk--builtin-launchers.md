---
# beans-ppxk
title: builtin launchers
status: in-progress
type: task
priority: normal
created_at: 2026-10-19T07:26:37Z
updated_at: 2026-10-19T12:20:48Z
---

move some launchers to scripts/folder in deliverable (embed in binary?)

i'm thinking of cascading them as well

child can call:

    beans launch parent.tmux-window ; of parent beans.yml
    beans launch builtin.tmux-window ; og beans

(also rename from launcher to launch)

so they can be composed somehow in a launcher

writing those with bash feels appropriate, but writing in go might also be worth considering for the builtins


## Plan (agreed 2026-10-19)

### CLI (breaking, `feat!`)

    beans launch <name> [id] [-- args...]
    beans launch --list [--json]

- `<name>` is positional; `--launcher` is removed.
- Bean ID: `[id]` arg → `$BEANS_ID` → first line of stdin (so nested calls need no ID).
- Args after `--` reach the script as `"$@"` (`sh -c "$script" <name> args…`; builtins via `bash -c`).
- Config key `launchers:` and the TUI stay. TUI picker shows configured launches only; `--list` shows builtins in a separate section.

### Name resolution

- `X` → merged config, else `builtin.X`.
- `builtin.X` → always the embedded script.
- `parent.X` → X from the nearest config layer *above* the layer that declared the calling launch, else `builtin.X`. Config keeps per-layer launcher definitions for this.

### Nested-call environment (read only by `beans launch`)

- `BEANS_BIN`: path of the running binary; scripts call `"$BEANS_BIN" launch …`.
- `BEANS_LAUNCH_CONFIG`: config file, so nested calls from another cwd (worktree) use the same config.
- `BEANS_LAUNCH_LAYER`: dir of the calling launch's layer, for `parent.`.
- `BEANS_LAUNCH_DEPTH`: recursion guard, error past 16 (e.g. a `prompt` override calling `prompt` instead of `parent.prompt`).

### Builtins (embedded bash, `internal/launcher/builtin/*.sh`, `go:embed`)

| name | usage |
|---|---|
| `tmux-window` | `[--cwd DIR] [--name N] -- cmd…` new tmux window running cmd |
| `tmux-session` | `[--cwd DIR] [-- cmd…]` create or switch to a session per bean |
| `kitty-window` | `[--cwd DIR] -- cmd…` |
| `worktree` | `[--base BRANCH] [--dir DIR]` ensure git worktree on `task/<id>`, print path; defaults: current branch, `$BEANS_ROOT/.worktrees/<id>` |
| `prompt` | `[--tmux] [--worktree DIR]` `beans show --prompt` + generic tmux/draft notes and a worktree note without hub-specific merge steps |
| `agent` | `<claude\|opencode\|crush>`, prompt from `"$BEANS_BIN" launch prompt` (unqualified, so it can be overridden) |

Builtins call each other by unqualified name so a project can override single pieces.

### Out of scope (follow-up bean)

Migrate the hub `.beans.yml` + `scripts/` to the builtins: override `prompt` (calls `parent.prompt`, adds dev-merge cleanup) and `worktree` (`--base dev --dir "$BEANS_ROOT/$BEANS_ID"`, sparse-checkout); remove `bean-agent`, `bean-launch`, `tmux-session-run`.

### Tasks

- [ ] CLI: positional name, ID fallback chain, `--` args passthrough
- [x] Config: keep per-layer launcher definitions
- [x] Resolver: `X` / `builtin.X` / `parent.X`
- [ ] Nested-call env (`BEANS_BIN`, `BEANS_LAUNCH_CONFIG`, `BEANS_LAUNCH_LAYER`, `BEANS_LAUNCH_DEPTH`) + recursion guard
- [x] Embedded builtins: tmux-window, tmux-session, kitty-window, worktree, prompt, agent
- [ ] `--list` builtin section (text + JSON)
- [ ] Docs (README / launcher docs, `beans prime` if it mentions `--launcher`)
- [ ] Tests: resolution table, ID order, `$@` passthrough, recursion guard, builtin smoke tests with stubbed tmux/kitty/agents, worktree on temp repo

## Work Log

2026-10-19 10:45: Planned with user; plan written into bean
2026-10-19 21:41: Task 1 done: config keeps per-layer launchers (Launcher, ParentLauncher, LoadedPath, reserved names), c732fc7
2026-10-19 21:42: Task 2 done: Launch carries Name/Args/Builtin/ConfigPath/Depth, nested-call env, $0/$@ passing, ed8dba4
2026-10-19 21:43: Task 3 done: builtin registry + tmux-window, tmux-session, kitty-window scripts
2026-10-19 21:44: Task 4 done: worktree builtin, 482383a
2026-10-19 21:44: Task 5 done: prompt and agent builtins, 0b7a1a3
2026-10-19 21:44: Review fix: ParentLauncher empty layerDir resolves above leaf, 00d280b
2026-10-19 21:45: Task 6 done: name resolution, ceb2fa7

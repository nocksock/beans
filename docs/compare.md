# beads vs. beans vs. this fork

How this fork ([nocksock/beans](https://github.com/nocksock/beans)) compares to the tool it forked, [hmans/beans](https://github.com/hmans/beans), and to [beads](https://github.com/gastownhall/beads) (`bd`, formerly `steveyegge/beads`).

**Versions compared** (checked 2026-10-22):

| Project | Version |
|---|---|
| beads | v1.3.1 (2026-09-30) |
| hmans/beans | `upstream/main` at `99260bf` (2026-04-06) |
| fork | `dev` at `4665cfb` |

The fork split from upstream at `76dc94a` (2025-12-14) and has not merged upstream since. Since the split the fork has added about 120 commits and upstream about 440, so both have moved a long way from the common base. Treat them as related but separate tools.

## At a glance

- **beads** stores issues in a database. It is built for many agents working at once, with claiming, messaging, molecules, compaction and dozens of relationship types. There is no official UI, but the community has built many.
- **hmans/beans** stores issues as Markdown files and is growing towards a web app. It has `beans serve` with a SvelteKit UI, agent chat, worktree workspaces and PR integration.
- **This fork** also stores issues as Markdown files, but stays in the terminal. It has a lazygit-style TUI, cascading config for monorepos, custom statuses, and launchers that hand beans to tmux, kitty, worktrees and agents.

## Three-way comparison

### Storage & data model

| | beads | hmans/beans | fork |
|---|---|---|---|
| Storage | Dolt (versioned SQL). Embedded by default; server mode for multiple writers | Markdown + YAML frontmatter in `.beans/` | Markdown + YAML frontmatter in `.beans/` |
| Source of truth | Dolt database. `issues.jsonl` is only an export | The files | The files |
| Daemon | No (removed in v0.59/0.60) | No (`beans serve` is optional, for the web UI) | No |
| Sync / merge | `bd dolt push/pull` to `refs/dolt/data`. Dolt merges cell by cell; `bd doctor --fix` resolves conflicts | Plain git; conflicts are text conflicts in single files | Plain git; conflicts are text conflicts in single files |
| Concurrent writes | Atomic `bd ready --claim`, leases | ETags (`update --if-match`, `beans.require_if_match`) | Nothing built in |
| IDs | Hash IDs with a prefix (`bd-a1b2`); children get hierarchical IDs (`bd-a1b2.1`) | NanoID with a prefix (`beans-x7k2`), configurable length | Same as upstream; ambiguous partial IDs raise an error listing the matches |
| Statuses | open, in_progress, blocked, deferred, closed, pinned, hooked; you can add custom ones in a category | draft, todo, in-progress, completed, scrapped | Upstream's set + `archived`; you can add custom statuses grouped under a built-in one |
| Types | bug, feature, task, epic, chore, decision (+ internal types); you can add custom ones | milestone, epic, feature, bug, task | Same as upstream |
| Priority | 0–4 | critical, high, normal, low, deferred | Same as upstream |
| Estimates | `estimated_minutes`, due date, defer date | — | `size`: xs / sm / md / lg / xl |
| Title | Required | Required | Optional (a snippet of the body is shown instead) |
| Manual ordering | — | `order` (fractional index, for drag & drop) | — |
| Custom fields | Free-form `metadata` JSON | — | — |
| Relationships | ~20 types: blocks, conditional-blocks, waits-for, parent-child, related, discovered-from, duplicates, supersedes, … | parent, blocking, blocked_by. A parent's blocked state and status also apply to its children | parent, blocking, blocked_by (explicit only) |
| Closed work | Close/reopen; `bd delete` is permanent; old closed issues get summarised (compaction) | `beans archive` moves files to `.beans/archive/`, where they stay loaded as project memory | `beans archive` sets `status: archived` in place and keeps the links |
| History | Dolt commit history; `bd versions` (not yet released) | Git (no special support) | GraphQL `history` / `diff` per bean, following renames by ID |

### Human UX

| | beads | hmans/beans | fork |
|---|---|---|---|
| Official UI | None. Many community TUIs (perles, LazyBeads, b9s, …), web UIs (beads-ui, …) and editor plugins | Web UI (`beans serve`, SvelteKit) + TUI with list/detail views | TUI with panes (list, details, dependency graph, log) |
| Key bindings | n/a | Hard-coded | All configurable, with chords (`"g s"`) and `<disabled>`; `?` opens a searchable keybinds dialog |
| Filtering | `bd query` language: `field=value`, `AND/OR/NOT`, relative dates | `/` text filter, filter by tag | TUI query: `login type:bug size:sm -status:completed sort:created` |
| "What's next" | `bd ready` (`--claim`, `--explain`), `bd blocked`, `bd stale` | `beans list --ready` | `beans list --unblocked` |
| Default list | Open issues | No default filter | Hides completed, scrapped and archived beans (`--include` to show them) |
| Graph view | `bd graph`: terminal DAG, `--dot`, interactive `--html`; `bd dep tree` | Tree in the list | `beans graph` (DOT or `--ascii`); in the TUI, a dependency pane you can move around in |
| Editing | `bd edit` ($EDITOR, per field), `bd create-form` | `e` opens $EDITOR; `update --body-append/--body-replace-*` | `e` edit popup, `E` $EDITOR; create form with field-jump keys and type-ahead; `update --body-append/--body-replace-*` |
| Monorepo scoping | Route new beads to other repos; `external:` deps | — | Cascading config; lists are scoped to the nearest config's prefix (`g s` / `--all` to widen) |

### Config & workflow

| | beads | hmans/beans | fork |
|---|---|---|---|
| Config files | `config.yaml` (user → repo → local) + settings stored in the database (`bd config set`) | `.beans.yml` | `.beans.yml`, cascading up the tree, plus a user-global `~/.config/beans/config.yml` |
| Editing config | `bd config set` | Edit the file | `beans settings get/set` (keeps comments) or edit the file |
| `init` | Sets up the database, hooks and agent files | `beans init` | `beans init` with flags for every setting; `--cascade`, `--inherit`; won't overwrite without `--force` |
| Workflow templates | Formulas → protos → molecules; wisps; gates | — | — |
| Running work | Multi-agent: `bd assign`, merge slots, `bd swarm`, `bd mail` | Web UI: worktree workspaces, agent chat (Plan/Act), terminals, Run / Integrate / PR | Launchers: shell commands run on one or more beans, plus builtins for tmux, kitty, worktrees, prompts and agents (see below) |
| Language / licence | Go, MIT | Go (+ SvelteKit), Apache-2.0 | Go, Apache-2.0 |

Agent integration is about the same in all three: a `prime` command to give the agent context, `--json` everywhere, and a Claude Code plugin or hooks. beads also has an MCP server (`beads-mcp`), more agent setups (`bd setup <agent>`) and stored memories. Both beans variants answer queries through GraphQL (`beans query`), and neither has an MCP server.

### When to pick which

- **beads**: you run many agents at once and need claiming, rich relationships, workflow templates and compaction. You are fine with a database instead of readable files.
- **hmans/beans**: you want plain Markdown files and a browser UI that manages worktrees, agent sessions and PRs for you.
- **fork**: you want plain Markdown files and do everything in the terminal. You work in monorepos, or you want to put your own tmux/agent scripts behind a key.

## Fork vs. hmans/beans: reworked areas

These are areas where the fork has replaced upstream's behaviour, not only added to it. ⚠️ marks a change that breaks compatibility with upstream data or habits.

| Area | hmans/beans | fork | Impact |
|---|---|---|---|
| ⚠️ Archiving | Moves the file to `.beans/archive/` | Sets `status: archived` in place (`Y` in the TUI, `A` to show archived beans) | The two layouts don't match: the fork doesn't read upstream's `archive/` folder as archived beans, and upstream doesn't know the `archived` status |
| ⚠️ `beans list` default | No default filter | Hides completed, scrapped and archived beans | Scripts that expect every bean need `--include` |
| TUI layout | List view + full-screen detail view | Panes like lazygit: list, details, dependency graph and launcher log; layout presets (`+`/`-`), focus by number keys, a single pane on narrow terminals | The full-screen detail view is gone; you navigate between panes instead |
| Key bindings | Hard-coded; `?` shows a help overlay | Every key configurable per context in `keybinding:`; `?` opens a searchable dialog where `enter` runs the key | Both copy the ID with `y`; the fork archives with `Y` |
| Config | One `.beans.yml` per project | Cascading `.beans.yml` (parent sets `cascade: true`; launchers and other named lists merge by `name`) + user-global layer + `beans settings` | One config can be shared across a monorepo; per-user keybinds and launchers |
| Statuses | Fixed set | Fixed set + `archived` + custom statuses (e.g. `review` under `in-progress`) that behave like their built-in parent | Statuses that fit your workflow, without breaking filters |
| Running agents | `beans serve`: worktree workspaces, agent chat, PR flow in the browser | Launchers: `!` in the TUI, run on several selected beans or in the background (output goes to the log pane), `beans launch <name> [id] [-- args]`, builtins `tmux-window`, `tmux-session`, `kitty-window`, `worktree`, `prompt`, `agent`, which can call each other | Same goal, opposite approach: composable shell scripts instead of a built-in app |
| Bean fields | `title` required; `order` | `title` optional; `size`; no `order` | Upstream may misread fork beans that have no title; `size` is ignored upstream |
| Relationships | Explicit + inherited from the parent (a parent's blocked state and status apply to its children) | Explicit only | Blocked-ness is computed differently |
| Git | Worktree file changes and branch status, for the web UI | History and diffs per bean through GraphQL | — |
| Agent prompts | `beans prime` | `beans prime` + `beans show --prompt` (with `prompt.extra`) | Prompts for each bean, used by the `prompt`/`agent` builtins |

### Upstream features the fork lacks

The fork doesn't have these changes, all made upstream since the split:
- The web UI and `beans serve`, plus the multi-binary layout (`beans`, `beans-serve`, `beans-tui`).
- The public `pkg/` Go API.
- Inherited blocking and status (the "implicit" blocking and status above).
- ETags and `--if-match`.
- `list --ready`, `show --etag-only`.
- The opencode plugin.

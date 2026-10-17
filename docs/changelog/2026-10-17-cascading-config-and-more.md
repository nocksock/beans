# What's new in beans: cascading config, custom statuses, rebindable keys

*October 2026*

The last few days of work on beans had one theme: making it fit monorepos and
multi-project setups. A repository can now share one task store and one set of
launchers with every package in it, while each package keeps its own prefix.
Your personal preferences live in one place too. This post covers the main
changes and how to move an existing setup over.

> **Breaking changes ahead.** `beans archive` no longer deletes beans,
> `beans list` hides finished beans by default, and the `session` config section
> is gone. See [Migration checklist](#migration-checklist).

---

## Cascading `.beans.yml` for monorepos

Until now beans used the nearest `.beans.yml` and nothing else. In a monorepo
that left two options: one config for the whole repo, or a copy of everything in
each package. Now a parent config can pass its settings down to configs in
subdirectories.

```yaml
# repo/.beans.yml
cascade: true
beans:
  path: .beans
  prefix: repo-
launchers:
  - name: claude
    exec: claude "Work on task $BEANS_ID"
```

```yaml
# repo/packages/api/.beans.yml
beans:
  prefix: api-
```

Inside `packages/api`, beans now uses the prefix `api-`, the shared `repo/.beans`
directory and the inherited `claude` launcher.

The rules:

- **The parent opts in.** A config passes settings down only if it sets
  `cascade: true`. The chain keeps going up as long as each parent also
  cascades. `cascade` itself is never inherited.
- **The closest config wins.** Maps merge key by key. Lists of named items, such
  as `launchers`, merge by `name`: a launcher with the same name replaces the
  inherited one, and new names are added. Any other list replaces the inherited
  list.
- **Paths resolve where they were written.** A relative `beans.path` resolves
  against the file that declared it, so the shared `.beans` above stays at
  `repo/.beans`. To give a package its own task store, set `beans.path` in its
  config. If no config sets a path, beans uses `.beans` next to the nearest
  config.
- **`~` works** in `beans.path`, `--beans-path` and `--config`.

### Inherited launchers run where they were defined

A launcher runs in the directory of the config that declared it. A worktree
script defined at the repo root therefore runs at the repo root, even when you
start it from a package. A new environment variable tells the script where that
is:

| Variable | Meaning |
|---|---|
| `BEANS_ROOT` | The project root: the directory of the **nearest** `.beans.yml` |
| `BEANS_CONFIG_DIR` | The directory of the config that **declared** the launcher, and the launcher's working directory |
| `BEANS_DIR` | The beans directory |
| `BEANS_ID`, `BEANS_TASK` | The bean's ID and the path to its file |

> ⚠️ `BEANS_ROOT` used to be the parent of the beans directory. It is now the
> directory of the nearest config. In a package that inherits a shared `.beans`,
> the two differ. Scripts that build paths into the repo should use
> `BEANS_CONFIG_DIR`.

---

## A user-global config layer

Settings that apply to all your projects go in `~/.config/beans/config.yml`
(or `$XDG_CONFIG_HOME/beans/config.yml`). It uses the same format as
`.beans.yml` and sits below every project config, so any project can override
it. It's the right place for your own launchers and key bindings.
`beans.path` can't be set there, and `cascade` has no effect.

So the full stack, from lowest to highest priority, is:

```
~/.config/beans/config.yml       ← you
repo/.beans.yml (cascade: true)  ← the team / monorepo
repo/packages/api/.beans.yml     ← the package
```

The TUI watches every file in that stack and reloads when one changes. You can
also reload by hand with `R`.

---

## Custom statuses, grouped under the builtins

You can add your own statuses. Each one is attached to the builtin status it is
a kind of:

```yaml
statuses:
  in-progress:
    - name: review
      color: purple
      description: Waiting for code review
  completed:
    - name: merged
```

A custom status takes its behaviour from its builtin. `merged` counts as done:
it no longer blocks other beans, it's hidden by default, and `beans archive`
archives it. Filtering by a builtin also matches its custom statuses, so
`beans list -s in-progress` shows `review` beans too. Filtering by a custom
status matches only that status. GraphQL has a new `statusCategory` field that
holds the builtin a bean's status belongs to.

> Statuses written in the old flat list form are now ignored. Move them under a
> builtin status.

---

## Every key is rebindable

Every key in the TUI can be rebound, in a style similar to lazygit's. Put the
bindings in your user config, or in a repo config if the team wants to share
them:

```yaml
keybinding:
  global:
    quit: ctrl+q
  list:
    changeStatus: [x, "g s"]   # several keys; "g s" is a chord
    archive: <disabled>
```

Bindings are grouped by context (`global`, `list`, `detail`, `picker`,
`createModal`, …). Unknown action names are an error. If one key is bound to two
actions, the footer shows a warning. The help overlay and the footers always
show your current bindings. The README has the full list of actions.

---

## Archive instead of delete, and a quieter `beans list`

- **`archived` is a new builtin status.** `beans archive` moves completed and
  scrapped beans to `archived` and keeps their links. To delete beans, use
  `beans delete`.
- **`beans list` hides completed, scrapped and archived beans by default.** Use
  `--include completed,archived` to show them. `--include merged` shows a single
  custom status. An explicit `--status` filter still shows exactly the statuses
  you asked for.
- In the TUI, `Y` archives a bean and `A` shows or hides archived beans.

---

## Blocked-by relationships and `beans graph`

- Beans can now declare a **`blocked_by`** relationship as well as `blocking`.
  The two directions are combined into a single graph, which is used for cycle
  detection and the blocking filters. Use `beans create --blocked-by`,
  `beans update --blocked-by/--remove-blocked-by`, or press `B` in the TUI.
- **`beans list --unblocked`** lists only beans that nothing blocks, so it shows
  what can be worked on now.
- **`beans graph`** prints parent and blocking relationships as Graphviz DOT,
  with `--ascii` output for the terminal and `--type` to choose which kinds of
  relationship to show.

---

## Better input for agents

- **`beans show <id> --prompt`** prints a ready-to-use markdown prompt for an
  agent. It includes the bean, its parent, blockers, children and a short guide
  to tracking progress with the CLI. You can append project-specific
  instructions in config, and those cascade like everything else:

  ```yaml
  prompt:
    extra: |
      Run mise test before committing.
  ```

- **Titles are optional.** A bean without a title is shown with a snippet of its
  body.

---

## TUI improvements

- A two-column layout with a preview pane and tree navigation. `h`/`l` fold and
  unfold in the list; `z`/`Z` toggle one node or all of them.
- In the detail view, `h`/`l` move to the previous and next bean.
- `c` opens a create form with fields for type, status and priority. `C` creates
  a child bean.
- `e` edits a bean in a popup. `E` opens it in `$EDITOR`.
- `<` and `>` raise and lower priority. `T` opens a tag editor.
- On light terminals, markdown now uses the light style.

---

## Built-in session management is gone

The `session` config section, the session poller and the `S` key have been
removed. Sessions such as tmux, zellij or a Claude session are now ordinary
launchers. Define them once in a cascading root config or in your user config,
and run them with `!`.

---

## Migration checklist

For a monorepo that already uses beans:

1. **Choose one task store.** Keep a single `.beans/` at the repo root and add
   `cascade: true` to the root `.beans.yml`.
2. **Thin out the package configs.** Each package config needs only what is
   different for that package, usually just `beans.prefix`. Delete the
   duplicated launchers, statuses and settings. Remove `beans.path` unless the
   package should have its own store.
3. **Move personal settings to the user config.** Your own launchers, key
   bindings and editor preferences go in `~/.config/beans/config.yml`.
4. **Replace the `session` section with launchers.** Rewrite tmux or zellij
   attach and launch commands as `launchers` entries in the root config.
5. **Check launcher scripts for `BEANS_ROOT`.** If a script builds paths into
   the repo (worktrees, for example), it should use `BEANS_CONFIG_DIR`, or the
   launcher should be defined in the config whose directory it expects.
6. **Convert custom statuses** to the grouped form under a builtin status.
7. **Update scripts that call `beans archive`** and expect the files to be
   deleted. Use `beans delete` for that now. Also update scripts and agent
   prompts that rely on `beans list` showing completed beans: add
   `--include completed`.
8. **Run `beans prime` again** in agent hooks so agents learn about the new
   statuses and relationships. If your hooks already run it on session start,
   this happens automatically.

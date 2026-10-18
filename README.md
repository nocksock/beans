![beans](https://github.com/user-attachments/assets/776f094c-f2c4-4724-9a0b-5b87e88bc50d)

[![License](https://img.shields.io/github/license/hmans/beans?style=for-the-badge)](LICENSE)
[![Release](https://img.shields.io/github/v/release/hmans/beans?style=for-the-badge)](https://github.com/hmans/beans/releases)
[![CI](https://img.shields.io/github/actions/workflow/status/hmans/beans/test.yml?branch=main&label=tests&style=for-the-badge)](https://github.com/hmans/beans/actions/workflows/test.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/hmans/beans?style=for-the-badge)](https://go.dev/)
[![Go Report Card](https://goreportcard.com/badge/github.com/hmans/beans?style=for-the-badge)](https://goreportcard.com/report/github.com/hmans/beans)

**Beans is an issue tracker for you, your team, and your coding agents.** Instead of tracking tasks in a separate application, Beans stores them right alongside your code. You can use the `beans` CLI to interact with your tasks, but more importantly, so can your favorite coding agent!

This gives your robot friends a juicy upgrade: now they get a complete view of your project, make suggestions for what to work on next, track their progress, create bug issues for problems they find, and more.

You've been programming all your life; now you get to be a product manager. Let's go! 🚀

## Announcement Trailer ✨

https://github.com/user-attachments/assets/dbe45408-d3ed-4681-a436-a5e3046163da

## Stability Warning ⚠️

Beans is still under heavy development, and its features and APIs may still change significantly. If you decide to use it now, please follow the release notes closely.

Since Beans emits its own prompt instructions for your coding agent, most changes will "just work"; but sometimes, we modify the schema of the underlying data files, which may require some manual migration steps. If you get caught by one of these changes, your agent will often be able to migrate your data for you:

> "The Beans data format has changed. Please migrate this project's beans to the new format."

## Features

- **Track tasks, bugs, features**, and more right alongside your code.
- **Plain old Markdown files** stored in a `.beans` directory in your project. Easy to version control, readable and editable by humans and machines alike!
- Use the `beans` CLI to create, list, view, update, and archive beans; but more importantly, **let your coding agent do it for you**!
- **Supercharge your robot friend** with full context about your project and its open tasks. A built-in **GraphQL query engine** allows your agent to get exactly the information it needs, keeping token use to a minimum.
- A beautiful **built-in** TUI for browsing and managing your beans from the terminal.
- Generates a **Markdown roadmap document** for your project from your data.

## Installation

We'll need to do three things:

1. Install the `beans` CLI tool.
2. Configure your project to use it.
3. Configure your coding agent to interact with it.

Either download Beans from the [Releases section](https://github.com/hmans/beans/releases), or install it via Homebrew:

```bash
brew install hmans/beans/beans
```

Alternatively, install directly via Go:

```bash
go install github.com/hmans/beans@latest
```

## Configure Your Project

Inside the root directory of your project, run:

```bash
beans init
```

This will create a `.beans/` directory and a `.beans.yml` configuration file at the project root. All of it is meant to be tracked in your version control system.

`beans init` takes a flag for each setting in `.beans.yml`, e.g. to keep the beans outside the project:

```bash
beans init --path ~/path/to/tasks --prefix app- --id-length 6
```

Run `beans init --help` for the full list.

From this point onward, you can interact with your Beans through the `beans` CLI. To get a list of available commands:

```bash
beans help
```

But more importantly, you'll want to get your coding agent set up to use it. Let's dive in!

### Reading and Changing Settings

Settings are addressed by dotted keys:

```bash
beans settings get beans.prefix          # effective value, after defaults and inherited configs
beans settings get launchers.claude.exec # list items by name or index
beans settings get                       # the whole effective config
beans settings set beans.id_length 6     # change it in the nearest .beans.yml
```

`set` keeps the rest of the file, including comments, intact. Values are parsed as YAML, so `6` is a number, `true` a boolean and `[a, b]` a list. Config files that take part in cascading (see below) can't be changed with `set` yet.

### Monorepos: Cascading Configuration

beans uses the nearest `.beans.yml` found by walking up from the current directory. A config can pass its settings down to configs in subdirectories with `cascade: true`:

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

Running beans inside `packages/api` uses prefix `api-`, the shared `repo/.beans` directory, and the inherited `claude` launcher.

- **Opt-in by the parent**: only a config with `cascade: true` passes settings down. Chains continue upward while each next config cascades; `cascade` itself is never inherited.
- **Merging**: the closest config wins. Nested settings merge key by key; lists of named items (like `launchers`) merge by `name` (same name replaces, new names are added); other lists are replaced.
- **Paths**: relative paths resolve against the config file that declared them. Set `beans.path` in a subdirectory config to give it its own beans directory. If no config sets it, `.beans` next to the nearest config is used.

### Custom Statuses

The builtin statuses (`in-progress`, `todo`, `draft`, `completed`, `scrapped`) are always available. You can add your own statuses in `.beans.yml`, grouped below the builtin status they are a kind of:

```yaml
statuses:
  in-progress:
    - name: review
      color: purple          # optional, defaults to the builtin's color
      description: Waiting for code review
  completed:
    - name: merged
```

- **Order**: custom statuses sort right after their builtin status.
- **Filtering**: filtering by a builtin status also matches its custom statuses (`beans list -s in-progress` shows `review` beans too); filtering by a custom status matches only that status. The same applies to `status` and `excludeStatus` in GraphQL.
- **Behaviour**: custom statuses inherit the builtin's archive behaviour, so `merged` beans are hidden from `beans list` by default, are removed by `beans archive`, and no longer block other beans. `beans list --include completed` shows `completed` and `merged` beans; `--include merged` shows only `merged` ones.
- **GraphQL**: a bean's `statusCategory` field holds the builtin status its status belongs to.

### User Config

Settings that should apply to all your projects go in `~/.config/beans/config.yml` (or `$XDG_CONFIG_HOME/beans/config.yml`). It uses the same format as `.beans.yml` and is the base layer below every project config, which can override it. `beans.path` can't be set there, and `cascade` has no effect.

### Custom Key Bindings

Every TUI key can be rebound in the `keybinding` section of the user config or a `.beans.yml`, much like lazygit's:

```yaml
keybinding:
  global:
    quit: ctrl+q
  list:
    changeStatus: [x, "g s"]  # a key or a list of keys; "g s" is a chord
    archive: <disabled>       # unbind (an empty list works too)
  picker:
    cancel: [esc, q]
```

- Bindings are grouped by context; the names mirror the TUI's Go key maps in lowerCamel (`ListKeyMap.ChangeStatus` → `list.changeStatus`). Unknown names are an error.
- An override replaces all default keys of that action. Keys use bubbletea's names (`ctrl+c`, `enter`, `esc`, `shift+tab`, `pgdown`, …); use `space` for the space bar.
- Keys bound to several actions of one context (`global` and `layout` count as part of `list`, `context` and `depGraph`) are reported as a warning in the footer.
- The tag editor and the create dialog have text inputs: printable keys bound there can't be typed anymore.
- Footers and the help overlay show your bindings. Key bindings are reloaded with the rest of the config (`R`, or automatically when a config file changes).

| Context | Actions (default keys) |
|---------|------------------------|
| `global` | `quit` (`q`), `forceQuit` (`ctrl+c`), `help` (`?`), `reloadConfig` (`R`), `openLog` (`O`) |
| `layout` | `nextPreset` (`+` `=`), `prevPreset` (`-`), `focusNext` (`tab`), `focusPrev` (`shift+tab`), `toggleContext` (`v`), `toggleDepGraph` (`D`), `toggleLog` (`L`) |
| `list` | `up` (`up` `k`), `down` (`down` `j`), `prevPage` (`left` `pgup` `u`), `nextPage` (`right` `pgdown` `f` `d`), `goToStart` (`home`), `goToEnd` (`end` `G`), `filter` (`/`), `open` (`enter`), `back` (`esc` `backspace`), `toggleSelect` (`space`), `collapse` (`h`), `expand` (`l`), `toggleCollapse` (`z`), `collapseAll` (`Z`), `filterByTag` (`g t`), `create` (`c`), `createChild` (`C`), `edit` (`e`), `openEditor` (`E`), `changeStatus` (`s`), `changeType` (`t`), `changePriority` (`P`), `raisePriority` (`<`), `lowerPriority` (`>`), `editTags` (`T`), `changeParent` (`p`), `manageBlocking` (`b`), `manageBlockedBy` (`B`), `launch` (`!`), `archive` (`Y`), `toggleArchived` (`A`) |
| `context` | `scrollUp` (`up` `k`), `scrollDown` (`down` `j`), `pageUp` (`pgup`), `pageDown` (`pgdown` `space` `f`), `halfPageUp` (`u` `ctrl+u`), `halfPageDown` (`d` `ctrl+d`), `goToStart` (`home` `g`), `goToEnd` (`end` `G`), `back` (`esc` `backspace`), `edit` (`e`), `openEditor` (`E`), `changeStatus` (`s`), `changeType` (`t`), `changePriority` (`P`), `changeSize` (`S`), `editTags` (`T`), `changeParent` (`p`), `manageBlocking` (`b`), `manageBlockedBy` (`B`), `launch` (`!`), `archive` (`Y`) |
| `depGraph` | `up` (`up` `k`), `down` (`down` `j`), `refocus` (`enter`), `toggleDone` (`A`), `back` (`esc` `backspace`) |
| `help` | `close` (`?` `esc`) |
| `picker` | `up` (`up` `k`), `down` (`down` `j`), `prevPage` (`left` `h` `pgup` `b` `u`), `nextPage` (`right` `l` `pgdown` `f` `d`), `goToStart` (`home` `g`), `goToEnd` (`end` `G`), `filter` (`/`), `confirm` (`enter`), `cancel` (`esc` `backspace`), `toggle` (`space`, blocking picker) |
| `tagEditor` | `up` (`up` `ctrl+p`), `down` (`down` `ctrl+n`), `toggle` (`space` `tab`), `apply` (`enter`), `cancel` (`esc`) |
| `createModal` | `cancel` (`esc`), `nextField` (`tab`), `prevField` (`shift+tab`), `submit` (`ctrl+s`), `submitAndEdit` (`ctrl+e`), `quickSubmit` (`enter`, not in the description), `cyclePrev` (`left` `h` `up` `k`), `cycleNext` (`right` `l` `down` `j` `space`), `quickSubmitAndEdit` (`e`); the last three only on type/status/priority |
| `archiveConfirm` | `confirm` (`y` `Y`), `cancel` (`n` `N` `esc` `backspace`) |
| `configureLaunchers` | `up` (`up` `k`), `down` (`down` `j`), `toggle` (`space` `enter`), `confirm` (`y`), `cancel` (`esc` `q`) |

## Agent Configuration

We'll need to teach your coding agent that it should use Beans to track tasks, and how to do so. The exact steps will depend on which agent you're using.

### Claude Code

An official Beans plugin for Claude is in the works, but for the time being, please manually add the following hooks to your project's `.claude/settings.json` file:

```json
{
  "hooks": {
    "SessionStart": [
      { "hooks": [{ "type": "command", "command": "beans prime" }] }
    ],
    "PreCompact": [
      { "hooks": [{ "type": "command", "command": "beans prime" }] }
    ]
  }
}
```

### Other Agents

You can use Beans with other coding agents by configuring them to run `beans prime` to get the prompt instructions for task management. We'll add specific integrations for popular agents over time.

## Usage Hints

As a human, you can get an overview of the CLI's functionalities by running:

```bash
beans help
```

You might specifically be interested in the interactive TUI:

```bash
beans tui
```

The TUI is made of panes, lazygit-style: the bean list, the details of the selected bean, its dependency graph and the log. The focused pane has a highlighted frame.

- `enter` on a bean shows it in the details pane and moves focus there; `tab`/`shift+tab` cycle through the panes and `esc` goes back.
- `+` and `-` step through layout presets, which depend on the focused pane: from the list, `+` hides the details, then the log; from the details, `+` swaps the list for the dependency graph, then shows the details alone.
- `v`, `D` and `L` toggle the details, the dependency graph and the log anywhere. In the dependency graph, moving the cursor shows that bean in the details; `enter` makes it the graph's root and moves focus to the details.
- In terminals narrower than 120 columns, the list and the details don't fit side by side, so only the focused one is shown.

**But the real power of Beans** comes from letting your coding agent manage your tasks for you.

Assuming you have integrated Beans into your coding agent correctly, it will already know how to create and manage beans for you. You can use the usual assortment of natural language inquiries. If you've just
added Beans to an existing project, you could try asking your agent to identify potential tasks and create beans for them:

> "Are there any tasks we should be tracking for this project? If so, please create beans for them."

If you already have some beans available, you can ask your agent to recommend what to work on next:

> "What should we work on next?"

You can also specifically ask it to start working on a particular bean:

> "It's time to tackle myproj-123."

## Launchers

You can configure external tools to launch from the TUI. Press `!` when viewing or selecting a bean to open the launcher picker.

**First-time setup:** If no launchers are configured, you'll be prompted to select from default options (opencode, claude, crush). Only installed tools will be pre-selected. This provides a quick start with sensible defaults.

Configure launchers in `.beans.yml`:

```yaml
launchers:
  # Simple single-line launcher
  - name: opencode
    exec: opencode -p "Work on task $BEANS_ID"
    description: "Open task in OpenCode"
  
  # Git worktree + tmux launcher
  - name: worktree-tmux
    description: "Create git worktree and open in new tmux pane"
    exec: |
      #!/bin/bash
      set -euo pipefail
      
      WORKTREE_DIR=".worktrees/$BEANS_ID"
      
      # Create worktree if it doesn't exist
      if [ ! -d "$WORKTREE_DIR" ]; then
        git worktree add "$WORKTREE_DIR" -b "task/$BEANS_ID"
      fi
      
      # Open in new tmux pane
      tmux split-window -h -c "$BEANS_ROOT/$WORKTREE_DIR"
      tmux send-keys "opencode -p 'Work on task $BEANS_ID'" Enter
  
  # Kitty terminal window launcher
  - name: worktree-kitty
    description: "Create git worktree and open in new kitty window"
    exec: |
      #!/bin/bash
      set -euo pipefail
      
      WORKTREE_DIR=".worktrees/$BEANS_ID"
      
      # Create worktree if it doesn't exist
      if [ ! -d "$WORKTREE_DIR" ]; then
        git worktree add "$WORKTREE_DIR" -b "task/$BEANS_ID"
      fi
      
      # Open in new kitty window
      kitty @ launch --type=window --cwd="$BEANS_ROOT/$WORKTREE_DIR" \
        opencode -p "Work on task $BEANS_ID"
```

### Multi-Bean Launchers

Some launchers can handle multiple beans in parallel, while others can only work with one bean at a time. Mark launchers that support parallel execution with `multiple: true`:

```yaml
launchers:
  # Single-bean only (default)
  - name: opencode
    exec: opencode -p "Work on task $BEANS_ID"
    description: "Open task in OpenCode"
    # multiple: false (default)
  
  # Can handle multiple beans in parallel
  - name: worktree-tmux
    multiple: true  # ← Supports multi-bean launching
    description: "Create git worktree and open in new tmux pane"
    exec: |
      #!/bin/bash
      set -euo pipefail
      
      WORKTREE_DIR=".worktrees/$BEANS_ID"
      
      if [ ! -d "$WORKTREE_DIR" ]; then
        git worktree add "$WORKTREE_DIR" -b "task/$BEANS_ID"
      fi
      
      tmux split-window -h -c "$BEANS_ROOT/$WORKTREE_DIR"
      tmux send-keys "opencode -p 'Work on task $BEANS_ID'" Enter
```

**When to use `multiple: true`:**
- Launchers that create separate workspaces (worktrees, tmux panes, kitty windows)
- Scripts that can safely run in parallel without conflicts
- Tools that operate on isolated resources per bean

**When to keep default `multiple: false`:**
- Interactive tools that can only open one file/project at a time (editors, IDEs)
- Tools that modify shared state
- Single-instance applications

**Behavior:**
- **Single bean selected** → All launchers shown
- **Multiple beans selected** → Only `multiple: true` launchers shown
- **No `multiple: true` launchers available** → Error message displayed

### Background Launchers

By default a launcher takes over the terminal until it exits, which interactive tools like `claude` or `opencode` need. Launchers that only start something elsewhere (a tmux window, a kitty window, a script) can run in the background instead, so the TUI stays usable:

```yaml
launchers:
  - name: worktree-tmux
    background: true  # ← Runs without the terminal
    multiple: true    # ← Runs for every selected bean in parallel
    exec: ...
```

Background launchers get no stdin; their stdout and stderr are captured. With `multiple: true` and several beans selected, one run per bean starts in parallel.

Every run, foreground or background, shows up in the **log panel** below the bean list: time, ⟳/✓/✗, launcher, bean ID, exit code, and the last lines of output (only stderr for foreground runs, since their stdout is the terminal). The panel appears when a run starts or fails; press `L` to toggle it and `O` to open the latest run's full output in `$EDITOR`. The log is kept for the TUI session only. Background runs still going when you quit keep running; their output files (`beans-launch-*.log` in your temp directory) are left behind for you to inspect.

### How It Works

- **Single-line exec**: Runs via `sh -c`, just like a shell command. Use for simple commands.
- **Multi-line exec**: Must start with a shebang (`#!/bin/bash`, etc.). Note that for now multi-line scripts also run via `sh -c`, so the shebang line is treated as a comment and the script must be valid `sh`.
- **Environment variables**: All launchers receive `$BEANS_ROOT`, `$BEANS_CONFIG_DIR`, `$BEANS_DIR`, `$BEANS_ID`, `$BEANS_TASK`
- **Working directory**: Set to the directory of the `.beans.yml` that declared the launcher (`$BEANS_CONFIG_DIR`). For a launcher defined in your project's own config, that's the project root (`$BEANS_ROOT`)
- **Unix/Linux/macOS only**: Shebang mechanism is Unix-specific

### Multi-Select Launching

In the TUI, you can launch for multiple beans simultaneously using launchers marked with `multiple: true`:

1. **Select multiple beans** using `x` (mark/unmark) or `X` (mark all visible)
2. **Press `!`** to open the launcher picker (only shows `multiple: true` launchers)
3. **Choose a launcher** - it will run in parallel for all selected beans

All runs start simultaneously (perfect for creating multiple tmux panes or worktrees), and each one shows up in the log panel with its own status and exit code.

**Use cases:**
- Create git worktrees for multiple tasks at once
- Open multiple beans in separate tmux panes/kitty windows
- Batch process related beans

### Examples

```yaml
# Simple tool invocation
- name: cursor
  exec: cursor "$BEANS_ROOT"

# Shell command with pipes
- name: show-bean
  exec: beans show "$BEANS_ID" | less

# Multi-step bash script
- name: test-and-open
  exec: |
    #!/bin/bash
    set -e
    cd "$BEANS_ROOT"
    npm test -- "$BEANS_ID" || true
    code "$BEANS_ROOT"
```

### Environment Variables

Launchers receive these environment variables:

- `BEANS_ROOT`: Project root directory (where the nearest `.beans.yml` lives)
- `BEANS_CONFIG_DIR`: Directory of the `.beans.yml` that declared the launcher (the working directory; differs from `BEANS_ROOT` for [inherited launchers](#monorepos-cascading-configuration))
- `BEANS_DIR`: Beans directory (e.g., `/path/to/project/.beans`)
- `BEANS_ID`: Bean ID (e.g., `beans-abc123`)
- `BEANS_TASK`: Bean title (e.g., `Fix login redirect`)

### CLI Usage

Run a launcher for a bean from the command line with `beans launch`:

```bash
# Launch by full or partial (unique) bean ID
beans launch beans-abc123 --launcher opencode
beans launch beans-abc -l opencode

# Read the bean ID from the first line of stdin
echo beans-abc123 | beans launch --launcher worktree-tmux

# List the configured launchers (add --json for scripting)
beans launch --list
beans launch --list --json
```

The launcher runs attached to your terminal, also when the bean ID came from stdin, so interactive tools keep working. `beans launch` waits for it and exits with its exit code. `background` and `multiple` only affect the TUI.

## Contributing

This project currently does not accept contributions -- it's just way too early for that!
But if you do have suggestions or feedback, please feel free to open an issue.

## License

This project is licensed under the Apache-2.0 License. See the [LICENSE](LICENSE) file for details.

## Getting in Touch

If you have any questions, suggestions, or just want to say hi, feel free to reach out to me [on Bluesky](https://bsky.app/profile/hmans.dev), or [open an issue](https://github.com/hmans/beans/issues) in this repository.

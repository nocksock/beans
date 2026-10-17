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

From this point onward, you can interact with your Beans through the `beans` CLI. To get a list of available commands:

```bash
beans help
```

But more importantly, you'll want to get your coding agent set up to use it. Let's dive in!

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

**First-time setup:** If no launchers are configured, you'll be prompted to select from default options (opencode, claude, crush). Only installed tools will be pre-selected. This provides a quick start with sensible defaults.

Configure launchers in `.beans.yml`:

```yaml
launchers:
  # Simple single-line launcher
  - name: opencode
<<<<<<< HEAD
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
=======
    command: opencode run "Work on task $BEANS_ID"
    description: "Open task in OpenCode"
  - name: claude
    command: claude "Work on task $BEANS_ID"
    description: "Open task in Claude Code"
  - name: custom-tool
    command: .beans/scripts/my-tool.sh
    description: "Run custom analysis"
```

**Command execution:**
- All launcher commands execute via shell (`sh -c`), allowing environment variable expansion
- Use `$BEANS_ID`, `$BEANS_ROOT`, `$BEANS_TASK` in your commands
- Commands can be executable names in PATH, relative paths, or absolute paths

**Command templates:** The default launchers serve as templates for custom commands. You can create sophisticated launcher commands using shell features:

```yaml
# Extract and use the task title
command: opencode run "Work on $(head -n1 $BEANS_TASK | sed 's/^# //')"

# Pass multiple environment variables
command: my-tool --id=$BEANS_ID --root=$BEANS_ROOT

# Chain commands
command: cd $BEANS_ROOT && my-script.sh $BEANS_ID
```
>>>>>>> main

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

### How It Works

- **Single-line exec**: Runs via `sh -c`, just like a shell command. Use for simple commands.
- **Multi-line exec**: Passed to the interpreter via stdin. **Must start with a shebang** (`#!/bin/bash`, `#!/usr/bin/env python3`, etc.). The shebang determines which interpreter executes your script.
- **Environment variables**: All launchers receive `$BEANS_ROOT`, `$BEANS_CONFIG_DIR`, `$BEANS_DIR`, `$BEANS_ID`, `$BEANS_TASK`
- **Working directory**: Set to the directory of the `.beans.yml` that declared the launcher (`$BEANS_CONFIG_DIR`). For a launcher defined in your project's own config, that's the project root (`$BEANS_ROOT`)
- **Unix/Linux/macOS only**: Shebang mechanism is Unix-specific

### Multi-Select Launching

In the TUI, you can launch for multiple beans simultaneously using launchers marked with `multiple: true`:

1. **Select multiple beans** using `x` (mark/unmark) or `X` (mark all visible)
2. **Press `!`** to open the launcher picker (only shows `multiple: true` launchers)
3. **Choose a launcher** - it will run in parallel for all selected beans

**Features:**
- **Parallel execution**: All launchers start simultaneously (perfect for creating multiple tmux panes or worktrees)
- **Real-time progress**: See status for each bean as it runs
- **Stop on failure**: If any bean fails, all others are immediately stopped
- **Confirmation prompt**: For 5+ beans, you'll be asked to confirm (can be disabled in config)
- **Selection management**: Selection is cleared on success, kept on failure (so you can retry)

**Use cases:**
- Create git worktrees for multiple tasks at once
- Open multiple beans in separate tmux panes/kitty windows
- Batch process related beans

### TUI Configuration

You can customize TUI behavior in `.beans.yml`:

```yaml
tui:
  # Disable confirmation prompt when launching for 5+ beans
  disable_launcher_warning: false
```

### Examples

```yaml
# Simple tool invocation
- name: cursor
  exec: cursor "$BEANS_TASK"

# Shell command with pipes
- name: show-bean
  exec: cat "$BEANS_TASK" | less

# Multi-step bash script
- name: test-and-open
  exec: |
    #!/bin/bash
    set -e
    cd "$BEANS_ROOT"
    npm test -- "$BEANS_ID" || true
    code "$BEANS_TASK"

# Ruby script
- name: ruby-tool
  exec: |
    #!/usr/bin/env ruby
    bean_id = ENV['BEANS_ID']
    puts "Processing #{bean_id}"
```

### Environment Variables

Launchers receive these environment variables:

- `BEANS_ROOT`: Project root directory (where the nearest `.beans.yml` lives)
- `BEANS_CONFIG_DIR`: Directory of the `.beans.yml` that declared the launcher (the working directory; differs from `BEANS_ROOT` for [inherited launchers](#monorepos-cascading-configuration))
- `BEANS_DIR`: Beans directory (e.g., `/path/to/project/.beans`)
- `BEANS_ID`: Bean ID (e.g., `beans-abc123`)
- `BEANS_TASK`: Full path to bean file (e.g., `/path/to/project/.beans/beans-abc123.md`)

### CLI Usage

<<<<<<< HEAD
Launch beans from the command line:

```bash
# List all configured launchers with availability status
beans launch -l

# Example output:
# NAME            AVAILABLE  DESCRIPTION
# opencode        ✓          Open task in OpenCode
# worktree-tmux   ✓          Create git worktree and open in new tmux pane
# test-echo       ✓          Simple test launcher

# Launch a specific launcher for a bean
beans launch opencode beans-abc123    # Full ID
beans launch opencode beans-abc       # Partial ID (if unique)

# Launch worktree launchers
beans launch worktree-tmux beans-def456
beans launch worktree-kitty beans-ghi789

# JSON output for scripting
beans launch -l --json
```

**CLI launcher features:**
- Supports partial bean ID matching (like other commands)
- Executes launchers interactively (you see output in your terminal)
- Shows availability status with `-l` (✓ if command/interpreter found, ✗ if not)
- Returns non-zero exit code on failure

=======
>>>>>>> main
## Contributing

This project currently does not accept contributions -- it's just way too early for that!
But if you do have suggestions or feedback, please feel free to open an issue.

## License

This project is licensed under the Apache-2.0 License. See the [LICENSE](LICENSE) file for details.

## Getting in Touch

If you have any questions, suggestions, or just want to say hi, feel free to reach out to me [on Bluesky](https://bsky.app/profile/hmans.dev), or [open an issue](https://github.com/hmans/beans/issues) in this repository.

# Tmux Session Launcher for Beans

`tmux-session-run` opens (or switches to) a tmux session for a bean. Beans has no
built-in session management; sessions are plain launchers.

## Environment Variables

Launchers receive these environment variables:

| Variable | Description |
|----------|-------------|
| `BEANS_ID` | The bean ID (e.g., `beans-abc123`) |
| `BEANS_TASK` | The bean title |
| `BEANS_DIR` | The `.beans` directory path |
| `BEANS_ROOT` | The project root directory |

## Example Configuration

Add to your `.beans.yml`:

```yaml
launchers:
  - name: tmux
    exec: ./scripts/tmux-session-run
    description: "Open task in tmux"
```

Then press `!` on a bean and pick the `tmux` launcher.

# Composed agent launchers

`bean-launch` starts a coding agent on a bean from three independent choices:

    scripts/bean-launch [--worktree] [--tmux|--kitty] <claude|opencode|crush>

| Piece | Script | Choice |
|-------|--------|--------|
| Where | `bean-worktree` | the dev worktree (default) or the bean's own worktree (`--worktree`) |
| Prompt | `bean-prompt` | bean prompt plus tmux, cleanup and draft notes as they apply |
| Host | `bean-launch` | this terminal (default), a new tmux window or a new kitty window |
| Agent | `bean-agent` | how each agent CLI takes the prompt |

A new agent is one line in `bean-agent`. A new combination is one launcher in `.beans.yml`:

```yaml
  - name: worktree-tmux+opencode
    multiple: true
    background: true
    exec: '"${BEANS_ROOT:-.}/scripts/bean-launch" --worktree --tmux opencode'
```

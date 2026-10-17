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

# Tmux Session Scripts for Beans

Example scripts demonstrating the multiplexer-agnostic session pattern for beans launchers.

## Files

| Script | Purpose |
|--------|---------|
| `tmux-session-check` | Check if a session exists for a bean (exit 0 = exists) |
| `tmux-session-attach` | Attach to or create a session for a bean |
| `tmux-session-run` | Run a command in a new tmux session |

## Environment Variables

All scripts receive these environment variables:

| Variable | Description |
|----------|-------------|
| `BEANS_ID` | The bean ID (e.g., `beans-abc123`) |
| `BEANS_TASK` | The bean title |
| `BEANS_DIR` | The `.beans` directory path |
| `BEANS_ROOT` | The project root directory |

## Example Configuration

Add to your `.beans.yml`:

```yaml
session:
  check: ./scripts/tmux-session-check
  attach: ./scripts/tmux-session-attach
  launch: ./scripts/tmux-session-run
```

## Usage

1. **Install**: Copy scripts to your project's `scripts/` directory
2. **Make executable**: `chmod +x scripts/tmux-session-*`
3. **Configure**: Add the session config to `.beans.yml`
4. **Use**: Press `S` on a bean to launch/start a session

## How It Works

1. **Polling**: Beans continuously polls `check` command for in-progress beans (round-robin, 200ms interval)
2. **Status**: When `check` exits 0, the bean shows "running" status
3. **Launch**: Press `S` to run the `launch` command (starts new session)
4. **Attach**: Press `!` then select a session launcher, or use the session dialog (`S` in list view)

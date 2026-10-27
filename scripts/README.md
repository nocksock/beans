# Hub launches

The agent launches for this hub live in [`../.beans.yml`](../.beans.yml) and are
composed from the builtin launches that ship with beans (`beans launch --list`
shows them as `builtin.*`):

| Launch | Place | Host |
|--------|-------|------|
| `worktree-tmux+<agent>` | the bean's own worktree | new tmux window |
| `worktree-kitty+opencode` | the bean's own worktree | new kitty window |
| `tmux+<agent>` | `dev` | new tmux window |
| `<agent>` | `dev` | this terminal |
| `tmux` | `dev` | tmux session for the bean |

Two builtins are overridden for the hub:

- `prompt` calls `parent.prompt` and appends the dev-merge cleanup steps.
- `worktree` calls `parent.worktree` with `--base dev` and
  `--dir "$BEANS_ROOT/$BEANS_ID"`, then hides `.beans/` and `.beans.yml` with
  sparse-checkout on branches that predate the hub.

Run any of them from the command line, e.g.

    beans launch worktree-tmux+opencode beans-abc1

Adding an agent means adding one agent launch (and its combinations) in
`../.beans.yml`; the agent CLIs take the prompt from `builtin.agent`.

## Remaining scripts

| Script | Purpose |
|--------|---------|
| `check-worktrees.sh` | Report worktree, merge and bean status; `merge --completed` merges completed, unmerged worktrees into the current branch |
| `shorten` | Shorten text from stdin to a 20-character label, e.g. for a tmux window name |

---
# beans-dkr9
title: Migrate hub launchers to builtin launches
status: todo
type: task
priority: normal
created_at: 2026-10-19T12:20:48Z
updated_at: 2026-10-19T20:30:38Z
blocked_by:
    - beans-ppxk
---

Once beans-ppxk lands: rewrite the hub .beans.yml launchers on top of the builtins. Override `prompt` (call parent.prompt, append dev-merge cleanup notes) and `worktree` (--base dev --dir "$BEANS_ROOT/$BEANS_ID", sparse-checkout for old branches). Remove scripts/bean-agent, bean-launch, tmux-session-run.

Notes from beans-ppxk (as built):
- Flags for a script go after `--`: `"$BEANS_BIN" launch tmux-window -- --cwd "$dir" -- "$BEANS_BIN" launch agent -- claude --tmux --worktree "$dir"`.
- Configured launchers run via `sh -c`, so use `#!/bin/sh`-compatible scripts (see the README example).
- The hub `worktree` override must keep stdout to the path only (git output to stderr), like the builtin.
- `agent` calls the unqualified `prompt`, so a hub `prompt` override applies to every agent launch.
- Old `beans launch <id> --launcher X` calls (scripts, keybindings, muscle memory) are now `beans launch X <id>`.
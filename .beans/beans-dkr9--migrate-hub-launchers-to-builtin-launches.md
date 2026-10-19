---
# beans-dkr9
title: Migrate hub launchers to builtin launches
status: todo
type: task
created_at: 2026-10-19T12:20:48Z
updated_at: 2026-10-19T12:20:48Z
blocked_by:
    - beans-ppxk
---

Once beans-ppxk lands: rewrite the hub .beans.yml launchers on top of the builtins. Override `prompt` (call parent.prompt, append dev-merge cleanup notes) and `worktree` (--base dev --dir "$BEANS_ROOT/$BEANS_ID", sparse-checkout for old branches). Remove scripts/bean-agent, bean-launch, tmux-session-run.
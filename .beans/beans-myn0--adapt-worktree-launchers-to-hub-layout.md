---
# beans-myn0
title: Adapt worktree launchers to hub layout
status: completed
type: task
priority: normal
created_at: 2026-10-17T12:04:15Z
updated_at: 2026-10-17T12:04:52Z
---

worktree-tmux+claude and worktree-kitty launchers still create .worktrees/<id> with a sparse-checkout excluding .beans. Switch them to hub-root worktrees via a shared scripts/bean-worktree helper.
## Work Log
2026-10-17 14:00: Added scripts/bean-worktree, switched worktree launchers to it, tested new + legacy branches
2026-10-17 14:10: Pointed non-worktree launchers and tmux session scripts at dev/

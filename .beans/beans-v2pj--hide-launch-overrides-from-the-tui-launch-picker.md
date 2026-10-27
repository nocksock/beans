---
# beans-v2pj
title: Hide launch overrides from the TUI launch picker
status: todo
type: task
tags:
    - idea
created_at: 2026-10-27T15:53:24Z
updated_at: 2026-10-27T15:53:24Z
---

The TUI launcher picker lists every configured launcher, including ones that only override a builtin. The beans hub now overrides `prompt` and `worktree` (see beans-dkr9), so pressing `!` also offers those two internal pieces, which just print a prompt or create a worktree.

Ideas:
- Hide configured launchers whose name matches a builtin from the picker (they stay resolvable by name and in `beans launch --list`).
- Or add a `hidden: true` field to the launcher config.
- Multi-select should follow the same rule.
---
# beans-bw3m
title: Builtin prompt worktree note contradicts hub cleanup
status: completed
type: bug
priority: normal
created_at: 2026-10-28T09:53:05Z
updated_at: 2026-10-28T17:24:43Z
---

Review of beans-dkr9: with --worktree, builtin prompt adds '## Worktree' (leave merging/removal to user), while the hub prompt override appends '## Cleanup when done' (merge + remove). Contradictory.

Fix: add an opt-out flag to internal/launcher/builtin/prompt.sh for the worktree note; hub override passes it to parent.prompt.

Also noted (not fixed here): builtin draft note no longer says to commit the bean in the hub; draft note sits between Worktree and Cleanup sections.


## Work Log

2026-10-28 10:00: Added --no-worktree-note to builtin prompt.sh (+test, README); commit 301b21e on task/beans-bw3m.
2026-10-28 10:02: Pending hub step (needs merged + installed binary): in .beans.yml prompt override, pass --no-worktree-note to parent.prompt.

2026-10-28 18:26: Merged into dev, installed; hub prompt override passes --no-worktree-note, rendered prompt has Cleanup but no ## Worktree.
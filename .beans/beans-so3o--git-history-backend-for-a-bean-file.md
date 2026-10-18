---
# beans-so3o
title: Git history backend for a bean file
status: completed
type: task
priority: normal
created_at: 2026-10-18T09:49:29Z
updated_at: 2026-10-18T09:54:28Z
parent: beans-7jm4
---

Follow-up of beans-0ggp (task history pane). Independent of the pane framework, can be built now.

Provide a data layer that returns the git commit history of a bean's file (follow renames: slug part of the filename changes) and the diff of the bean file for a given commit. Expose it via GraphQL (e.g. Bean.history { hash author date subject } plus a diff field/query) so TUI and CLI can use it; run mise codegen. Shell out to git (no go-git dependency) and handle: not a git repo, uncommitted/untracked bean, beans dir in a different repo/worktree (hub layout). Table-driven tests with a temp git repo.

## Work Log

2026-10-18 11:51: Added internal/githistory (git log/show via ID pathspecs) with temp-repo tests
2026-10-18 11:53: Exposed Bean.history(limit), Bean.diff(commit), BeanCommit.diff via GraphQL; codegen + tests
2026-10-18 11:53: Verified against hub .beans via beans query

---
# beans-so3o
title: Git history backend for a bean file
status: todo
type: task
created_at: 2026-10-18T09:49:29Z
updated_at: 2026-10-18T09:49:29Z
parent: beans-7jm4
---

Follow-up of beans-0ggp (task history pane). Independent of the pane framework, can be built now.

Provide a data layer that returns the git commit history of a bean's file (follow renames: slug part of the filename changes) and the diff of the bean file for a given commit. Expose it via GraphQL (e.g. Bean.history { hash author date subject } plus a diff field/query) so TUI and CLI can use it; run mise codegen. Shell out to git (no go-git dependency) and handle: not a git repo, uncommitted/untracked bean, beans dir in a different repo/worktree (hub layout). Table-driven tests with a temp git repo.
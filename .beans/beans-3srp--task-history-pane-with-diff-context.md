---
# beans-3srp
title: Task history pane with diff context
status: todo
type: task
priority: normal
created_at: 2026-10-18T09:49:37Z
updated_at: 2026-10-18T11:15:17Z
parent: beans-7jm4
blocked_by:
    - beans-0ggp
    - beans-so3o
    - beans-8pxh
---

Follow-up of beans-0ggp. Add a navigable 'history' pane listing the git log of the current bean (backend: beans-so3o). When focused, the context pane shows the diff of the bean file for the selected commit. Add toggle key and preset cycle in presets.go.
## Backend (from beans-so3o)

Available via GraphQL: `bean(id) { history(limit: Int) { hash author date subject diff } }` and `bean(id) { diff(commit: "<hash>") }` (returns empty string if the commit didn't touch the bean). History is empty (not an error) outside git, for untracked beans and without commits. Uncommitted working-tree changes are not included; add a pseudo entry here if the pane wants them.

## Notes from beans-0ggp

- Add a pane by writing an adapter implementing `pane` (`internal/tui/panes.go`), registering it in `App.newPanes()`, and placing it in `layoutTree`/`presetCycles` (`presets.go`).
- The context pane currently always shows a bean (`contextModel`); a different context (log detail / diff) means letting the context pane render per source, e.g. via the focused `selectionSource`.
- The git history backend (beans-so3o) is merged: `internal/githistory`.

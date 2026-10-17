---
# beans-7wi6
title: Make bean titles optional
status: completed
type: feature
priority: normal
created_at: 2026-10-17T12:22:35Z
updated_at: 2026-10-17T12:29:04Z
---

Allow creating beans without a title. Where a title is displayed, fall back to a muted snippet of the body (first line / ~120 chars).

Plan: /home/nr/.claude/plans/how-complex-would-it-effervescent-torvalds.md

## Checklist
- [x] Bean.DisplayTitle() helper + omitempty title
- [x] Allow empty title in cmd create, GraphQL CreateBeanInput, TUI create modal
- [x] Muted body fallback in RenderBeanRow, detail header, pickers, filter values
- [x] Sorting, launcher env, prompt docs
- [x] Tests
## Work Log
2026-10-17: Created worktree optional-titles; added Bean.DisplayTitle/HasTitle/BodySnippet, title omitempty
2026-10-17: Empty title allowed in CLI create, GraphQL CreateBeanInput (title optional) and TUI create modal; title or body required
2026-10-17: DisplayTitle wired into rows (muted), detail/preview header (muted italic), pickers, sorts, launcher env, templates
2026-10-17: Untitled beans get a slug from the body (fallback "untitled"): ID-only filenames like proj-abc1.md were misparsed by the legacy id-slug fallback
2026-10-17: go test ./... green; verified CLI + TUI list rendering in scratch project

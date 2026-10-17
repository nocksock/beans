---
# beans-6aax
title: '`C` to create a child of the currently selected task(s)'
status: archived
type: feature
priority: normal
created_at: 2025-12-15T12:59:39Z
updated_at: 2026-10-17T14:16:52Z
parent: beans-7jm4
---

auto sets the selected (or current cursor position) as parent.
when multiple are selected, auto sets those. 

## Code review 2026-10-17 (beans-9pxj)

Verified done: CreateChild in list.go:651. Multi-select picks a random parent → tracked in beans-et11.

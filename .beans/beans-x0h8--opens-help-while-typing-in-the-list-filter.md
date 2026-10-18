---
# beans-x0h8
title: '''?'' opens help while typing in the list filter'
status: todo
type: bug
created_at: 2026-10-17T13:15:26Z
updated_at: 2026-10-17T13:15:26Z
parent: beans-7jm4
---

Typing '?' in the list's text filter (after '/') opens the help overlay instead of adding '?' to the filter.

## Analysis
- App.Update handles the global help key for viewList/viewDetail without checking whether a text filter is active (unlike quit/reload, which check canQuit()).
- Pre-existing: the old hardcoded case "?" in tui.go had the same check. Still present after beans-dx97 (now keymap.Global.Help).
- Likely the same in the detail view while filtering links.

## Repro
Verified with a test: newTestApp, press '/', press '?', and the state is viewHelpOverlay with filter value "".

## Fix idea
Guard the help case with a.canQuit() (or !a.typingText()), like ReloadConfig, and add a TestQuitKey-style table test.

## Update (beans-kumc)

Still reproducible after beans-kumc replaced the bubbles list filter with its own query input (`listModel.queryInput`): `/` then `?` opens help and the query stays empty. `canQuit()`/`typingText()` now use `a.list.editingQuery()` for viewList, so guarding the help case with `!a.typingText()` should fix it.

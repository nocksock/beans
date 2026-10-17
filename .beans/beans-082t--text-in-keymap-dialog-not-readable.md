---
# beans-082t
title: text in keymap dialog not readable
status: archived
type: task
priority: normal
created_at: 2026-10-17T13:20:06Z
updated_at: 2026-10-17T14:16:52Z
---

in light mode its white

## Cause

`internal/tui/help.go:74` renders shortcut descriptions with a hardcoded `Foreground(lipgloss.Color("#fff"))` and no background, so on light terminals the text is white-on-white. Other `#fff` usages in the TUI sit on coloured badge backgrounds and are fine.

## Plan

- [x] Remove the hardcoded foreground from `descStyle` in `help.go` so descriptions use the terminal default colour
- [x] Add a test asserting the help overlay's description style doesn't force a foreground colour
- [x] Run `go test ./...`

## Work Log

2026-10-17 15:25: Found cause (hardcoded #fff in help overlay); plan agreed: use terminal default colour
2026-10-17 15:32: Removed #fff from help overlay descriptions, added test; go test ./... green

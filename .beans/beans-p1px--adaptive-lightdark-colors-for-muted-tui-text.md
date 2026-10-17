---
# beans-p1px
title: Adaptive light/dark colors for muted TUI text
status: todo
type: task
created_at: 2026-10-17T13:06:53Z
updated_at: 2026-10-17T13:06:53Z
---

Follow-up from beans-wrpz. ui.ColorMuted (#9CA3AF) and similar fixed grays have low contrast on light backgrounds (~2.5:1). Consider lipgloss.AdaptiveColor for muted/subtle colors; requires ResolveColor/NamedColors to return lipgloss.TerminalColor.
## Code review 2026-10-17 (beans-9pxj)

Also: hardcoded colors bypassing palette: #ff0000 (launcherpicker.go:242,251, configurelaunchersdialog.go:245,254), #555 (modal.go:93,139), #fff (tui/styles.go:18, detail.go:238).

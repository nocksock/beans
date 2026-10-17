---
# beans-p1px
title: Adaptive light/dark colors for muted TUI text
status: todo
type: task
created_at: 2026-10-17T13:06:53Z
updated_at: 2026-10-17T13:06:53Z
---

Follow-up from beans-wrpz. ui.ColorMuted (#9CA3AF) and similar fixed grays have low contrast on light backgrounds (~2.5:1). Consider lipgloss.AdaptiveColor for muted/subtle colors; requires ResolveColor/NamedColors to return lipgloss.TerminalColor.
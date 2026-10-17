---
# beans-re37
title: Remove dead launcher manager and other dead code
status: completed
type: task
priority: normal
size: sm
tags:
    - refactoring
    - code-review
created_at: 2026-10-17T14:02:23Z
updated_at: 2026-10-17T14:33:40Z
parent: beans-gl8t
---

Review beans-9pxj (`docs/reviews/2026-10-17-code-quality-review.md` in hub), R1.

- [x] Delete `internal/launcher/manager.go` (LaunchManager, GetSummary) + tests — no production callers
- [ ] Delete `internal/launcher/launcher.go` (CreateExecCommand/ExecutionResult) after shebang fix lands (beans-m1x0) → moved to beans-m1x0
- [x] ui/styles.go unused: RenderStatus, RenderStatusText, StatusOpen/Done/InProgress(+Text), RenderTypeWithColor, Header, Path, Secondary
- [x] TUI: openConfigureLaunchersMsg, configWriteErrorMsg, openHelpMsg (never sent), tagPickerModel.tags, always-true `!m.ready` branch (detail.go:292), unused beansRoot param in launcher discover fns, orphaned getEditor comment (tui.go:1134), pointer-to-map `&m.selectedBeans` in delegates
- [x] Core/output: output.SuccessWithWarnings, ErrNoBeansDir/ErrAmbiguousID codes, Core.FullPath, Bean.HasParent, config.LegacyConfigFile, TUIConfig.DisableLauncherWarning
- [x] cmd: `truncate` (list.go:313, test-only), empty cmd/update_test.go
- [x] ui/tree.go: dedupe tree prefix builder (263-278 vs 341-357), FlattenTree = FlattenTreeWithCollapsed(nil)
- [x] gofmt all files (`gofmt -l` reports ~14)
## Sequencing (review beans-9pxj)

Step 3 of 5: after beans-m1x0 (launcher.go removal needs the shebang fix).

## Work Log
2026-10-17 16:20: Deleted launcher/manager.go + GetSummary tests; launcher.go kept (beans-m1x0 still todo)
2026-10-17 16:21: Removed dead config/output/core/cmd/ui symbols, DisableLauncherWarning (+init flag, README), deduped tree prefix
2026-10-17 16:22: TUI dead code removed, delegates use plain maps, gofmt clean, all tests pass. launcher.go item left open pending beans-m1x0

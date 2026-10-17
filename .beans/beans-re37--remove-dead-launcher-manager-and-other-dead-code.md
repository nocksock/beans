---
# beans-re37
title: Remove dead launcher manager and other dead code
status: todo
type: task
tags:
    - refactoring
    - code-review
created_at: 2026-10-17T14:02:23Z
updated_at: 2026-10-17T14:02:23Z
parent: beans-gl8t
---

Review beans-9pxj (`docs/reviews/2026-10-17-code-quality-review.md` in hub), R1.

- [ ] Delete `internal/launcher/manager.go` (LaunchManager, GetSummary) + tests — no production callers
- [ ] Delete `internal/launcher/launcher.go` (CreateExecCommand/ExecutionResult) after shebang fix lands (beans-m1x0)
- [ ] ui/styles.go unused: RenderStatus, RenderStatusText, StatusOpen/Done/InProgress(+Text), RenderTypeWithColor, Header, Path, Secondary
- [ ] TUI: openConfigureLaunchersMsg, configWriteErrorMsg, openHelpMsg (never sent), tagPickerModel.tags, always-true `!m.ready` branch (detail.go:292), unused beansRoot param in launcher discover fns, orphaned getEditor comment (tui.go:1134), pointer-to-map `&m.selectedBeans` in delegates
- [ ] Core/output: output.SuccessWithWarnings, ErrNoBeansDir/ErrAmbiguousID codes, Core.FullPath, Bean.HasParent, config.LegacyConfigFile, TUIConfig.DisableLauncherWarning
- [ ] cmd: `truncate` (list.go:313, test-only), empty cmd/update_test.go
- [ ] ui/tree.go: dedupe tree prefix builder (263-278 vs 341-357), FlattenTree = FlattenTreeWithCollapsed(nil)
- [ ] gofmt all files (`gofmt -l` reports ~14)
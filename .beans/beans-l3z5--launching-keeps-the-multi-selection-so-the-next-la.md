---
# beans-l3z5
title: Launching keeps the multi-selection, so the next launch reruns all marked beans
status: archived
type: bug
priority: normal
created_at: 2026-10-19T19:41:26Z
updated_at: 2026-10-19T20:46:15Z
parent: beans-gl8t
---

After marking several beans and running a launcher, the selection stayed. Pressing launch again on another bean ran the launcher for all previously marked beans, not the cursor bean.

Cause: the launcherSelectedMsg handler in internal/tui/tui.go never cleared a.list.selectedBeans, though every other batch action does. The launch key handler in list.go prefers selectedBeans over the cursor.

Fix: clear the selection when a launcher runs. Cancelling the picker keeps it. Regression test: TestLaunchClearsSelection in internal/tui/logpanel_test.go.

## Work Log
2026-10-19: Confirmed cause, added clear + regression test; go test ./... passes (uncommitted in dev)
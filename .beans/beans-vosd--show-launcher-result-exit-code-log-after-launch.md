---
# beans-vosd
title: Show launcher result (exit code + log) after launch
status: completed
type: feature
priority: normal
created_at: 2026-10-14T16:10:16Z
updated_at: 2026-10-14T16:12:25Z
parent: beans-gl8t
---

After a launcher/session-launch command exits, show a result modal with success/failure, exit code, and the last lines of stderr. 'e' opens the full log in the editor.

## Work Log
2026-10-14 10:00: Picked up uncommitted WIP (launcherresult.go, RunWithOutput). Found RunWithOutput runs interactive launchers without a TTY -> switching back to ExecProcess with stderr tee.2026-10-14 10:20: Replaced RunWithOutput with CreateLoggedExecCommand (stderr tee) + ExitCode; launches go through App.runLauncher via tea.ExecProcess
2026-10-14 10:25: Result modal: handle resize, truncate long lines, fix border newlines; removed dead code after clearFilterMsg
2026-10-14 10:30: Added tests (sessionpoller, launcherresult); verified in tmux: failure modal, log cleanup, launcher keeps TTY

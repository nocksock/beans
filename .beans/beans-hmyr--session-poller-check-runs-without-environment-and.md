---
# beans-hmyr
title: Session poller check runs without environment and wrong BEANS_ROOT/BEANS_DIR
status: todo
type: bug
priority: normal
created_at: 2026-10-17T11:05:00Z
updated_at: 2026-10-17T12:14:17Z
parent: beans-gl8t
---

Found while implementing beans-jzty.

`internal/sessionpoller/poller.go` checkSession builds `cmd.Env` from scratch (no `os.Environ()`), so PATH/HOME etc. are missing for the check script. It also sets both `BEANS_ROOT` and `BEANS_DIR` to `p.beansRoot`, which is the beans directory (`core.Root()`), not the project root.

Fix: build the check env via `launcher.Launch{...}.Env()` with the project root passed into `sessionpoller.New`.
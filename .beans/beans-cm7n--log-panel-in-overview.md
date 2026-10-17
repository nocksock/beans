---
# beans-cm7n
title: log panel in overview
status: in-progress
type: task
priority: normal
created_at: 2026-10-17T13:08:21Z
updated_at: 2026-10-17T13:16:43Z
---

similar to what lazygit has: a log of commands that are run in the background, eg from launcher.

this is better than having the success modal after running a launcher.

## Plan

Replace the launcher result modal with a lazygit-style log panel at the bottom of the overview, and allow launchers to run in the background.

### Decisions
- Per-launcher opt-in: `background: true` in `.beans.yml` (default stays foreground, so interactive launchers like claude/opencode keep working).
- Panel is a toggleable bottom strip under list+preview; hidden by default, auto-shows when a run starts or fails.
- Entry = status header + output tail. In-memory log for the TUI session, capped at 50 runs.
- Foreground runs also produce a log entry; the result modal is removed.
- Background launchers with `multiple: true` start one run per selected bean in parallel (fixes: today only `beanIDs[0]` is launched).

### Design
- **Config**: `config.Launcher.Background bool \`yaml:"background,omitempty"\``.
- **Exec** (`internal/launcher/exec.go`): `CreateBackgroundCommand(l) (*exec.Cmd, *os.File, error)`: no stdin, stdout+stderr into temp log file, `Setpgid` so TUI Ctrl-C doesn't kill it. Foreground keeps `CreateLoggedExecCommand`. Runs still in flight on TUI quit keep running; only finished runs' temp files are deleted.
- **Model** (`internal/tui/logpanel.go`): `logEntry{id, launcher, beanID, started, finished, state running|ok|failed, exitCode, logPath}`; `launchStartedMsg` / `launchFinishedMsg{id, exitCode}`; background run = `tea.Cmd` blocking on `cmd.Run()`; tail read on finish (reuse `readLastLines`).
- **UI**: ~6-row strip with bordered "Log" title; per run `15:04 ⟳/✓/✗ <launcher> <bean-id> [exit N]` + up to 3 tail lines (muted / red on failure); newest at bottom. Non-focusable. `L` toggles, `O` opens the latest run's full output in `$EDITOR`. Footer + help updated; list height shrinks while visible.
- **Remove**: `launcherresult.go` modal, `viewLauncherResult`, related msgs (move `readLastLines`/`centerContent` if still used).

### Checklist
- [x] Add `background` field to launcher config (+ docs/README)
- [x] `CreateBackgroundCommand` in `internal/launcher` with tests
- [x] Log model + messages with tests (state transitions, 50 cap)
- [x] Panel rendering + layout integration (list height, auto-show, `L`, `O`) with tests
- [x] Route foreground + background launches into the log; multi-bean parallel background runs
- [x] Remove launcher result modal
- [x] Manual test in a scratch project

## Work Log
2026-10-17 15:20: Planned design with user; plan written
2026-10-17 15:24: Implemented exec helper, config flag, log panel, App wiring; removed result modal
2026-10-17 15:25: Manually tested in scratch project via tmux (multi-bean bg, failure, L, O); filed beans-z7jr for README conflict markers
2026-10-17 15:27: Code review fixes: Setsid, bounded tail read, eviction keeps running entries

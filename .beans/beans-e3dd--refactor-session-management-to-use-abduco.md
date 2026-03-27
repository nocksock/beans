---
# beans-e3dd
title: Refactor session management to use abduco
status: completed
type: task
priority: normal
created_at: 2026-02-18T11:37:28Z
updated_at: 2026-03-27T10:40:55Z
---

## Summary

Refactored beans session management to be multiplexer-agnostic, replacing the abduco/PTY-specific internal session management with external script-based approach.

### What was done

1. **Deleted internal/session/** - Removed abduco-specific code
2. **Added session: config** in .beans.yml with:
   -  - Command that exits 0 if session exists
   -  - Command to attach to existing session
   -  - Command to launch/start new session

3. **Created scripts/** for tmux:
   -  - Checks if session exists
   -  - Attaches to session
   -  - Creates and attaches to session

4. **Added polling** - Round-robin polling (200ms default) for session status

5. **Added S key** - Press S on any bean to launch a session

### Key fixes

- **PTY/bubbletea issue #431**:  outputs View to terminal after command runs. Fixed by adding  flag that makes View() return empty string during external command execution.

- **tmux "terminal does not support clear"**:  was replacing the entire environment, losing TERM variable needed for terminfo. Fixed by preserving os.Environ() and appending BEANS_* vars.

## Work Log
2026-03-26 17:10: Fixed nil pointer dereference in sessionpoller.New when config.Session is nil. Build succeeds.
2026-03-27 10:30: Completed session refactor. Added tmux scripts, S key for session launch, fixed bubbletea ExecProcess clearing issue, preserved TERM env for tmux.
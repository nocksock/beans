---
# beans-e3dd
title: Refactor session management to use abduco
status: in-progress
type: task
created_at: 2026-02-18T11:37:28Z
updated_at: 2026-02-18T11:37:28Z
---

Replace the complex custom PTY implementation (~700 LOC) with a simple abduco wrapper (~100 LOC).

## Context
- Current implementation has performance/rendering issues
- Research docs recommend abduco as the optimal solution
- abduco provides: session persistence, detach (Ctrl+\), reattach, multiple sessions

## Approach
Start with requiring abduco to be installed (simplest), can add binary bundling later.

## Checklist
- [x] Create simplified session/manager.go that wraps abduco
- [x] Remove PTY-specific code (session.go, attach.go, ringbuffer.go)
- [x] Update TUI integration to use tea.ExecProcess for attach
- [x] Update session list view for new Session struct
- [x] Add helpful error if abduco not found
- [x] Bundle abduco binary (with fallback to system PATH)
- [ ] Test manually with session launchers
- [x] Clean up unused dependencies (creack/pty removed, golang.org/x/term kept for other uses)
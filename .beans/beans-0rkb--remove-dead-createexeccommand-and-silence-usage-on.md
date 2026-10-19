---
# beans-0rkb
title: Remove dead CreateExecCommand and silence usage on config errors
status: todo
type: task
created_at: 2026-10-19T19:55:37Z
updated_at: 2026-10-19T19:55:37Z
---

Found during the beans-ppxk final review; both predate that branch.

- internal/launcher/launcher.go CreateExecCommand (stdin/shebang exec path) is only used by tests (launcher_test.go, launch_test.go, internal/tui/launcher_test.go, whose comment wrongly claims the TUI uses it). Delete it with its tests and fix the comment.
- cmd/root.go PersistentPreRunE errors (config YAML errors, missing --beans-path dir) print the full usage text for every command. Set cmd.SilenceUsage = true before returning those errors.
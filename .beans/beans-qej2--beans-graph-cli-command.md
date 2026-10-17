---
# beans-qej2
title: beans deps CLI command
status: todo
type: task
priority: deferred
created_at: 2026-10-17T13:29:54Z
updated_at: 2026-10-17T14:02:56Z
parent: beans-7jm4
---

Non-interactive `beans deps <id>` printing the transitive blocked-by / blocking trees (same as the TUI dependency navigator from beans-x17f).
## Code review 2026-10-17 (beans-9pxj)

`beans graph` with --ascii and DOT already exists (cmd/graph.go). Remaining scope: use internal/graphlayout for ASCII once beans-x17f lands.

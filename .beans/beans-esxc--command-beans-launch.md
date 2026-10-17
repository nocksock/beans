---
# beans-esxc
title: 'command: beans launch'
status: todo
type: task
priority: normal
created_at: 2026-10-17T11:04:07Z
updated_at: 2026-10-17T12:13:40Z
parent: beans-gl8t
---

beans launch <id> --launcher <launcher-key>
echo <id> | beans launch --launcher <launcher-key>


## Code review 2026-10-17 (beans-9pxj)

README 400-430 documents `beans launch -l / <name> <id> / --json` but no cmd exists. Decide: implement via GraphQL, or drop the section. Also fix README launcher docs: BEANS_TASK is the title, not the file path (launch.go:34; examples 367-371 cat it); progress/stop-on-failure/5+ confirm/clear-on-success (341-345) don't exist; multi-line shebang claim broken until beans-m1x0.

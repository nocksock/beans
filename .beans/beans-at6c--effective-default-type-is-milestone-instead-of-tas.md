---
# beans-at6c
title: Effective default_type is milestone instead of task
status: todo
type: bug
created_at: 2026-10-17T13:50:53Z
updated_at: 2026-10-17T13:50:53Z
parent: beans-xqrj
---

config.applyDefaults sets Beans.DefaultType to DefaultTypes[0].Name, which is "milestone" (DefaultTypes was reordered), while Default() uses "task". So a .beans.yml without default_type yields default_type=milestone. Seen via `beans settings get beans.default_type` (beans-4wwa). Check which default is intended and whether create relies on GetDefaultType.
## Code review 2026-10-17 (beans-9pxj)

Confirmed: applyDefaults uses DefaultTypes[0] (config.go:305) = milestone; Default() says task; loadBean (core.go:191) and resolver (schema.resolvers.go:94) hardcode task. 'todo'/IDLength 4 defaults duplicated (config.go:157/302/447, core.go:334). Fix: applyDefaults copies from Default(); single source.

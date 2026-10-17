---
# beans-4wwa
title: beans settings get command
status: archived
type: task
priority: normal
created_at: 2026-10-17T10:51:33Z
updated_at: 2026-10-17T14:16:52Z
parent: beans-xqrj
blocking:
    - beans-07j8
---

eg: beans settings get path

should allow nested access

beans settings get foo.bar

also 

beans settings set foo.bar value

sets it in the nearest beans.yml

if cascading is active, error out for now. will deal with it in a different task

## Work Log
2026-10-17 15:10: Added Config.Get (dotted keys, list items by index/name) and SetSetting (yaml.Node edit keeping comments, rejects cascading files) in internal/config/settings.go
2026-10-17 15:25: Added `beans settings get|set` command (cmd/settings.go), skips core load; README section
2026-10-17 15:30: Filed beans-at6c (default_type resolves to milestone) found while testing

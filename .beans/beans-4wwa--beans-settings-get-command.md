---
# beans-4wwa
title: beans settings get command
status: todo
type: task
priority: normal
created_at: 2026-10-17T10:51:33Z
updated_at: 2026-10-17T12:13:39Z
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

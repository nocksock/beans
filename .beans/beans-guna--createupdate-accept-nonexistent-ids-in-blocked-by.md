---
# beans-guna
title: create/update accept nonexistent IDs in --blocked-by/--blocking
status: todo
type: bug
created_at: 2026-10-18T09:49:47Z
updated_at: 2026-10-18T09:49:47Z
---

beans create ... --blocked-by "" -d "body" (empty var) stored '-d' as a blocked_by ID and folded the body into the title. Blocking/blocked-by IDs should be validated to exist (or at least reject values starting with '-').
---
# beans-j7qs
title: 'command: beans migrate'
status: draft
type: task
priority: normal
created_at: 2026-10-17T11:07:15Z
updated_at: 2026-10-17T11:17:51Z
---

beans migrate --path <new-path>

--move: will move files from old path into new path
--copy: will copy files from old path into new path

command requires also either --dry or --force

first check if there are file conflicts. if yes, a flag --mine or --theirs is also required


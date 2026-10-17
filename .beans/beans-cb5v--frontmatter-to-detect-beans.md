---
# beans-cb5v
title: frontmatter to detect beans
status: draft
type: task
priority: normal
created_at: 2026-10-17T10:07:44Z
updated_at: 2026-10-17T10:17:45Z
---


type: <task/epic/feature/bug/...>
in frontmatter should suffice for a bean to show up
default status is draft

instead of having "# beans-cb5v" at the top of frontmatter, we'll do:

id: beans-cb5v

we'll add a "beans <promote?import?migrate?>" to give all the beans in the path a proper id
markdown files without an id, will, well, not have an id and the tui will indicate as such that.

a file must have at least type: <one of our types> to show up in beans tui.

but we can also pass it a list of files via stdin to that command and it will add type: task

not sure what the right verb is for that.



---
# beans-um03
title: beans link
status: scrapped
type: task
priority: normal
created_at: 2026-10-17T11:20:08Z
updated_at: 2026-10-17T12:07:56Z
---

to link a beans id to a pid/tmux. uri format.

tmux://<tmux path to window>
pid://<pid>

but this should likely not be in the md file, but rather stored in a beans-global state?
probably in ~/.local/state/

then we can `beans check <beans-id>` and use a resolver, defined in beans.yml with pid being built in.
and later maybe  think of a `beans focus <beans-id>` that uses resolver to attach to eg tmux session


---
# beans-mrl5
title: command reload config in tui
status: archived
type: task
priority: normal
created_at: 2026-10-17T11:29:34Z
updated_at: 2026-10-17T13:20:53Z
parent: beans-xqrj
---

a way to reload the config in beans tui during runtime (or auto-reloading on change)

## Work Log

2026-10-17 12:40: Explored tui/config; plan: R key + fsnotify watcher on .beans.yml, swap config in place
2026-10-17 12:58: Added Config.Reload, config.Watch, Core.ReplaceConfig; R key + auto-reload in TUI with footer notice
2026-10-17 13:00: Tests pass (race), verified manually in tmux

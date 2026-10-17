---
# beans-9p2n
title: 'init: --cascade flag'
status: todo
type: task
created_at: 2026-10-17T13:49:45Z
updated_at: 2026-10-17T13:49:45Z
parent: beans-xqrj
---

Follow-up to beans-07j8. `beans init` has flags for every scalar .beans.yml setting except `cascade`, which isn't a Config field (it's per-file and dropped on merge), so Config.Save can't write it. Add `--cascade` to init. Complex settings (statuses, launchers, keybinding) are also not covered by init flags.
## Code review 2026-10-17 (beans-9pxj)

Also: re-running init overwrites existing .beans.yml without warning (cmd/init.go:68 → Config.Save os.WriteFile); refuse unless --force. Stale comment init.go:30. Check on dev.

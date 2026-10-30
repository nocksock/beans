---
# beans-9rh8
title: Copy bean ID with y in the TUI
status: archived
type: feature
priority: normal
size: sm
created_at: 2026-10-23T12:00:40Z
updated_at: 2026-10-23T12:33:56Z
parent: beans-zkj9
---

Add a configurable `copyID` keybinding, `y` by default, that copies the selected bean ID to the clipboard.

- `y` is free in the list context. It is only used as "confirm" in the archive-confirm and confirm dialogs (internal/tui/keys.go:394,401).
- Upstream reference: commit a3d95eb.
## Work Log
2026-10-23 14:15: Reviewed keymap, notice system and upstream a3d95eb; using atotto/clipboard (already in go.mod)
2026-10-23 14:16: Added copyID (y) to list+context keymaps, clipboard.go with injectable writer, notices, tests, README
2026-10-23 14:16: Committed 38ce14a on task/beans-9rh8; go test and go vet pass
2026-10-23 10:15: Verified tests, merged into task/beans-agzc (985864f); dev merge pending
2026-10-23 10:40: Merged into dev (96ed031)

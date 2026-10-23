---
# beans-9rh8
title: Copy bean ID with y in the TUI
status: in-progress
type: feature
priority: normal
size: sm
created_at: 2026-10-23T12:00:40Z
updated_at: 2026-10-23T12:13:35Z
parent: beans-zkj9
---

Add a configurable `copyID` keybinding, `y` by default, that copies the selected bean ID to the clipboard.

- `y` is free in the list context. It is only used as "confirm" in the archive-confirm and confirm dialogs (internal/tui/keys.go:394,401).
- Upstream reference: commit a3d95eb.
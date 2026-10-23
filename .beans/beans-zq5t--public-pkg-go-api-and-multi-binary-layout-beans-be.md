---
# beans-zq5t
title: Public pkg/ Go API and multi-binary layout (beans, beans-tui)
status: todo
type: feature
size: xl
created_at: 2026-10-23T12:00:40Z
updated_at: 2026-10-23T12:00:40Z
parent: beans-zkj9
---

Breaking: Go import paths change.

- Move internal/{bean,beancore,config,graph,...} to a public pkg/ API.
- Split the TUI into its own beans-tui binary.
- Leave beans-serve out for now.

Reference: upstream 4900811 (the pkg move) and 84b4afc (the binary split).

Do this early: it touches the files every other bean in this epic touches.
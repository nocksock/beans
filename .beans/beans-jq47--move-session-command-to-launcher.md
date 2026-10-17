---
# beans-jq47
title: move session command to launcher
status: completed
type: task
priority: high
created_at: 2026-10-17T10:45:34Z
updated_at: 2026-10-17T12:17:20Z
parent: beans-gl8t
blocking:
    - beans-hmyr
---

should just be a launcher instead of a first level feature (so we don't need to own tmux integration)

## Work Log
2026-10-17 14:16: Plan: drop session config/poller/dialog/S key; move sh-exec helpers to launcher pkg
2026-10-17 14:17: Removed session config, sessionpoller, session dialog, S key, running indicators; tests pass
2026-10-17 14:17: Committed; scrapped beans-hmyr (poller gone)
2026-10-17 14:19: Hub: removed session config + unused session scripts

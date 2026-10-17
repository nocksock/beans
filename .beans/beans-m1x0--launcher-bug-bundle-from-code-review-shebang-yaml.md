---
# beans-m1x0
title: Launcher bug bundle from code review (shebang, YAML, hang)
status: todo
type: bug
priority: normal
size: md
tags:
    - code-review
created_at: 2026-10-17T14:02:00Z
updated_at: 2026-10-17T14:10:28Z
parent: beans-gl8t
---

From review beans-9pxj (`docs/reviews/2026-10-17-code-quality-review.md` in hub), items B13, B16, B17. (B10 multi foreground → beans-yxao, B15 README → beans-esxc.)

- [ ] B13: multi-line shebang launchers always run via `sh -c` (`exec.go:11`) → `#!` is just a comment; README Ruby example fails, bashisms break on dash. For `#!` scripts: temp executable file (clean up after Wait) or `interp /dev/fd/3` via ExtraFiles (stdin must stay free for foreground). Test with `#!/usr/bin/env python3`. Unblocks deleting `launcher/launcher.go`.
- [ ] B16: `appendLaunchersToConfig` builds YAML via string formatting (`tui/launcher.go:96-101`) → invalid YAML for quotes/colons/#/newlines. Marshal `[]config.Launcher` with yaml.v3 (coordinate with beans-w0ua).
- [ ] B17: foreground run can hang if detached grandchild keeps stderr pipe (MultiWriter, `exec.go:29`). Set `cmd.WaitDelay`.
- [ ] Tests: tailLines with CR progress output and >16KiB offset
## Sequencing (review beans-9pxj)

Step 3 of 5: after beans-bufq; unblocks deleting launcher.go in beans-re37.

## Note from beans-esxc
README now says multi-line scripts run via `sh -c` (shebang ignored) and the Ruby example was removed. When B13 is fixed, restore the README "How It Works" shebang description and add a non-sh example back.

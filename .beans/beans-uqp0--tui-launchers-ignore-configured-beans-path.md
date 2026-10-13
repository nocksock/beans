---
# beans-uqp0
title: TUI launchers ignore configured beans path
status: completed
type: bug
priority: normal
created_at: 2026-10-12T16:00:08Z
updated_at: 2026-10-12T16:02:02Z
---

Launcher code in the TUI treats core.Root() (the beans data dir) as the project root: config append reads <beansdir>/.beans.yml, BEANS_TASK is built as <root>/.beans/beans-<id>.md, and launchers run/resolve relative to the beans dir. Breaks when beans.path is customized.
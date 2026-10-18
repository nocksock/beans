---
# beans-0ggp
title: overhaul ui
status: draft
type: task
priority: normal
created_at: 2026-10-18T09:25:35Z
updated_at: 2026-10-18T09:38:44Z
---

panes like lazygit
with key +/- zooming in/changing layouts.

default is


    [task-list] | task-view
    ---------------
    log

[] marks focus

there is no dedicated detail view
pressing enter moves focus to task-view

pressing + will change view to

    [task-list]
    ----------
    log

pressing + again will change view to

    [task-list]

pressing + again will change view back to

    [task-list] | task-view
    ---------------
    log


when focus is on task view we cycle
from

    task-list | [task-view]
    ---------------
    log

to

    dependecy-navigator
    -----
    [task-view]
    ------
    log

    [task-view]

log and dependecy-navigator (or rather anything) can be toggled on and off anywhere
+/- are basically cycling through presets of toggled states
however we might change these often while iterating, so we need a very modular and simple way to compose these view.


panes we'd like to have:
- context (detail view for whatever is currently selected in another pane)
- task list
- task view (displayed in context when task list is focused with selected task)
- log (navigatable, when focussed show log detail in context pane)
- log detail (output of a command)
- dependecy-graph (was: dependecy-navigator)
- task history (navigatable git log, when focussed show task diff in context )
- task diff (displayed in context when history is in focus)


i can also imagine that we have 2 task views open + dependecy-graph. 
where one task views the current task, and the other the one selected in dependency graph.


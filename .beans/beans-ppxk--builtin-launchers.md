---
# beans-ppxk
title: builtin launchers
status: draft
type: task
priority: normal
created_at: 2026-10-19T07:26:37Z
updated_at: 2026-10-19T08:34:56Z
---

move some launchers to scripts/folder in deliverable (embed in binary?)

i'm thinking of cascading them as well

child can call:

    beans launch parent.tmux-window ; of parent beans.yml
    beans launch builtin.tmux-window ; og beans

(also rename from launcher to launch)

so they can be composed somehow in a launcher

writing those with bash feels appropriate, but writing in go might also be worth considering for the builtins


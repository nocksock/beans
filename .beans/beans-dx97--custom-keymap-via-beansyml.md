---
# beans-dx97
title: custom keymap via .beans.yml
status: in-progress
type: task
priority: deferred
created_at: 2026-10-17T10:18:56Z
updated_at: 2026-10-17T12:56:55Z
parent: beans-7jm4
---

but also via a something like ~/.config/beans/config.yml 
very similar to the format that lazygit has

## Plan

### Decisions
- **Global config**: `$XDG_CONFIG_HOME/beans/config.yml` (fallback `~/.config/beans/config.yml`) is a full base layer, merged *below* the `.beans.yml` cascade chain. Any setting may go there; project files override it. `cascade` is meaningless there and is ignored; `beans.path` in the global file is rejected (it would point every project at one dir).
- **Scope**: every key handler is rebindable: global, list, detail, help overlay, and all pickers/modals/dialogs.
- **Naming**: config keys mirror the Go identifiers. Context = keymap struct (`ListKeyMap` → `list`), action = field name in lowerCamel (`ChangeStatus` → `changeStatus`), derived mechanically (reflection or a generated table), so they stay in sync.
- **Values**: a string or a list of strings (`quit: q` / `quit: [q, ctrl+c]`). `<disabled>` or `[]` unbinds. Key names use bubbletea `KeyMsg.String()` syntax. Space-separated chords (`g t`) are allowed.
- Format, lazygit-style:
  ```yaml
  keybinding:
    global:  { quit: [q, ctrl+c], help: "?", reloadConfig: R }
    list:    { changeStatus: s, filterByTag: "g t" }
    detail:  { prevBean: h, nextBean: l }
    picker:  { confirm: enter, cancel: esc }
  ```
- Defaults: overriding an action replaces its keys (no append). Unknown context/action names fail validation at load. Duplicate keys within one context produce a warning, not an error. The keymap reloads with `R` like the rest of the config.

### Steps
- [x] config: add a user-global layer at the bottom of `findChain`, with tests (XDG/HOME, missing file, override order, rejected `beans.path`)
- [x] config: `Keybinding` section (string-or-list unmarshal, `<disabled>`), validation against the known action table
- [x] tui: central keymap registry per context (`key.Binding` with keys + help text), defaults equal to today's hardcoded keys
- [x] tui: replace every `msg.String()` switch with `key.Matches` lookups: tui.go, list.go, detail.go, help.go, then each picker/modal. Text-input modals only bind non-printable keys so typing still works.
- [x] tui: chord support (`g t`) through the registry
- [x] tui: generate the help overlay and footers from the keymap so they show the actual bindings
- [x] wire through config reload (`R`)
- [x] docs: README/config docs section with the full action list and defaults
- [x] tests: table-driven tests for parsing/merging/validation, plus TUI tests that a rebound key triggers its action and the old key doesn't

## Work Log
2026-10-17 14:57: Planned with user, plan written, set in-progress
2026-10-17 15:04: Config: user-global layer + keybinding section done (tui validates names)
2026-10-17 15:13: TUI keymap registry, all handlers/footers/help converted, chords, reload, README, tests (ad91f22)
2026-10-17 15:13: Manually verified in tmux: rebinding, chords, disabled actions, conflict warning, invalid name error

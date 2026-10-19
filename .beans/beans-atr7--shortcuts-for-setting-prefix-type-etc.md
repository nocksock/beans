---
# beans-atr7
title: shortcuts for setting prefix, type etc
status: in-progress
type: task
priority: normal
created_at: 2026-10-19T10:52:10Z
updated_at: 2026-10-19T12:20:42Z
---

Field shortcuts and type-ahead for the bean form (create/edit modal).

Original idea: maybe prefix ctrl+g t (for type), s for status and so on? when focus also typing should select the value that starts with typed

## Plan

### Selector modes (prefix, type, status, priority, size)

**Navigation mode** (default when tabbing onto a selector):
- `h`/`l` move focus to prev/next selector column; `k` moves up to description, `j` from description back down to the selectors
- arrows / `space` cycle the value, `e` save & edit, `enter` save, `esc` cancels the form
- `i` enters type-ahead mode; any unbound letter also enters it and is used as the first typed letter (`f` -> feature)

**Type-ahead mode** (via `ctrl+g <key>`, `i`, or an unbound letter):
- all letters feed a case-insensitive prefix buffer; selection jumps to the first value starting with it
- no match: key ignored, buffer unchanged
- buffer resets after ~1s idle (timestamp on the model, no tick cmd)
- repeating the same single letter cycles through values starting with it
- arrows / `space` still cycle, `enter` saves, `esc` returns to navigation mode (second `esc` cancels the form)
- focused field gets a visual cue (accent/underline) while in type-ahead mode

### Jump chords
- `ctrl+g n/d/i/t/s/P/S` -> focus title / description / ID prefix / type / status / priority / size
- work from any field, including while typing in the title; a selector reached this way starts in type-ahead mode
- `focusPrefix` is a no-op when the prefix field is hidden
- new rebindable `CreateModal` actions (`focusTitle`, `focusBody`, `focusPrefix`, `focusType`, `focusStatus`, `focusPriority`, `focusSize`)
- chord plumbing: `chordBindings()` returns the `CreateModal` bindings for `viewCreateModal` even though it is a typing view; only chord-starting keys (`ctrl+g`) are held back, so normal typing is unaffected; unknown second key is a no-op

### Keymap changes
- `cyclePrev`/`cycleNext` lose `h/j/k/l` (keep arrows, `space`)
- new `navLeft/navRight/navUp/navDown` (`h/l/k/j`), `typeAhead` (`i`), `exitTypeAhead` (`esc`)
- `quickSubmitAndEdit` (`e`) stays -> no breaking config change

## Todo
- [ ] Chord support in the create modal (`chordBindings` for `viewCreateModal`)
- [ ] Jump chord bindings + focus handling
- [ ] Selector navigation/type-ahead modes and hjkl field navigation
- [ ] Type-ahead prefix buffer with timeout and same-letter cycling
- [ ] Visual cue for type-ahead mode, help/footer hints
- [ ] Table-driven tests: type-ahead matching/cycling/reset, mode transitions incl. double esc, hjkl nav incl. row crossing, each jump chord (also from title), `t` in title stays text, keymap has no conflicts
- [ ] Update docs/README keybinding docs if they list form keys

## Work Log
2026-10-19 13:00: Planned with user; design agreed

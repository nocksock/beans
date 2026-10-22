---
# beans-atr7
title: shortcuts for setting prefix, type etc
status: archived
type: task
priority: normal
created_at: 2026-10-19T10:52:10Z
updated_at: 2026-10-19T20:46:15Z
---

Field shortcuts and type-ahead for the bean form (create/edit modal).

Original idea: maybe prefix ctrl+g t (for type), s for status and so on? when focus also typing should select the value that starts with typed

## Plan

### Selector modes (prefix, type, status, priority, size)

**Navigation mode** (default when tabbing onto a selector):
- `h`/`l` move focus to prev/next selector column (stopping at the ends); `k` moves up to the description; `j` is a no-op (selectors are the bottom row; the description is a text field, so `j` types there)
- arrows / `space` cycle the value, `e` save & edit, `enter` save, `esc` cancels the form
- `i` enters type-ahead mode; any unbound letter also enters it and is used as the first typed letter (`f` -> feature)

**Type-ahead mode** (via `ctrl+g <key>`, `i`, or an unbound letter):
- all letters feed a case-insensitive prefix buffer; selection jumps to the first value starting with it
- no match: key ignored, buffer unchanged
- buffer resets after ~1s idle (timestamp on the model, no tick cmd)
- repeating the same single letter cycles through values starting with it
- arrows / `space` still cycle, `enter` saves, `esc` returns to navigation mode (second `esc` cancels the form); `esc` is the `cancel` binding, which leaves type-ahead first, to avoid a keymap conflict
- focused field gets a visual cue (accent/underline) while in type-ahead mode

### Jump chords
- `ctrl+g n/d/i/t/s/P/S` -> focus title / description / ID prefix / type / status / priority / size
- work from any field, including while typing in the title; a selector reached this way starts in type-ahead mode
- `focusPrefix` is a no-op when the prefix field is hidden
- new rebindable `CreateModal` actions (`focusTitle`, `focusBody`, `focusPrefix`, `focusType`, `focusStatus`, `focusPriority`, `focusSize`)
- chord plumbing: `chordBindings()` returns the `CreateModal` bindings for `viewCreateModal` even though it is a typing view; only chord-starting keys (`ctrl+g`) are held back, so normal typing is unaffected; unknown second key is a no-op

### Keymap changes
- `cyclePrev`/`cycleNext` lose `h/j/k/l` (keep arrows, `space`)
- `cyclePrev`/`cycleNext` also get `ctrl+p`/`ctrl+n`
- new `navLeft/navRight/navUp/navDown` (`h/l/k/j`), `typeAhead` (`i`)
- unknown chords are now dropped instead of passed on (otherwise `ctrl+g x` would be typed into the title)
- `quickSubmitAndEdit` (`e`) stays -> no breaking config change

## Todo
- [x] Chord support in the create modal (`chordBindings` for `viewCreateModal`)
- [x] Jump chord bindings + focus handling
- [x] Selector navigation/type-ahead modes and hjkl field navigation
- [x] Type-ahead prefix buffer with timeout and same-letter cycling
- [x] Visual cue for type-ahead mode, help/footer hints
- [x] Table-driven tests: type-ahead matching/cycling/reset, mode transitions incl. double esc, hjkl nav incl. row crossing, each jump chord (also from title), `t` in title stays text, keymap has no conflicts
- [x] Update docs/README keybinding docs if they list form keys

## Work Log
2026-10-19 13:00: Planned with user; design agreed
2026-10-19 15:05: Implemented chords, selector modes, type-ahead; tests + README; verified in real TUI via tmux

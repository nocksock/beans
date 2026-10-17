---
# beans-nhal
title: size attribute
status: completed
type: task
priority: normal
created_at: 2026-10-17T13:42:20Z
updated_at: 2026-10-17T14:09:34Z
---

Add a `size` attribute to beans: `xs sm md lg xl` (smallest → largest). Default is `md`.

## Decisions

- Values: `xs sm md lg xl`, hardcoded like priorities (not configurable in `.beans.yml`).
- Default is implicit: empty/missing `size` in frontmatter means `md`; nothing is written for unset beans. Existing beans untouched.
- Mirrors the priority pattern end-to-end.

## Plan

### Data model
- [x] `config.DefaultSizes` (`SizeConfig{Name, Color, Description}`), ordered xs→xl
- [x] Config helpers: `SizeNames()`, `IsValidSize()`, `GetSize()`, `SizeList()`
- [x] `bean.Bean.Size` → frontmatter `size,omitempty` (empty round-trips as empty)

### GraphQL (`mise codegen`)
- [x] `Bean.size: String!` resolves effective value (empty → `md`)
- [x] `size` on create/update inputs; validated; empty clears
- [x] `size` / `excludeSize` on `BeanFilter`; filtering `md` also matches empty

### CLI
- [x] `beans create --size/-S`, `beans update --size/-S` (empty clears)
- [x] `beans list --size`, `--no-size`, `--sort size` (xs→xl, empty = md); default sort unchanged
- [x] `beans show` header shows size next to priority
- [x] `beans prime` docs mention size

### TUI
- [x] List: 3-wide size column after type, always shown (incl. `md`); works in compact mode; include in tags-width math
- [x] Create/edit modal: Size selector after Priority (←/→ cycles), default `md`; edit only sends size if changed
- [x] `sizepicker.go` (modeled on `prioritypicker.go`), key `S` in list (supports marked beans) and detail
- [x] Help/hint bars include `S`
- [x] Size shown in detail/preview header

### Tests
- [x] Config validation, frontmatter round-trip, GraphQL filter (md matches empty), size sort
- [x] TUI: modal field, column render, picker

## Work Log

2026-10-17 15:00: Planned with user (brainstorming); plan written, set to in-progress
2026-10-17 15:20: Added config sizes, bean.Size, GraphQL size field/inputs/filters + tests
2026-10-17 15:45: Added CLI --size flags (create/update/list), --sort size, show header, prime docs
2026-10-17 16:05: Added TUI size column, modal Size selector, S size picker, detail/preview headers + tests
2026-10-17 16:10: No -S short flag (clashes with list --search)
2026-10-17 16:25: Added SZ column to `beans list` tree output; shared ui.SizeColor helper

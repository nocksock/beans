---
# beans-nhal
title: size attribute
status: in-progress
type: task
priority: normal
created_at: 2026-10-17T13:42:20Z
updated_at: 2026-10-17T13:53:05Z
---

Add a `size` attribute to beans: `xs sm md lg xl` (smallest → largest). Default is `md`.

## Decisions

- Values: `xs sm md lg xl`, hardcoded like priorities (not configurable in `.beans.yml`).
- Default is implicit: empty/missing `size` in frontmatter means `md`; nothing is written for unset beans. Existing beans untouched.
- Mirrors the priority pattern end-to-end.

## Plan

### Data model
- [ ] `config.DefaultSizes` (`SizeConfig{Name, Color, Description}`), ordered xs→xl
- [ ] Config helpers: `SizeNames()`, `IsValidSize()`, `GetSize()`, `SizeList()`
- [ ] `bean.Bean.Size` → frontmatter `size,omitempty` (empty round-trips as empty)

### GraphQL (`mise codegen`)
- [ ] `Bean.size: String!` resolves effective value (empty → `md`)
- [ ] `size` on create/update inputs; validated; empty clears
- [ ] `size` / `excludeSize` on `BeanFilter`; filtering `md` also matches empty

### CLI
- [ ] `beans create --size/-S`, `beans update --size/-S` (empty clears)
- [ ] `beans list --size`, `--no-size`, `--sort size` (xs→xl, empty = md); default sort unchanged
- [ ] `beans show` header shows size next to priority
- [ ] `beans prime` docs mention size

### TUI
- [ ] List: 3-wide size column after type, always shown (incl. `md`); works in compact mode; include in tags-width math
- [ ] Create/edit modal: Size selector after Priority (←/→ cycles), default `md`; edit only sends size if changed
- [ ] `sizepicker.go` (modeled on `prioritypicker.go`), key `S` in list (supports marked beans) and detail
- [ ] Help/hint bars include `S`
- [ ] Size shown in detail/preview header

### Tests
- [ ] Config validation, frontmatter round-trip, GraphQL filter (md matches empty), size sort
- [ ] TUI: modal field, column render, picker

## Work Log

2026-10-17 15:00: Planned with user (brainstorming); plan written, set to in-progress
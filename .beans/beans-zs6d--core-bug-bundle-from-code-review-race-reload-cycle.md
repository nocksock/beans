---
# beans-zs6d
title: Core bug bundle from code review (race, reload, cycles, ID collision)
status: todo
type: bug
priority: high
size: lg
tags:
    - code-review
created_at: 2026-10-17T14:01:43Z
updated_at: 2026-10-17T14:10:28Z
parent: beans-c8p4
---

From review beans-9pxj (`docs/reviews/2026-10-17-code-quality-review.md` in hub), items B1–B4 + core parts of B14/B20.

- [ ] B1: GraphQL mutations edit the live cached `*Bean` outside `c.mu` → race with TUI/watcher; memory≠disk on failed save. Also `RemoveLinksTo`/`FixBrokenLinks` skip UpdatedAt; `cmd/list.go:164` blanks cached Body. Fix: `Core.Update(id, func(*bean.Bean) error)` (mutate under lock, commit after write) or clone-on-get. (`schema.resolvers.go:148-173`, `core.go:368-387`, `links.go:386-499`)
- [ ] B2: `loadFromDisk` clears map then aborts on first unparsable file → beans vanish in TUI while an editor saves; watcher errors swallowed (`watcher.go:120,136`). Build new map + swap on success, skip+warn bad files. Also `watchLoop` reads `c.done` unlocked → goroutine leak after Unwatch/Watch (`watcher.go:75`): pass done/onChange as params.
- [ ] B3: cycle path aliasing — `append(path[cycleStart:], id)` shares backing array, sibling DFS overwrites stored cycle (reproduced). `slices.Clone` + regression test. (`links.go:312,338`)
- [ ] B4: `Create` never checks for ID collision → silent overwrite (`core.go:332-341`)
- [ ] Slug truncation `s[:50]` splits UTF-8; regex compiled per call (`bean/id.go:76,83`)
- [ ] Legacy single-dash fallback parses `beans-abcd.md` as id `beans` (`bean/id.go:42-47`)
- [ ] Unstable sort without ID tie-break (`bean/sort.go:448`)
- [ ] Tests: parse failure during reload, cycle regression, concurrent Update vs All under -race
## Sequencing (review beans-9pxj)

Step 1 of 5 for B2/B3/B4 (small); B1 (race / Update API) belongs to step 2 together with beans-rpmz and beans-ole9.

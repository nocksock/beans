---
# beans-tbbz
title: cascading ux improvements
status: in-progress
type: task
priority: normal
created_at: 2026-10-18T09:04:50Z
updated_at: 2026-10-18T09:13:20Z
---

when running beans tui in a folder, its lists etc should be scoped to that config's prefix.

give an example worktree in this task and I'll tell you how it'll be resolved.
add edge cases.


## Example tree

```
repo/
├── .beans.yml            # cascade: true, beans.path: .beans, prefix: repo-
├── .beans/               # shared dir: repo-*, api-*, api-v2-*, web-*, old-* beans
├── packages/
│   ├── api/
│   │   ├── .beans.yml    # cascade: true, prefix: api-
│   │   └── v2/
│   │       └── .beans.yml   # prefix: api-v2-   (overlaps api-)
│   ├── web/
│   │   ├── .beans.yml    # prefix: web-
│   │   └── src/          # no config; cwd here
│   ├── shared/
│   │   └── .beans.yml    # no prefix set → inherits repo-
│   ├── tools/
│   │   └── .beans.yml    # prefix: ""  (explicitly empty)
│   └── mobile/
│       ├── .beans.yml    # prefix: mob-, beans.path: .beans (own dir)
│       └── .beans/       # mob-* beans, plus a stray repo-* bean
└── vendor/lib/
    └── .beans.yml        # prefix: lib-, NOT cascading from repo (repo cascades, but lib could also set its own path or not)
```

Beans in `repo/.beans`: `repo-a1b2` (milestone), `api-c3d4` (parent: repo-a1b2), `api-v2-e5f6`, `web-g7h8` (blocked-by api-c3d4), `old-i9j0` (from before the prefix was renamed).

## Questions: what does `beans tui` show when run in…

1. `repo/` — all beans in `.beans`, or only `repo-*`?
2. `packages/api/` — `api-*` only. But `api-v2-e5f6` also starts with `api-`: in or out?
3. `packages/api/v2/` — `api-v2-*` only (straightforward).
4. `packages/web/src/` — no config here; nearest is web → `web-*`?
5. `packages/shared/` — inherited prefix `repo-` → same scope as root?
6. `packages/tools/` — empty prefix → no scoping (everything)?
7. `packages/mobile/` — own dir: everything in it, or still filter to `mob-*` (hides the stray `repo-*`)?
8. `vendor/lib/` — what if it shares `repo/.beans` via `path: ../../.beans` without cascade?

## Edge cases

- **Cross-scope relations**: `api-c3d4` has parent `repo-a1b2` (out of scope). Tree view: hide parent, show it greyed as context, or show child as root?
- **Blocking**: `web-g7h8` blocked by `api-c3d4`. In `web/`, still counted as blocked (yes, I'd assume). Shown in detail/dependency navigator?
- **Orphaned prefixes**: `old-i9j0` matches no config → only visible at root (if root shows all) or never.
- **Overlapping prefixes** (`api-` vs `api-v2-`): match by `prefix + id_length` exact shape, or longest-known-prefix wins (requires knowing sibling configs → beans-h0g0)?
- **Escape hatch**: toggle in TUI (key) to show all beans, plus a header indicator showing the active scope? CLI flag `--all`?
- **CLI**: does `beans list` / `beans query` scope too, or only the TUI? (CLAUDE.md: CLI goes through GraphQL → probably a new `BeanFilter.idPrefix`/scope field.)
- **Direct lookups**: `beans show repo-a1b2` from `api/` should still work — scope only affects lists?
- **Creating in TUI**: new bean gets `api-` prefix already; creating a child of an out-of-scope parent?
- **`--beans-path` / `--config` overrides**: scope from the loaded config's prefix as usual?

## Decisions (2026-10-18)

- Cascade root lists everything in its beans dir. Only nested configs are scoped.
- Subtree semantics: plain string prefix match, so `api-` includes `api-v2-*`.
- Out-of-scope parents show as greyed context rows in the tree (same as filtered-out ancestors).
- TUI: toggle key for scoped/all, plus a header indicator of the active scope.
- CLI `beans list` is scoped too, through a new GraphQL filter field, with `--all` to opt out.
- Lookups by ID (show/update/etc.) are never scoped.
- Create dialog gets a **prefix** field. When the TUI's folder covers several scopes (e.g. the repo root covers `repo-` and `api-`), you pick the prefix for the new bean there. With only one available, the field still renders but is fixed, so the layout stays the same.

## Plan

**Scope rule**: a config is scoped when its chain has a cascading ancestor (it isn't the chain root) and its effective prefix is non-empty. The scope is that prefix.

Resulting answers for the example tree:
1. `repo/` → everything (chain root)
2. `packages/api/` → `api-*` incl. `api-v2-*`
3. `packages/api/v2/` → `api-v2-*`
4. `packages/web/src/` → nearest config is web → `web-*`
5. `packages/shared/` → inherited `repo-` → `repo-*` only (not everything)
6. `packages/tools/` → empty prefix → unscoped
7. `packages/mobile/` → `mob-*` (stray `repo-*` hidden, visible via toggle)
8. `vendor/lib/` → it sits under the cascading root, so it inherits and is scoped to `lib-*`. Only configs outside any cascade (or below a non-cascading config) are unscoped
- Orphaned prefixes (`old-*`): visible at the root and via the toggle only.
- Blocking status still considers out-of-scope blockers.
- Creating a child of an out-of-scope parent is allowed; the new bean gets the scoped prefix.

### Steps
- [x] config: `Config.Scope() string`, which returns the prefix or "" per the rule above, with table tests over the example tree
- [x] graphql: `BeanFilter.idPrefix: String` (string prefix match), `mise codegen`, resolver tests
- [x] cli: `beans list` passes `idPrefix` from `cfg.Scope()`; add `--all`; tests
- [x] tui: apply scope to the list query; out-of-scope ancestors as greyed context rows; toggle key (configurable keybinding) and a header scope indicator
- [x] config: `Config.AvailablePrefixes() []string`: own prefix first, then prefixes of descendant configs that cascade from this one and share its beans dir (needs downward discovery; overlaps beans-h0g0). Tests over the example tree
- [x] graphql: `CreateBeanInput.prefix: String` (defaults to the config prefix), passed through to beancore ID generation (`internal/beancore/core.go:336`); `mise codegen`; tests
- [x] tui: prefix field in `internal/tui/createmodal.go`, selectable when >1 prefix is available and fixed otherwise; default is own prefix; tests
- [x] docs: README monorepo section on scoping

## Work Log
2026-10-18 11:16: config: added Scope() and AvailablePrefixes() with tests
2026-10-18 11:17: graphql: BeanFilter.idPrefix + CreateBeanInput.prefix with tests
2026-10-18 11:18: cli: beans list scoped, --all flag
2026-10-18 11:20: tui: list scoped, g s toggles all scopes, title indicator
2026-10-18 11:21: tui: prefix field in create dialog, tests
2026-10-18 11:23: docs + manual test in scratch monorepo; filed beans-vjqd (default type milestone)

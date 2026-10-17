---
# beans-jzty
title: cascading .beans.yml
status: completed
type: task
priority: normal
tags:
    - idea
created_at: 2026-10-17T10:12:42Z
updated_at: 2026-10-17T11:05:00Z
blocking:
    - beans-h0g0
---

so we can just

    beans: prefix: something-

in a .beans.yml in a subfolder. the settings then apply only to that subfolder, but the settings of the closest parent .beans.yml apply as well.

    cascade: true

will enable this feautre for all sub-beans.yml files.

    projects/
        .beans.yml      // has no cascade settings (default = false)
        a/.beans.yml    // sets cascade true, does not take settings from parent .beans.yml
        a/aa/.beans.yml // will take settings from parent in projects/a/.beans.yml
        a/ab/.beans.yml // will take settings from parent in projects/a/.beans.yml
        b/.beans.yml    // does not have settings from parent

so that this becomes highly compatible with monorepos. setting a .beans.yml with shared settings like launchers etc in project root and in packages or apps we can just define a separate prefix or dedicated path and the rest is available too

since will likely allow other formats than .yml too, the cascading needs to happen on a data structure level

## Spec (phase 1: discovery-time cascade)

Phase 2 (aggregated view from a root) is tracked in beans-h0g0.

### Discovery
1. Walk upward from cwd (or `--config`) to the nearest `.beans.yml` = **leaf** (unchanged behavior).
2. Continue upward to the next `.beans.yml`. If it has `cascade: true`, add it to the chain and repeat; otherwise stop (it is not included).
3. Merge farthest → leaf; closest wins.
- `cascade` is per-file and never inherited — it only says "my settings flow down".
- No artificial boundary: chain ends at first non-cascading file or filesystem root.

### Merge (format-agnostic)
- Each file is parsed into a generic `map[string]any`; merge happens on these maps, then the result is decoded into `Config`. Future TOML/JSON loaders only need to produce the same map.
- Maps: recursive merge.
- Scalars: closer file overrides.
- Lists whose items all have a `name` key: merge by name — child entry replaces same-named parent entry, new names appended (parent order first).
- Other lists: replaced wholesale.
- Out of scope for v1: removing inherited values (`null` deletion, `disabled: true` on keyed list items) — follow-up.

### Paths
- Relative path values resolve against the directory of the **file that declared them**. Inherited `beans.path: .beans` from root → `<root>/.beans`; a package setting only `prefix` shares the root beans dir.
- If no file in the chain sets `beans.path`, default `.beans` resolves against the leaf.
- Defaults (`id_length`, `default_status`, …) applied once, after merging.

### Launchers / session commands
- Each launcher/session command remembers its source file's directory; it runs with that as cwd so inherited `./scripts/...` keep working.
- `BEANS_ROOT` stays the leaf config dir ("this project").
- New env var `BEANS_CONFIG_DIR` = directory of the file that declared the launcher.

### Testing
- Table-driven tests for the merge function (maps, scalars, keyed lists, plain lists).
- Discovery tests with temp dir trees covering the example above (non-cascading root, transitive chain, chain broken in the middle).
- Path resolution tests: inherited `beans.path`, default path, launcher cwd/`BEANS_CONFIG_DIR`.

## Implementation Plan

### Layers
Domain view of `config.Load*`: find the chain → read each file as a layer → merge layers → build Config (decode, record origins, defaults, validate). Each step is its own unit; `Load` only orchestrates.

| Unit | File | Answers "what" |
|---|---|---|
| `readLayer(path) (layer, error)` | `config/layer.go` | reads one config file into `layer{dir, data map[string]any}` — the only place that knows YAML (future formats branch here) |
| `findChain(leafPath) ([]layer, error)` | `config/chain.go` | collects the leaf plus cascading ancestors, root-first |
| `mergeData(base, over map[string]any) map[string]any` | `config/merge.go` | merges two layers' data; generic, knows nothing about beans |
| `originDir(layers, keys...)` / `originDirOfNamed(layers, listKey, name)` | `config/origin.go` | which file (dir) a setting came from |
| `fromLayers(layers) (*Config, error)` | `config/config.go` | builds a Config: merge, drop `cascade`, decode, set origin dirs, `applyDefaults`, `validate` |
| `launcher.ExecContext` | `launcher/launcher.go` | everything a launch needs (script, work dir, project root, beans dir, bean) — replaces guessing the project root from the beans dir |

### Steps (TDD: test first each step)
- [x] 1. `mergeData` + table tests: nested maps, scalar override, keyed lists merged by `name` (override + append, parent order kept), plain lists replaced, type mismatch → override wins
- [x] 2. `readLayer` + `cascades(layer)`; tests: valid file, empty file, invalid YAML error, `cascade` absent/false/true
- [x] 3. `findChain` on temp dir trees: the example tree above (non-cascading root, `a` cascades, `b` isolated), transitive chain, chain broken in the middle, no ancestors
- [x] 4. `originDir` / `originDirOfNamed` + tests (closest layer wins, missing → "")
- [x] 5. Refactor `Load(path)` / `LoadFromDirectory` onto `findChain` + `fromLayers`; extract `applyDefaults` and `validate` from `Load`. Decode by re-marshalling the merged map through yaml. Explicit `--config` cascades too. All existing config tests must pass unchanged (single layer = old behavior)
- [x] 6. Paths: unexported `beansPathDir` = `originDir(layers, "beans", "path")`, falling back to the leaf; `ResolveBeansPath` uses it. Tests: inherited path → root, leaf override, default → leaf
- [x] 7. Origins on commands: `Launcher.Dir` (`yaml:"-"`) via `originDirOfNamed`; `SessionConfig.WorkDir(cmd)` per check/attach/launch via `originDir`. Tests
- [x] 8. Exec: `launcher.CreateExecCommand(ExecContext)` sets cwd = WorkDir, `BEANS_ROOT` = ProjectRoot (leaf `ConfigDir()`), new `BEANS_CONFIG_DIR` = WorkDir. Move `sessionpoller.CreateExecCommand` and the poller check onto the same context (drop the `/.beans` suffix guessing). Update TUI call sites (`tui.go` runLauncher/session, `launcher.go`) and tests
- [x] 9. Docs: document `cascade`, merge rules, path rule, `BEANS_CONFIG_DIR` wherever `.beans.yml` is documented (README, `beans prime` output if it lists config keys)
- [x] 10. Manual test in a scratch tree with `go run . --beans-path`/cwd variations; `go test ./...`

### Notes
- `Config.Save` marshals the whole struct; it must never be called on a merged config (today only `init` calls it, with `Default()`). Add a doc comment saying so.
- `appendLaunchersToConfig` (TUI) writes to the leaf file. That's correct, since leaf launchers merge by name.
- Out of scope, file separately: the session poller's check runs without `os.Environ()` and sets `BEANS_DIR` to the project root.

## Work Log
2026-10-17 12:41: Refined spec via brainstorming; created phase-2 bean beans-h0g0
2026-10-17 12:48: Added implementation plan (layered units, 10 TDD steps)
2026-10-17 13:05: Steps 1-7 done: config cascade (merge/layer/chain/origin), commit 8b43ad9
2026-10-17 13:05: Step 8 done: launcher.Launch, BEANS_CONFIG_DIR, commit deca916
2026-10-17 13:05: Docs + manual test in scratch monorepo passed; branch task/beans-jzty

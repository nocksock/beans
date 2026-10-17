# Code-quality review — 2026-10-17 (beans-9pxj)

Four parallel read-only reviews of `task/beans-9pxj` (based on dev @ 503e231; note dev has since gained `cmd/settings.go` and init flags):
tui · cmd+graph · beancore/bean/config/search/output · launcher+ui.

## Bugs (each → new bug bean unless noted)

| # | Bug | Where | Effort |
|---|-----|-------|--------|
| B1 | Mutations edit the live cached `*Bean` outside the lock → data race with TUI/watcher; memory≠disk on failed save. Fix: `Core.Update(id, func(*Bean) error)` or clone-on-get | `schema.resolvers.go` mutations, `core.go:368`, `links.go:386-499`, `cmd/list.go:164` blanks cached `Body` | M |
| B2 | Reload clears map then aborts on first unparsable file → beans vanish from TUI while an editor saves; watcher errors swallowed; `watchLoop` reads `c.done` unlocked → goroutine leak | `core.go:130-148`, `watcher.go:75,120,136` | S |
| B3 | Cycle path slice aliasing corrupts reported cycles (`beans check`) — reproduced | `links.go:312,338` | S |
| B4 | New ID never checked for collision → silent overwrite | `core.go:332-341` | S |
| B5 | Link mutations store the typed partial ID, not the resolved one → later "broken link" | `schema.resolvers.go:110-125,215,240,281` | S |
| B6 | Tags never validated (`applyTags` dead in prod); `mergeTags` reorders via map → diff churn | `cmd/create.go:82`, `cmd/update.go:179`, `cmd/content.go:46,69` | S |
| B7 | `init` re-run silently overwrites `.beans.yml` (also on dev) | `cmd/init.go:68` | S → fold into **beans-9p2n** |
| B8 | default_type effectively `milestone`; "task"/"todo"/IDLength defaults duplicated 3× | `config.go:305`, `core.go:191`, resolvers:94 | S → **beans-at6c** |
| B9 | TUI panic: config-write error from list hardcodes `previousState = viewDetail` with nil `detail.bean` | `tui.go:793` | S |
| B10 | Foreground `multiple:true` launcher runs only `beanIDs[0]`, silently | `tui.go:762-765` | S → fold into **beans-yxao** |
| B11 | Multi-select CreateChild picks random parent (map iteration) | `list.go:657` | S |
| B12 | TUI bulk status/type/priority/tags/parent/blocking swallow errors | `tui.go:296-483` | S → fold into **beans-p0mg** |
| B13 | Shebang multi-line launchers run under `sh -c` (README Ruby example broken) | `exec.go:11` | M |
| B14 | Byte-based truncation splits UTF-8 (list titles, tags padding, picker title, slug); overlay counts runes not cells | `ui/styles.go:327,542,575`, `tui/modal.go:45,125`, `bean/id.go:83` | S |
| B15 | README launcher docs wrong: `BEANS_TASK` is title not path; progress/confirm features nonexistent; `beans launch` documented but absent; `DisableLauncherWarning` dead | README 326-430, `launch.go:34`, `config.go:109` | S → reshape **beans-esxc** |
| B16 | `appendLaunchersToConfig` builds YAML via string formatting | `tui/launcher.go:96-101` | S |
| B17 | Foreground launch can hang on detached grandchild holding stderr — set `cmd.WaitDelay` | `exec.go:29` | S |
| B18 | Error modal says "Launcher Failed" for archive/edit/create errors | `tui.go:534,573,622` | fixed by R2 |
| B19 | Detail history not cleared on some paths → Back pops stale bean; detail rebuild loses scroll | `tui.go:533`, `tagpicker.go:125` | fixed by R5 |
| B20 | Small: Launch with no selection uses `Items()[0]` even if filtered; ChangeParent loses type of hidden selected beans; roadmap ignores `ExpandStatuses`; `errors.Is`; `IsValidColor` accepts `#zzz`; legacy single-dash ID parse | various | S (bundle) |

## Refactors

| # | Refactor | Effort | Affects beans |
|---|----------|--------|---------------|
| R1 | Delete dead `launcher/manager.go` + `launcher/launcher.go` (after B13), dead ui style helpers, dead TUI msgs/fields, other dead code; gofmt | S | obsoletes fx99, 359x, psqq, mptu |
| R2 | One `messageModal{title,msg,color}` replacing 4 copies (noLaunchers, noLaunchersMulti, launcherError, configWriteError); theme colors instead of `#ff0000`/`#555` | S | merges 5c6x + mszj + x3um remainder |
| R3 | Generic option picker for status/type/priority; shared `newPickerList`, `renderCursor`, `sortByTypeThenTitle` | M | prerequisite for jz5j |
| R4 | TUI `applyToBeans(ids, fn)` + `returnFromModal`; list/detail `targets()` + `openPickerFor` (~-120 lines in App.Update) | S/M | home for B12 |
| R5 | Modal interface (+optional stack) instead of 18-state enum & `previousState` | L | reshapes 603f, absorbs znuf; fixes B9/B19 structurally |
| R6 | Link abstraction in `bean` (`LinkType`, `Links()`, `RemoveLinks(pred)`) — halves `links.go`; move `ValidParentTypes` to config types; `Delete` exact-ID only | M | — |
| R7 | Single front-matter struct (currently 2 copies, 5 edit points per field) | S | do before nhal, digu |
| R8 | `filters.go` predicate list (~299→120 lines), deterministic relationship order, no hardcoded "normal" | S/M | — |
| R9 | Validation (status/type/priority/blocking cycles) in resolver/beancore, not CLI; single `updateBean` incl. link changes (atomic) | M | before digu |
| R10 | CLI helpers: `findBean`, `newResolver`, config `namesOf()` for flag help; config generic name/find helpers | S | — |
| R11 | `modalSize(w,h,pct,maxW,maxH)` helper; remaining inline copies | S | reshapes a2z4 |
| R12 | `check`/`delete` through GraphQL (or document exception) | M | — |

## Existing bean verdicts

| Bean | Verdict |
|------|---------|
| beans-z7jr README conflict markers | done (d3d4427) → complete |
| beans-w0nk fold/collapse | done → complete |
| beans-6aax `C` create child | done → complete (B11 new bug) |
| beans-359x LaunchManager query API | done but dead → scrap |
| beans-psqq unify launch paths | obsolete → scrap |
| beans-fx99 launcher manager tests | obsolete → scrap (R1) |
| beans-mptu launcher comments | obsolete → scrap |
| beans-a2z4 modal width helper | partly done → reshape to R11 |
| beans-x3um help text helper | mostly done → fold into R2 |
| beans-5c6x / mszj / znuf | merge → R2 (5c6x, mszj), znuf → R5 |
| beans-603f modal stack research | reshape → R5 |
| beans-wzbt partial ID match | prefix match done; reshape to "match without project prefix" (`show 9pxj`) |
| beans-qej2 graph CLI | reshape: exists; remaining = use graphlayout after x17f |
| beans-yxao / iweo | still valid; merge iweo into yxao + B10 |
| beans-esxc beans launch | reshape: implement via GraphQL or drop README section (B15) |
| beans-at6c, j5a5, p0mg, x0h8, 9p4x, p1px, jz5j, 9p2n, digu, nhal, w0ua, h0g0, bgih, cb5v, j7qs, j03e | still valid (some extended, see above) |

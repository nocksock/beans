---
# beans-rnkl
title: launchers in subdirectories cwd
status: archived
type: bug
priority: normal
created_at: 2026-10-18T11:10:11Z
updated_at: 2026-10-19T20:46:15Z
---

in ~/projects/studio/main when I use a launcher from a subdir, it has the wrong cwd and starts in ~
## Root cause

beans behaves as documented. Started from a subproject (e.g. `~/projects/studio/main/packages/paint`, which has its own `.beans.yml`), launchers are inherited from the hub `~/projects/studio/.beans.yml` (`cascade: true`):

- cwd and `BEANS_CONFIG_DIR` = `~/projects/studio` (declaring config)
- `BEANS_ROOT` = `~/projects/studio/main/packages/paint` (nearest config)

The studio `tmux+claude` launcher runs `tmux new-window -c "${BEANS_ROOT:-.}/main"`, i.e. `.../packages/paint/main`, which doesn't exist, so tmux silently falls back to `$HOME` (verified). `worktree-tmux+claude` breaks the same way (`cd $BEANS_ROOT` then `./scripts/bean-worktree` not found).

The README's worktree launcher examples use `$BEANS_ROOT/$WORKTREE_DIR` and have the same problem when inherited.

## Plan

- [x] README: change the worktree examples (worktree-tmux, worktree-kitty, the multi-bean example) to be inheritance-safe (`$BEANS_CONFIG_DIR`, the launcher's cwd)
- [x] README: add a short note on when to use `BEANS_ROOT` (the bean's project) vs `BEANS_CONFIG_DIR` (where the launcher and its scripts live)
- [x] Studio hub config: replace `${BEANS_ROOT:-.}` with `$BEANS_CONFIG_DIR` (or rely on the cwd) in `tmux+claude`, `worktree-tmux+claude`, `worktree-kitty`
- [x] Verify by launching from a studio subproject
- [x] Follow-up bean in the studio hub: the cleanup note commits beans with `git -C "$HUB" add .beans`, but subproject beans live in `main/packages/*/.beans`

## Work Log

2026-10-18 13:20: Root cause found (studio launchers use BEANS_ROOT; tmux falls back to ~). Plan agreed: README + studio config + studio follow-up bean.
2026-10-18 13:35: README examples + BEANS_ROOT/BEANS_CONFIG_DIR note; studio launchers switched to BEANS_CONFIG_DIR (uncommitted); verified old→~, new→main/; follow-up studio-rwuw

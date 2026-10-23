---
# beans-zkj9
title: Adopt upstream features
status: todo
type: epic
created_at: 2026-10-23T12:00:40Z
updated_at: 2026-10-23T12:00:40Z
---

Bring over the upstream (hmans/beans) features that are worth having in the fork. They come from the 'Upstream features the fork lacks' list in docs/compare.md (beans-wcdm).

Approach: reimplement each feature in the fork's code and use upstream's code only as a reference. Don't cherry-pick: the fork and upstream have diverged by about 440 commits.

Suggested order: copy ID and CLI flags (quick wins), then the pkg/ split, then archive and plugins. The web UI (beans-lbjp) comes later.
---
# beans-bufq
title: Single front-matter struct for parse and render
status: todo
type: task
tags:
    - refactoring
    - code-review
created_at: 2026-10-17T14:02:23Z
updated_at: 2026-10-17T14:02:23Z
parent: beans-c8p4
---

Review beans-9pxj (`docs/reviews/2026-10-17-code-quality-review.md` in hub), R7.

`frontMatter` (bean.go:209) and `renderFrontMatter` (:246) duplicate fields; each new field touches 5 places. Merge into one struct / one field list. Do before beans-nhal and beans-digu.
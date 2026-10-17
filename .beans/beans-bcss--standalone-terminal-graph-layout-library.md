---
# beans-bcss
title: Standalone terminal graph layout library
status: todo
type: task
priority: deferred
tags:
    - idea
created_at: 2026-10-17T13:54:18Z
updated_at: 2026-10-17T13:54:18Z
---

Hand-rolling a layered (Sugiyama-style) DAG layout with box-drawing output turned out to be a project of its own. Idea: a standalone Go library + small CLI (DOT/Mermaid in, terminal out) with node positions exposed for interactive navigation. beans could later use it for a 2D dependency graph view.

WIP code: branch `spike/graphlayout` (commit ce1d8d5): cycle breaking, longest-path layering with sources pulled right, placeholder nodes for long edges, barycenter ordering, per-source/per-target channels, junction glyphs, horizontal-over-vertical crossings. Known gap: horizontal runs of unrelated edges overlap on shared rows (e.g. a->z, b->x, b->y, c->x, c->z).

Spike vs mermaid-ascii: see beans-x17f.
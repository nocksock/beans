---
# beans-z7jr
title: README launcher section contains committed merge-conflict markers
status: todo
type: bug
created_at: 2026-10-17T13:22:57Z
updated_at: 2026-10-17T13:22:57Z
---

README.md (Launchers section, around 'Configure launchers in .beans.yml') contains literal <<<<<<< HEAD / ======= / >>>>>>> main markers from an old merge, plus a duplicated 'First-time setup' paragraph. The 'main' side documents the obsolete 'command:' key; keep the HEAD side (exec:) and drop the rest. Found while working on beans-cm7n.
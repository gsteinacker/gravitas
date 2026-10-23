---
id: T-008
title: Barnes-Hut-Octree
status: todo
milestone: M2
priority: 1
estimate: L
depends: [T-003]
---

## Ziel

O(N log N) statt O(N²).

## Akzeptanzkriterien

- [ ] θ→0 entspricht direkter Summe (rel. Fehler < 1e-10)
- [ ] Node-Pool, keine Allokationen pro Schritt
- [ ] BHs werden direkt summiert

## Notizen

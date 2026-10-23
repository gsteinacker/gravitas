---
id: T-008
title: Barnes-Hut-Octree
status: done
milestone: M2
priority: 1
estimate: L
depends: [T-003]
---

## Ziel

O(N log N) statt O(N²).

## Akzeptanzkriterien

- [x] θ→0 entspricht direkter Summe (rel. Fehler < 1e-10)
- [x] Node-Pool, keine Allokationen pro Schritt
- [x] BHs werden direkt summiert

## Notizen
- Umsetzung: Morton-Keys (21 Bit/Achse) + Radix-Sort, Top-down-Build aus sortierten Bereichen, Skip-Pointer-Traversierung ohne Stack, Öffnungskriterium mit Barnes-Offset-Korrektur (δ). Nur Monopol.
- Genauigkeit (Median rel. Fehler, Scheibe + 2 BHs, N=5000): θ 0,3 → 0,06 %, 0,6 → 0,25 %, 0,9 → 0,5 %.
- Energie-Drift ist vom Integrator dominiert (direkt 1,5e-3, BH 1,0e-3 nach 500 Schritten): enge BH-Vorbeiflüge bei fester Schrittweite. Folgearbeit für hierarchische Zeitschritte siehe T-025.
- Mögliche Verbesserung: Quadrupol-Momente erlauben größeres θ bei gleicher Genauigkeit.

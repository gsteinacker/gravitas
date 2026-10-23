---
id: T-009
title: Parallele Kraftberechnung
status: done
milestone: M2
priority: 1
estimate: M
depends: [T-008]
---

## Ziel

Alle Kerne nutzen.

## Akzeptanzkriterien

- [x] Speedup ≥ 7× auf 12 Kernen bei N=100k
- [x] Race-Detector grün

## Notizen
- N=100k: seriell 137,6 ms, 12 Kerne 18,9 ms → 7,3×. Traversierung allein 8,8×, der serielle Build (3,6 ms) begrenzt nach Amdahl.
- `sim.ParallelFor` mit dynamischer Chunk-Vergabe (atomarer Zähler), damit E-Kerne nicht bremsen. Auch der direkte Löser nutzt es.

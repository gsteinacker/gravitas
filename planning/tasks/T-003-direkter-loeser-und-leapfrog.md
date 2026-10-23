---
id: T-003
title: Direkter Löser, Leapfrog-Integrator, Energie-Diagnose
status: done
milestone: M0
priority: 1
estimate: M
depends: [T-002]
---

## Ziel

Physikalische Referenz, gegen die alle schnelleren Löser getestet werden.

## Akzeptanzkriterien

- [x] Kepler-Test: Energie-Drift < 1e-4 über 1000 Umläufe
- [x] Diagnose: E_kin, E_pot, Impuls
- [x] Benchmark für N=1k, 10k

## Notizen
- Benchmark (M4 Pro): direkte Summe ~6 Mrd. Paare/s, N=10k in 16 ms. Siehe docs/perf-log.md.
- ~97 Allokationen pro Aufruf stammen aus den Worker-Goroutinen und sind unabhängig von N. Bei Barnes-Hut (T-009) einen persistenten Worker-Pool prüfen.

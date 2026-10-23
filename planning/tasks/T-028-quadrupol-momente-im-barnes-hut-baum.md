---
id: T-028
title: Quadrupol-Momente im Barnes-Hut-Baum
status: todo
milestone: M2
priority: 2
estimate: M
depends: [T-027]
---

## Ziel

Die Laufzeit des Baums ist proportional zu den Wechselwirkungen pro Stern (T-027: ~530 bei θ = 0,6). Quadrupol-Momente machen jeden Knoten genauer. So erreicht ein größeres θ dieselbe Genauigkeit mit deutlich weniger Wechselwirkungen (in der Literatur typisch θ 0,6 → 0,8–0,9 bei gleichem Fehler).

## Akzeptanzkriterien

- [ ] Quadrupol-Tensor pro Knoten beim Build (bottom-up, parallelverschiebungsfähig)
- [ ] Kraft und Potential mit Quadrupol-Term, Gradiententest wie bei `PairForce`
- [ ] θ so wählen, dass der Median-Fehler ≤ dem heutigen bei θ = 0,6 bleibt
- [ ] `BenchmarkAccelerationsEvolved` mindestens 25 % schneller (ohne laufende App gemessen)

## Notizen

---
id: T-028
title: Quadrupol-Momente im Barnes-Hut-Baum
status: done
milestone: M2
priority: 2
estimate: M
depends: [T-027]
---

## Ziel

Die Laufzeit des Baums ist proportional zu den Wechselwirkungen pro Stern (T-027: ~530 bei θ = 0,6). Quadrupol-Momente machen jeden Knoten genauer. So erreicht ein größeres θ dieselbe Genauigkeit mit deutlich weniger Wechselwirkungen (in der Literatur typisch θ 0,6 → 0,8–0,9 bei gleichem Fehler).

## Akzeptanzkriterien

- [x] Quadrupol-Tensor pro Knoten beim Build (bottom-up, Steiner-Verschiebung der Kinder)
- [x] Kraft und Potential mit Quadrupol-Term, Gradiententest (`TestQuadrupoleForceIsGradientOfPotential`)
- [x] θ so wählen, dass der Median-Fehler ≤ dem heutigen bei θ = 0,6 bleibt: θ = 0,9 mit 0,046 % statt 0,23 %
- [x] `BenchmarkAccelerationsEvolved` mindestens 25 % schneller (ohne laufende App gemessen): 29,0 → 20,7 ms (−29 %)

## Notizen
- Fernfeld-Test: Ein Haufen als einzelner Knoten hat mit Monopol 0,24 % Fehler, mit Quadrupol 0,0035 %.
- θ-Scan mit allen Werten in docs/perf-log.md. θ = 1,0 wäre 13 % schneller, hat aber 4× größere Ausreißer.
- Nebeneffekt: Das Potential der Energie-Diagnose ist jetzt bei θ = 0,6 auf 1,5·10⁻⁶ genau (vorher 4·10⁻⁴ bei θ = 0,3). Die Baum-Schätzung im HUD ist damit verlässlich, die alte Einschränkung „±30 %“ aus T-007 gilt nicht mehr.
- Schalter `MonopoleOnly` für Vergleiche.

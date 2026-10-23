---
id: T-027
title: Gruppen-Walk: Interaktionslisten pro Blatt
status: todo
milestone: M2
priority: 2
estimate: M
depends: [T-024]
---

## Ziel

`walk` ist 78 % der Laufzeit, jeder Stern läuft einzeln durch den Baum. Beim Gruppen-Walk (Barnes 1990) wird pro Blatt bzw. kleiner Gruppe einmal eine Interaktionsliste aufgebaut (Knoten, die für die ganze Gruppe weit genug weg sind, plus Nachbarsterne), die dann für alle Sterne der Gruppe ausgewertet wird. Das spart Baumdurchläufe und erlaubt einen cache-freundlichen, vektorisierbaren inneren Loop.

## Akzeptanzkriterien

- [ ] Öffnungskriterium konservativ für die ganze Gruppe (Bounding-Box der Gruppe)
- [ ] Genauigkeit nicht schlechter als der heutige Walk bei gleichem θ (Median-Fehler-Test)
- [ ] `BenchmarkAccelerationsEvolved` ≤ 20 ms (heute 35 ms), also ≥ 30 Schritte/s auch bei dichtem Kern
- [ ] Allokationen pro Aufruf weiterhin unabhängig von N

## Notizen

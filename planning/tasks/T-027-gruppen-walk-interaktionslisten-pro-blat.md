---
id: T-027
title: Gruppen-Walk: Interaktionslisten pro Blatt
status: done
milestone: M2
priority: 2
estimate: M
depends: [T-024]
---

## Ziel

`walk` ist 78 % der Laufzeit, jeder Stern läuft einzeln durch den Baum. Beim Gruppen-Walk (Barnes 1990) wird pro Blatt bzw. kleiner Gruppe einmal eine Interaktionsliste aufgebaut (Knoten, die für die ganze Gruppe weit genug weg sind, plus Nachbarsterne), die dann für alle Sterne der Gruppe ausgewertet wird. Das spart Baumdurchläufe und erlaubt einen cache-freundlichen, vektorisierbaren inneren Loop.

## Akzeptanzkriterien

- [x] Öffnungskriterium konservativ für die ganze Gruppe (Bounding-Box der Gruppe)
- [x] Genauigkeit nicht schlechter als der heutige Walk bei gleichem θ (Median 0,23 % → 0,16 %)
- [ ] ~~`BenchmarkAccelerationsEvolved` ≤ 20 ms~~ **verfehlt: 41 ms statt 29 ms (+41 %)**
- [x] Allokationen pro Aufruf weiterhin unabhängig von N

## Notizen
- **Ergebnis: nicht übernommen (Code verworfen).** Ohne Last gemessen, 100k: t = 0 von 17,3 auf 25,0 ms, entwickelt von 29,0 auf 41,0 ms.
- Ursache gemessen: Wechselwirkungen pro Stern 530 (pro Stern) gegenüber 634 / 693 / 757 bei Gruppen à 4 / 8 / 24. Die Laufzeit ist proportional zur Zahl der Wechselwirkungen. Das konservative Gruppenkriterium kostet mehr Paare, als der gesparte Baumdurchlauf in skalarem Go einbringt.
- Ein Gruppen-Walk lohnt sich erst, wenn die Listen vektorisiert ausgewertet werden (SIMD oder GPU, vgl. T-023).
- Der eigentliche Hebel ist, die Wechselwirkungen bei gleicher Genauigkeit zu reduzieren → T-028 (Quadrupol-Momente).
- Lehre für Messungen: Eine parallel laufende App (des Nutzers) verfälschte den ersten Vergleich bis zur Umkehr des Vorzeichens. Siehe CLAUDE.md.

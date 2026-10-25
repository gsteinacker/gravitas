---
id: T-072
title: Kerr Stufe 2: Lense-Thirring-Kraft (Frame-Dragging)
status: todo
milestone: M9
priority: 3
estimate: L
depends: [T-071]
---

## Ziel

Rotierende Löcher ziehen den Raum mit (Modell C aus `docs/analysis/03-kerr.md`): eine Lense-Thirring-Beschleunigung
in Ordnung 1/c² neben Paczyński-Wiita. Bahnen gegen die Drehung werden instabiler als solche mit ihr, und
Bahnebenen präzedieren.

## Akzeptanzkriterien

- [ ] ADR „Gravitomagnetische Kraft rotierender Löcher“ (Entwurf in der Analyse)
- [ ] a_LT = 2/(c² r³) [3(J·r̂)(r̂×v) + v×J], integriert mit Boris-Rotationsschritt; Spinpräzession des Lochs aus
      dem Gegendrehmoment
- [ ] Tests: Knotenpräzession eines Testteilchens gegen Ω_LT = 2J/(c²r³); Energie bei reiner LT-Kraft erhalten;
      Gesamtdrehimpuls inkl. Spin erhalten
- [ ] GPU-Pfad und Substeps berücksichtigt (`TestGPUWalkMatchesCPU`, falls betroffen); Benchmark vorher/nachher
- [ ] Lernmodus: Frame-Dragging im Kerr-Thema, Modellgrenze „nur erste Ordnung“

## Notizen

Aus der Analyse T-059.

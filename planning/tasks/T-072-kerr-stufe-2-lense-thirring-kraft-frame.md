---
id: T-072
title: Kerr Stufe 2: Lense-Thirring-Kraft (Frame-Dragging)
status: done
milestone: M9
priority: 3
estimate: L
depends: [T-059]
---

## Ziel

Rotierende Löcher ziehen den Raum mit (Modell C aus `docs/analysis/03-kerr.md`): eine Lense-Thirring-Beschleunigung
in Ordnung 1/c² neben Paczyński-Wiita. Bahnen gegen die Drehung werden instabiler als solche mit ihr, und
Bahnebenen präzedieren.

## Akzeptanzkriterien

- [x] ADR „Gravitomagnetische Kraft rotierender Löcher“ (Entwurf in der Analyse)
- [x] a_LT = 2/(c² r³) [3(J·r̂)(r̂×v) + v×J], integriert mit Boris-Rotationsschritt; Spinpräzession des Lochs aus
      dem Gegendrehmoment
- [x] Tests: Knotenpräzession eines Testteilchens gegen Ω_LT = 2J/(c²r³); Energie bei reiner LT-Kraft erhalten;
      Gesamtdrehimpuls inkl. Spin erhalten
- [x] GPU-Pfad und Substeps berücksichtigt (`TestGPUWalkMatchesCPU`, falls betroffen); Benchmark vorher/nachher
- [x] Lernmodus: Frame-Dragging im Kerr-Thema, Modellgrenze „nur erste Ordnung“

## Notizen

Aus der Analyse T-059.

Erledigt 2026-10-26: ADR-0013, `internal/sim/framedrag.go`. Boris-Rotation in den RESPA-Kicks, Rückwirkung auf
Impuls und Spin des Lochs. Tests: Knotenpräzession 0,527 rad statt 0,540 rad (Lense-Thirring, −2,5 %), ohne Spin
keine Präzession; Impuls und Gesamtdrehimpuls erhalten; |u| bleibt bei der Rotation gleich. Kosten +0,7 %
(perf-log). Kein GPU-Pfad betroffen: Lochkräfte laufen auf der CPU.


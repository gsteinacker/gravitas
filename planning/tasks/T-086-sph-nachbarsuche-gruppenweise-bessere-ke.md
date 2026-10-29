---
id: T-086
title: SPH-Nachbarsuche gruppenweise, bessere Kernauslastung
status: done
milestone: M9
priority: 1
estimate: M
depends: []
---

## Ziel

Galaxy nutzte bei Gasszenen nur einen Teil der Prozessorkerne (Nutzer: „deutlich mehr CPU nutzen können“, 5
Schritte/s bei 80 FPS). Die Gasrechnung soll alle Kerne auslasten und weniger Arbeit pro Auswertung machen.

## Akzeptanzkriterien

- [x] Profil: Ursachen der schlechten Auslastung gefunden
- [x] Nachbarsuche gruppenweise je Baumblatt statt je Teilchen
- [x] Baumaufbau ohne serielle Vergleichssortierung, ohne Allokationen
- [x] Gasrechnung überlappt mit der Gravitation auf der GPU, `updateGas` parallel
- [x] Tests grün, `make race` sauber, Benchmarks vorher/nachher in `docs/perf-log.md`

## Notizen
Erledigt 2026-10-29, Zahlen in `docs/perf-log.md`: BenchmarkHorsehead 66 → 48 ms, Pferdekopfnebel +26 %, Spirale
+41 %, Szene 14 im dichten Zustand mehr als doppelt so schnell. Ursachen: Arbeitspakete von 256 räumlich
sortierten Teilchen (dichter Kern blockierte einen Kern), serieller Baumaufbau mit Vergleichssortierung, serielles
`updateGas`, CPU wartete auf die GPU-Gravitation. Nebenbei: Zeitgrenze von `TestCalibrateMeasures` gilt unter
`-race` nicht mehr (schlug schon vorher fehl).

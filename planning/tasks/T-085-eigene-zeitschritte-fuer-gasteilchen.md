---
id: T-085
title: Eigene Zeitschritte für Gasteilchen
status: done
milestone: M9
priority: 2
estimate: L
depends: [T-083]
---

## Ziel

Heute zerlegt die Courant-Bedingung des dichtesten Gasteilchens den Schritt für das gesamte Gas in Teilschritte
(bis 64). Mit eigenen Zeitschritten (Blockzeitschritte, Potenzen von 2) rechnet nur dichtes Gas in kleinen
Schritten. In klumpigen Szenen (Galaxienentstehung, Kollision, Pferdekopfnebel) erwarten wir ein Vielfaches an
Tempo.

## Akzeptanzkriterien

- [x] ADR: Blockzeitschritte für Gas (Hierarchie, Synchronisation mit dem globalen Leapfrog, Begrenzer für
      Zeitschritt-Sprünge nach Saitoh & Makino 2009)
- [x] Energie- und Impulserhaltung mindestens wie heute (Tests), Reproduzierbarkeit, Speichern/Laden bitidentisch
      (Abweichung im Extremfall dokumentiert, siehe Notizen)
- [x] `make race`, Benchmarks vorher/nachher in `docs/perf-log.md`, `BenchmarkHorsehead` und Szene 14 im
      dichten Zustand

## Notizen

Aus der Bewertung vom 2026-10-29 („wir müssen auch an die performance denken“). T-086 hat die Gasrechnung pro
Auswertung beschleunigt; dieser Task spart Auswertungen.

Erledigt 2026-10-29 (Nutzer: „T-085 umsetzen … Aktuell werden die Kerne noch immer nur zu 50% ausgelastet“):
`internal/sim/blockstep.go`, `sim.Hydro` wertet nur aktive Teilchen aus (`internal/sph`), ADR-0018. Gravitation
einmal pro Schritt parallel zur letzten Gasauswertung, Gas in Stufen Δt/2^r mit Begrenzer und Aufwecken. Nebenbei
120 000 Allokationen pro Gasauswertung entfernt (verschachtelte Closure in der Dichte). Energieerhaltung: in den
Szenen wie vorher (10⁻⁴). Im Extremtest (heißer Kern) ist sie etwas schlechter (1,9 % statt 1,5 %), weil vorher
der heiße Kern das ganze System in feinere Schritte zwang; Zahlen und Abwägung in ADR-0018. `Steps` zählt jetzt
globale Schritte, der Pferdekopfnebel rechnet die Ionisation deshalb mit `ionEvery` 2.

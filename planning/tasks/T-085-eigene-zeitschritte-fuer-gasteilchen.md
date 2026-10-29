---
id: T-085
title: Eigene Zeitschritte für Gasteilchen
status: todo
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

- [ ] ADR: Blockzeitschritte für Gas (Hierarchie, Synchronisation mit dem globalen Leapfrog, Begrenzer für
      Zeitschritt-Sprünge nach Saitoh & Makino 2009)
- [ ] Energie- und Impulserhaltung mindestens wie heute (Tests), Reproduzierbarkeit, Speichern/Laden bitidentisch
- [ ] `make race`, Benchmarks vorher/nachher in `docs/perf-log.md`, `BenchmarkHorsehead` und Szene 14 im
      dichten Zustand

## Notizen

Aus der Bewertung vom 2026-10-29 („wir müssen auch an die performance denken“). T-086 hat die Gasrechnung pro
Auswertung beschleunigt; dieser Task spart Auswertungen.

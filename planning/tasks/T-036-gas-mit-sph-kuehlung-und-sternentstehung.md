---
id: T-036
title: Gas mit SPH, Kuehlung und Sternentstehung
status: done
milestone: M6
priority: 1
estimate: L
depends: [T-034]
---

## Ziel

Gas für realistische Spiralen, Starbursts, Staub und Nebel (ADR-0007).

## Akzeptanzkriterien

- [x] SPH (Kernel, adaptive h, symmetrische Kräfte, Viskosität, Energiegleichung), Nachbarbaum
- [x] Tests: Kernel-Normierung, Nachbarn gegen Brute Force, Gitterdichte, exakte Impuls- und Energieerhaltung der Hydro-Terme, adiabatische Gaskugel (1 %), Kühlung und reproduzierbare Sternentstehung
- [x] Szenen mit Gas (1, 2, 6, 7), junge Sterne blau, Gas sichtbar (`o`), HUD, Spielstände Format 2

## Notizen
- Spirale (150k): 63 neue Sterne nach 57 Myr, 90 → ~35 Schritte/s. Optimierung möglich (Dichte-Iterationen, Nachbarlisten wiederverwenden).

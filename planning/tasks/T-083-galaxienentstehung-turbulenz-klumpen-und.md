---
id: T-083
title: Galaxienentstehung: Turbulenz, Klumpen und Sternbildung in unaufgelöstem Gas
status: done
milestone: M9
priority: 2
estimate: M
depends: [T-082]
---

## Ziel

Die Gaswolke der Szene „Galaxy formation“ ist zu glatt und kugelförmig. Sie soll mit Turbulenz und
Dichteschwankungen beginnen. Gas, das dichter wird, als die Simulation auflöst, soll zu Sternen werden, statt
den Zeitschritt des ganzen Gases zu diktieren. Die Szene soll unterm Strich nicht langsamer werden.

## Akzeptanzkriterien

- [x] Turbulenz (30 km/s) und 30 % Dichteschwankungen in Szene 14, Texte und Quellen
- [x] Sternentstehung in unaufgelöstem Gas (ADR-0017), Test `TestUnresolvedGasFormsStarsAtOnce`
- [x] Messung vorher/nachher: Szene 14 im Mittel 41 → 76 Schritte/s, Scheibe entsteht weiter (0,37), Spirale
      behält ihre Sternentstehung (70 → 74 neue Sterne in 400 Myr)
- [x] Folgetasks T-084 (Klumpen, länglicher Halo) und T-085 (eigene Gas-Zeitschritte)

## Notizen

Nutzerfrage vom 2026-10-29: Gaswolken zu homogen? Bewertung nach Rechenaufwand: (1) Turbulenz gering, (2)
Klumpen mittel, (3) kosmologische Anfangsbedingungen hoch.

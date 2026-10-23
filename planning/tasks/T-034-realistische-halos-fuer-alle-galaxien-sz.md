---
id: T-034
title: Realistische Halos fuer alle Galaxien-Szenen
status: done
milestone: M6
priority: 1
estimate: S
depends: [T-033]
---

## Ziel

Alle Galaxien-Szenen sollen realistische Halos aus Dunkler Materie haben (Nutzerwunsch). Bisher stabilisierte in Szene 1/2 ein übergroßes Loch die Scheiben, 6/7 hatten Plummer-Halos mit zu wenig DM.

## Akzeptanzkriterien

- [x] Szenen 1, 2, 6, 7 mit Hernquist-Halos (85–90 % DM), realistischen Lochmassen, Toomre-Scheiben
- [x] Szene 1 zeigt Spiralarme und Balken, Szene 2 einen sichtbaren Anflug (Start bei ~47 kpc statt 21 kpc)
- [x] BH-Tests von Szenendateien entkoppelt (eigene feste Testgalaxie)

## Notizen
- θ-Scan auf der realistischen Galaxie (150k): θ = 0,5: 0,08 % Median, 193 ms; 0,7: 0,25 %, 90 ms; 0,9: 0,57 %, 33 ms. In DM-dominierten Systemen sind die relativen Fehler größer, weil sich große Halo-Beiträge im Inneren teilweise aufheben. θ = 0,9 bleibt Standard (Interaktivität), dokumentiert in docs/perf-log.md.

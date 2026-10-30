---
id: T-088
title: Milchstraße mit Satellitengalaxien und Gezeitenströmen
status: done
milestone: M9
priority: 2
estimate: M
depends: []
---

## Ziel

Eine Szene zeigt die Milchstraße mit ihren größten Begleitern: Große und Kleine Magellansche Wolke und die
Sagittarius-Zwerggalaxie. Die Satelliten bekommen eine feinere Massenauflösung als die Hauptgalaxie, damit sie nicht
durch Zweikörper-Relaxation künstlich zerfallen. Sichtbar werden sollen Gezeitenströme (Sagittarius-Strom),
dynamische Reibung und die Störung der Scheibe.

## Akzeptanzkriterien

- [x] Szene `15-milky-way-satellites.json`: Milchstraße wie Szene 1 (Drehsinn wie die echte), LMC und SMC als
      gasreiche Scheiben, Sagittarius als sphärische Zwerggalaxie; Positionen und Bahnebenen aus der Literatur,
      Geschwindigkeiten an das leichtere Modell angepasst (Quellen und Annahmen in `note`)
- [x] ADR zu gemischten Teilchenmassen (Satelliten feiner aufgelöst als der Wirt)
- [x] Test: ein isolierter Zwerg bleibt über einige Umläufe gebunden und behält seinen Radius (kein künstlicher
      Zerfall); Test: Sagittarius verliert auf seiner Bahn Sterne an einen Strom
- [x] Messung mit `tools/evolve`: gebundener Anteil von Sagittarius über die Zeit, Schritte pro Sekunde
- [x] Lernthema „satellites“ (Zwerggalaxien, Gezeitenströme, Kugelsternhaufen als Ausblick) in drei Stufen
- [x] Screenshot-Prüfung

## Notizen

Kugelsternhaufen sind bei ~10⁶ M☉ pro Teilchen und 150 pc Glättung nicht als Haufen auflösbar (siehe Bewertung
vom 2026-10-30); höchstens später als Punktmassen mit eigener Darstellung.

## Ergebnis

Messung mit 20 000 Körpern (Analysewerkzeug, gebundene Sterne aus der Eigenenergie):

| Zeit | Sagittarius gebunden | LMC Abstand | SMC gebunden |
|---|---|---|---|
| 0 | 100 % | 20 | 100 % |
| 660 Myr (1. Perizentrum) | 81 % | 21 | 86 % |
| 1,2 Gyr | 71 % | 18 | 66 % |
| 1,7 Gyr | 69 % | 11,5 | 43 % |

- Sagittarius verliert beim ersten Perizentrum ~30 % seiner Sterne an einen sichtbaren Strom.
- Die LMC (15 % der Masse) sinkt durch dynamische Reibung nach innen.
- Die SMC gibt nach und nach Sterne und Gas an die LMC ab, wie bei der Magellanschen Brücke.
- Isoliert sind alle drei stabil: Halbmassenradius Sgr 0,49 → 0,51, LMC 0,97 → 0,97, SMC 0,40 → 0,45 in
  0,8–1 Gyr.
- Bei 200 Teilchen (Teilchenmassen des Wirts) wächst der Radius von Sagittarius dagegen um ~40 %.
- `tools/evolve`: 161 Schritte/s bei 30 000 Körpern (App lief parallel). Szene 1 bei gleicher Zahl liegt ähnlich,
  die Satelliten kosten kaum Gas-Rechenzeit.

Erste Versuche: Mit der SMC in 11 kpc Abstand und doppelter Masse versank sie durch dynamische Reibung in 500 Myr
in der LMC. Mit ausgedehnterer Scheibe wurde sie durch Gezeiten in 1 Gyr zerrissen.

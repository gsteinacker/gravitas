---
id: T-047
title: Laufzeit-Regler für flüssigen Betrieb
status: done
milestone: M7
priority: 1
estimate: L
depends: [T-046]
---

## Ziel

Auch innerhalb einer Sitzung schwankt die Last (dichter Kern, Gas, Zoom, Wärme). Ein Regler hält die Darstellung
flüssig, indem er Renderauflösung, Effektqualität, θ und das Schritttempo anpasst, ohne die Physik zu verfälschen.

## Akzeptanzkriterien

- [x] Schrittzeit in `Runner`, GPU-Kernelzeit und Frame-Zeit gemessen
- [x] `perf.Governor` mit Hysterese und gleitendem Mittel, Tabellentests (langsam → runter, schnell → rauf, kein Flackern)
- [x] Stellgrößen: Renderfaktor (inkl. Retina über `DeviceScaleFactor`), Bloom-Qualität, θ ∈ [0,7; 1,0], Ziel-steps/s
- [x] Vom Nutzer gewählte Effekte bleiben an; nur deren Qualität sinkt
- [x] Thermalstatus/Stromsparmodus senken die Stufe
- [x] Mit `GOMAXPROCS=2 -gpu=false` bleibt die App ≥ 30 FPS; Messung vorher/nachher in `docs/perf-log.md`

## Notizen

- `perf.Governor`: Tick alle 0,5 s.
  - Renderfaktor in den Stufen 2 / 1,5 / 1 / 0,75 / 0,5.
  - Danach eine Schrittbremse auf das Profilziel.
  - θ steigt in 0,05-Schritten bis 1,0, wenn die Simulation unter 80 % des Ziels liegt.
  - Hysterese: 3 Ticks runter, 10 Ticks hoch, 4 Ticks Pause nach jeder Änderung.
- Der Renderer skaliert feste Pixelgrößen mit dem Faktor, Retina ist neu.
- `tuning.observe` vergleicht die gemessene mit der vorhergesagten Schrittzeit und korrigiert die Sternzahl
  beim nächsten Laden der Szene.
- Messwerte stehen in `docs/perf-log.md`. Das Bench-Ziel blieb unverändert: Die Hot Loops der Physik sind
  nicht berührt, im Renderer kamen nur konstante Faktoren hinzu.

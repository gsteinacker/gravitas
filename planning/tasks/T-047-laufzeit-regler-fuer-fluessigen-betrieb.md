---
id: T-047
title: Laufzeit-Regler für flüssigen Betrieb
status: todo
milestone: M7
priority: 1
estimate: L
depends: [T-046]
---

## Ziel

Auch innerhalb einer Sitzung schwankt die Last (dichter Kern, Gas, Zoom, Wärme). Ein Regler hält die Darstellung
flüssig, indem er Renderauflösung, Effektqualität, θ und das Schritttempo anpasst, ohne die Physik zu verfälschen.

## Akzeptanzkriterien

- [ ] Schrittzeit in `Runner`, GPU-Kernelzeit und Frame-Zeit gemessen
- [ ] `perf.Governor` mit Hysterese und gleitendem Mittel, Tabellentests (langsam → runter, schnell → rauf, kein Flackern)
- [ ] Stellgrößen: Renderfaktor (inkl. Retina über `DeviceScaleFactor`), Bloom-Qualität, θ ∈ [0,7; 1,0], Ziel-steps/s
- [ ] Vom Nutzer gewählte Effekte bleiben an; nur deren Qualität sinkt
- [ ] Thermalstatus/Stromsparmodus senken die Stufe
- [ ] Mit `GOMAXPROCS=2 -gpu=false` bleibt die App ≥ 30 FPS; Messung vorher/nachher in `docs/perf-log.md`

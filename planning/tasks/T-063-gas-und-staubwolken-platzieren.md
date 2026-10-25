---
id: T-063
title: Gas- und Staubwolken platzieren
status: todo
milestone: M10
priority: 2
estimate: M
depends: [T-058, T-064]
---

## Ziel

Neben Schwarzen Löchern und Sternhaufen soll man auch Gas- bzw. Staubwolken in die Szene setzen können: eine
Molekülwolke, die unter Eigengravitation kollabiert, Sterne bildet, von heißen Sternen ionisiert wird und Licht
dahinter abschwächt.

## Akzeptanzkriterien

- [ ] Werkzeug „Gaswolke“ (Menü, Inspektor, Taste) neben Loch und Sternhaufen
- [ ] Parameter: Masse, Radius, Temperatur, Turbulenz, ggf. Rotation; in physikalischen Einheiten der Szene, sonst
      Simulationseinheiten; sinnvolle Standardwerte je Szene (Galaxie vs. Pferdekopf-Maßstab)
- [ ] Aufbau wie `gasclump` der Szenen (`addGasCloud`), Teilchenzahl an Sternzahl/Profil angepasst, h konsistent
- [ ] Wolke wirkt mit SPH, Kühlung, Sternentstehung, Ionisation und Staub zusammen (Screenshot-Serie)
- [ ] Platzierung nutzt das 3D-Werkzeug (T-064)
- [ ] Lernmodus: Hinweis/Thema zur Wolke (Jeans-Instabilität)

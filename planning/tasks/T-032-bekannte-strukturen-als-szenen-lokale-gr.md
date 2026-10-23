---
id: T-032
title: Bekannte Strukturen als Szenen (Lokale Gruppe, Antennen, M51, S-Sterne)
status: done
milestone: M5
priority: 1
estimate: L
depends: [T-031]
---

## Ziel

Nutzerwunsch: Szenen wie die Lokale Gruppe oder andere bekannte Strukturen, ladbar und speicherbar.

## Akzeptanzkriterien

- [x] Szenenformat JSON (docs/scenes.md), eingebettete Presets, `-scene`
- [x] Bisherige Presets als JSON, parametergleich
- [x] Lokale Gruppe (echte Abstände, Geschwindigkeiten, c), Antennen (Toomre & Toomre), M51, S-Sterne (Gillessen et al. 2017)
- [x] Jede Szene mit physikalischer Zeiteinheit im HUD (yr/kyr/Myr/Gyr)

## Notizen
- Gefunden: Die Linsenstärke hing am gezeichneten (auf 3 px geklemmten) Lochradius, winzige Löcher bei echtem c erzeugten große Einstein-Ringe. Jetzt zählt der physikalische Radius.
- Annahmen und Quellen stehen im `note`-Feld jeder Komponente. Werte aus der Literatur sind gerundet und teils aus dem Gedächtnis (z. B. S-Stern-Elemente). Vor wissenschaftlicher Verwendung bitte prüfen.

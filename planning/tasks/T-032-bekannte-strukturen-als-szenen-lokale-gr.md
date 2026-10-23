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
- Nutzerfrage „Magellansche Wolken verschmelzen sofort?“ nachgemessen: Sie verschmolzen mit der **Milchstraße** (nicht mit Andromeda) nach ~2 Gyr, auf zu enger Bahn (Apozentrum 67 kpc). Ursache: Plummer-Halos zu konzentriert. Mit Hernquist-Halos: Apozentrum 130 kpc, Verschmelzung nach ~4,5 Gyr, erste MW-M31-Begegnung nach 4,8 Gyr. Plausibel gegenüber Cautun et al. 2019 und van der Marel et al. 2012/2019.
- Gefundener Fehler: Die Radiusziehung des Hernquist-Halos war falsch (s statt s² gleichverteilt), der Virialtest zeigte 0,43 statt 1.

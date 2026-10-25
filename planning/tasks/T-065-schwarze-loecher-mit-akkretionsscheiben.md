---
id: T-065
title: Schwarze Löcher mit Akkretionsscheiben
status: todo
milestone: M9
priority: 2
estimate: L
depends: []
---

## Ziel

Schwarze Löcher sollen Akkretionsscheiben bekommen: heißes Gas, das auf Kreisbahnen bis zur innersten stabilen
Bahn (ISCO) spiralt, dabei hell leuchtet und das Loch für den Betrachter sichtbar macht – mit den bekannten Effekten
aus Bildern wie dem des Event Horizon Telescope: helle, durch Doppler-Beaming einseitig aufgehellte Seite,
Gravitationsrotverschiebung nach innen, durch die Linse über und unter das Loch gebogene Rückseite der Scheibe.

## Akzeptanzkriterien

- [ ] Analyse und Entscheidung (Notiz oder ADR): reine Darstellung vs. Dynamik
      - Darstellung: dünne Scheibe (Novikov-Thorne/Shakura-Sunyaev, T(r) ∝ r^(−3/4) mit Innenkante am ISCO), Ausrichtung
        aus dem Drehimpuls des einfallenden Materials oder der Umgebung, Helligkeit aus der Akkretionsrate
      - Dynamik: Gas (SPH, T-036) mit Drehimpulstransport (α-Viskosität), Akkretion über den Horizont, Rückwirkung
        (Strahlung/Eddington-Grenze, Jets ausdrücklich nur als Ausblick)
- [ ] Akkretionsrate aus der Simulation (Masse, die das Loch pro Zeit schluckt) treibt Helligkeit und Größe
- [ ] Metal-Darstellung: Temperaturfarbe, Doppler-Beaming (∝ D³…⁴) und Rotverschiebung in der Scheibe, Linse biegt die
      Rückseite sichtbar über das Loch (Abgleich mit Schwarzschild-Raytracing-Bildern, mindestens qualitativ)
- [ ] Wirkt mit reduziertem c (größere Horizonte → große, gut sichtbare Scheiben) und realem c (punktförmig hell)
- [ ] Szene oder Startoption, die eine Scheibe gut zeigt (z. B. Loch mit Gaswolke/zerrissenem Stern, T-063)
- [ ] Lernmodus: Thema „Akkretionsscheiben“ in drei Stufen mit Modellgrenzen; Bezug zu ISCO und Kerr (T-059)
- [ ] Physik-Invarianten (Energie/Impuls) unverändert, wenn nur Darstellung; sonst Test der Massen- und
      Drehimpulsbilanz

## Notizen

Wunsch des Nutzers vom 2026-10-25. Hängt inhaltlich mit T-059 (Kerr: ISCO 1–9 GM/c², Frame-Dragging) und T-063
(Gaswolken als Materialquelle) zusammen.

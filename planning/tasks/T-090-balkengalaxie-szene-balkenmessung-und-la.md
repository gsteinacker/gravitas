---
id: T-090
title: 'Balkengalaxie: Szene, Balkenmessung und langlebige Balken'
status: todo
milestone: M9
priority: 2
estimate: M
depends: []
---

## Ziel

Eine Szene zeigt, wie aus einer rotierenden Scheibe von selbst ein Balken entsteht (Balkeninstabilität, Ostriker &
Peebles 1973), wie er über Resonanzen Drehimpuls an den lebendigen Halo abgibt, länger wird und langsamer rotiert
(Athanassoula 2002), und wie er Gas nach innen treibt.

## Akzeptanzkriterien

- [ ] Balkenmessung im Kern: m=2-Amplitude A₂ und Phase der Sterne in Ringen, Musterdrehzahl Ω_p aus der
      Phasenänderung; im Inspektor (mit (i)) und in `tools/evolve`
- [ ] Szene „Balkengalaxie“ (scheibendominiertes Inneres, kleiner Bulge, kühle Scheibe); Balken mit A₂ ≥ 0,2 über
      mindestens 2 Gyr
- [ ] Ursache klären, warum der Balken im Vorversuch nach ~1,1 Gyr zerfällt: schwere DM-Teilchen (Aufheizung),
      Glättung, Gaseinstrom und zentrale Masse (Loch), Gas-Rückkopplung. Jeweils einzeln abschalten und messen
- [ ] Test: Scheibe ohne Halo-Dominanz bildet einen Balken, die Scheibe aus Szene 1 nicht
- [ ] Lernthema „bars“ in drei Stufen (Milchstraße hat einen Balken; Resonanzen; Modellgrenzen)
- [ ] Screenshot-Prüfung

## Notizen

Vorversuch am 2026-10-30, 20 000 Körper, A₂ der Sterne in 0,75–2,5 kpc:
- Szene 1 bildet keinen Balken (A₂ ≤ 0,08, Phase springt).
- Variante mit ausgedehnterem Halo (haloScale 8 statt 4), Bulge 5 % statt 20 % und Q = 1,0 bildet in 130 Myr einen
  Balken mit A₂ = 0,25–0,33 bei stabiler Phase. Nach ~1,1 Gyr zerfällt er (A₂ → 0,05).

Echte Balken leben viele Gyr. Für die Boxy/Peanut-Form (Knick des Balkens nach oben) reicht die vertikale Auflösung
vermutlich nicht (Glättung 150 pc bei ~300 pc Scheibendicke).

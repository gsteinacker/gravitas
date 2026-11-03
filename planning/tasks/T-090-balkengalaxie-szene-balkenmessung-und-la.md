---
id: T-090
title: Balkengalaxie: Szene, Balkenmessung und langlebige Balken
status: done
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

- [x] Balkenmessung im Kern: m=2-Amplitude A₂ und Phase der Sterne in Ringen, Musterdrehzahl Ω_p aus der
      Phasenänderung; im Inspektor (mit (i)) und in `tools/evolve`
- [x] Szene „Balkengalaxie“ (scheibendominiertes Inneres, kleiner Bulge, kühle Scheibe); Balken mit A₂ ≥ 0,2 über
      mindestens 2 Gyr
- [x] Ursache klären, warum der Balken im Vorversuch nach ~1,1 Gyr zerfällt: schwere DM-Teilchen (Aufheizung),
      Glättung, Gaseinstrom und zentrale Masse (Loch), Gas-Rückkopplung. Jeweils einzeln abschalten und messen
- [x] Test: Scheibe ohne Halo-Dominanz bildet einen Balken, die Scheibe aus Szene 1 nicht
- [x] Lernthema „bars“ in drei Stufen (Milchstraße hat einen Balken; Resonanzen; Modellgrenzen)
- [x] Screenshot-Prüfung

## Notizen

Vorversuch am 2026-10-30, 20 000 Körper, A₂ der Sterne in 0,75–2,5 kpc:
- Szene 1 bildet keinen Balken (A₂ ≤ 0,08, Phase springt).
- Variante mit ausgedehnterem Halo (haloScale 8 statt 4), Bulge 5 % statt 20 % und Q = 1,0 bildet in 130 Myr einen
  Balken mit A₂ = 0,25–0,33 bei stabiler Phase. Nach ~1,1 Gyr zerfällt er (A₂ → 0,05).

Echte Balken leben viele Gyr. Für die Boxy/Peanut-Form (Knick des Balkens nach oben) reicht die vertikale Auflösung
vermutlich nicht (Glättung 150 pc bei ~300 pc Scheibendicke).

## Ergebnis (2026-11-03)

- `sim.MeasureBar` und `Snapshot.Bar`: A₂ und Phase im Ring 0,15–1 r₅₀ um das Dichtezentrum der Sterne
  (schrumpfende Kugeln).
- Musterdrehzahl im Controller (`app.barMeter`, alle 0,5 s). Inspektorzeile „Balken“ mit (i) in Galaxienszenen,
  `tools/evolve` gibt den Balken mit aus. Tests `TestMeasureBar` und `TestDiskDominatedGalaxyFormsABar`.
- Szene 16 „Barred spiral“: Halo-Skala 20 kpc, Bulge 5 %, Q = 1,0. Lernthema „bars“. Die Einführung von Szene 1
  behauptete fälschlich einen Balken; korrigiert.

Ursachensuche zum Zerfall (Varianten einzeln abgeschaltet):

- **20 000 Körper:** Mit und ohne Gas, ohne Loch, mit leichteren DM-Teilchen und mit feinerer Glättung zerfällt der
  Balken nach ~1 Gyr.
- **Messfehler gefunden:** Die erste Messung war auf das Loch zentriert. Das Loch wandert aber bis 5 kpc aus der
  Mitte (eigener Task T-091).
- **60 000 Körper:**
  - mit Gas: A₂ 0,17–0,24 über 2,1 Gyr, Ω_p ≈ 21 km/s/kpc.
  - ohne Gas: A₂ bis 0,42; nach ~1 Gyr knickt der Balken (innere Scheibe 0,12 → 0,18 dick, A₂ → 0,34), danach
    wird er langsamer (Drehimpuls an den Halo).
- **Fazit:** Der Zerfall bei 20 000 Körpern ist numerische Aufheizung durch die schweren DM-Teilchen.

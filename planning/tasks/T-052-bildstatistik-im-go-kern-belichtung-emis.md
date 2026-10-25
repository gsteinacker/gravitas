---
id: T-052
title: Bildstatistik im Go-Kern (Belichtung, Emission, Haze, Löcher)
status: done
milestone: M8
priority: 1
estimate: M
depends: []
---

## Ziel

Die Swift-App rechnet Belichtung und Löcher bisher selbst nach, Staub und Haze fehlen. Damit die Darstellungslogik
nur einmal existiert, berechnet der Go-Kern alles Grafikfreie pro Bild: Belichtung, Emissionsnormierung,
Haze-Deckkraft, projizierte Löcher (mit Linsenradius) und später die Staubabschwächung pro Körper. Swift zeichnet
nur noch.

## Akzeptanzkriterien

- [x] `engine.Prepare(viewport, fx)` liefert Belichtung, Emissionsnormierung, Haze-Deckkraft (DM, Gas), Löcher;
      neu berechnet nur bei neuem Stand, anderer Kamera, anderen Effekten oder Größe
- [x] C-API `galaxy_prepare`, Swift nutzt sie; `measure`/`projectHoles` in Swift entfallen
- [x] Tests: Belichtung konvergiert, Cache greift, Löcher werden projiziert
- [x] Bild der Spiralgalaxie unverändert (Screenshot-Vergleich)

## Notizen

`engine.Prepare` arbeitet auf demselben Puffer, den Swift zeichnet (`galaxy_prepare`). Die Belichtung berücksichtigt
wie in Swift zuvor noch keinen Doppler-Beaming-Faktor und keinen Staub; Staub folgt mit T-053.

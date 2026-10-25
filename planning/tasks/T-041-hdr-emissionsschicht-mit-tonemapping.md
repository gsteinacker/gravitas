---
id: T-041
title: HDR-Emissionsschicht mit Tonemapping
status: done
milestone: M6
priority: 2
estimate: M
depends: [T-037]
---

## Ziel

Hα-Emission wird heute additiv in 8-Bit-Bilder gezeichnet: Rot sättigt zuerst, danach laufen Grün und
Blau weiter, helle Ionisationsfronten werden rosa-weiß. Sehr schwache Sprites (Alpha < 1/255) gehen
verloren, große IC-434-Teilchen wirken fleckig. Ziel ist ein Bild wie in Hα-Aufnahmen: tiefrot, mit
weichem Übergang von der HII-Region zu den hellen Rändern.

## Akzeptanzkriterien

- [x] Emission (vor und hinter dem Staub) in eigener Float-Schicht oder als Intensität mit Reserve akkumuliert
- [x] Tonemapping pro Pixel (z. B. 1 − exp(−k I)) mit Farbe erst danach, kein Kippen nach Weiß
- [x] Hintergrund der Pferdekopf-Szene glatt (keine Teilchenflecken), Rand des Kopfes heller als IC 434
- [x] FPS der Pferdekopf-Szene bei 60k Teilchen ≥ 60

## Notizen

Entstanden aus T-038 (Screenshots bei t = 0,5: Silhouette und heller Rand stimmen, Sättigung nicht).

Erledigt 2026-10-26: Der Swift-Renderer (ADR-0011) akkumuliert alles in float16 (HDR), damit sind die ersten drei
Punkte zum Großteil schon durch den Umstieg erfüllt. Neu ist das farbtonerhaltende Tonemapping in `finalFragment`:
Der hellste Kanal wird mit 1 − exp(−1,6 m) abgebildet, die anderen werden mitskaliert, dazu kommen 25 % Abbildung
pro Kanal, damit Galaxienkerne weiß bleiben. Der helle Rand des Pferdekopfs bleibt dadurch tiefrot statt rosa-weiß.
Pferdekopf 60k: 62–65 fps.


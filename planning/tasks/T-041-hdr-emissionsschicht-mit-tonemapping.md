---
id: T-041
title: HDR-Emissionsschicht mit Tonemapping
status: todo
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

- [ ] Emission (vor und hinter dem Staub) in eigener Float-Schicht oder als Intensität mit Reserve akkumuliert
- [ ] Tonemapping pro Pixel (z. B. 1 − exp(−k I)) mit Farbe erst danach, kein Kippen nach Weiß
- [ ] Hintergrund der Pferdekopf-Szene glatt (keine Teilchenflecken), Rand des Kopfes heller als IC 434
- [ ] FPS der Pferdekopf-Szene bei 60k Teilchen ≥ 60

## Notizen

Entstanden aus T-038 (Screenshots bei t = 0,5: Silhouette und heller Rand stimmen, Sättigung nicht).

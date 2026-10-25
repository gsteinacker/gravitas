---
id: T-069
title: Gas- und Staubwolken beim Zoomen glatt und in richtiger Größe
status: done
milestone: M10
priority: 2
estimate: M
depends: []
---

## Ziel

Beim Hineinzoomen wirken Gas- und Staubwolken grob gekörnt, und alle Wolken scheinen gleich groß. Aus der Ferne passt
es. Ursache vermutlich: Sprite-Größen sind auf feste Pixelgrenzen geklemmt (z. B. Staub 2–80 px, Emission 3–120 px,
Schleier 6–48 px) statt der SPH-Glättungslänge h zu folgen, und die Staubkarte hat eine feste Gitterauflösung.

## Akzeptanzkriterien

- [x] Sprite-Größe folgt der Glättungslänge (2h) ohne obere Pixelgrenze; Kernel-Form statt hartem Rand
- [x] Staubkarte/Abschwächung bleibt beim Zoomen glatt (Auflösung an Zoom anpassen oder Staub auf der GPU)
- [x] Wolken unterschiedlicher Masse/Größe sehen unterschiedlich groß aus (Screenshot nah und fern)
- [x] Leistung beim Hineinzoomen bleibt ≥ 60 FPS (große Sprites: Überzeichnung prüfen)

## Notizen

Rückmeldung des Nutzers vom 2026-10-25.

Erledigt 2026-10-26:
- Staub, Hα und Gaskarte zeichnen einen glatten Kern (1 − d²)³ mit Durchmesser 2h bzw. 3h, ohne obere Pixelgrenze.
  Die Gaskarte hatte vorher eine feste Weltgröße (0,12) für alle Teilchen. Größer als 1,5 Bildbreiten wird nur der
  Rand abgeschnitten, die Flächenhelligkeit bzw. optische Tiefe kommt aus der wahren Größe.
- Die Gaskarte ist jetzt eine massengewichtete Säulendichte statt einer Bedeckung. Dünnes, großes Gas ist
  schwächer, dichte Knoten sind kräftiger.
- Staubkarte im Go-Kern: 7 statt 5 Pyramidenstufen (bis 64× die feinste Zelle), Fußabdruck ohne 80-px-Grenze.
- Screenshots: Pferdekopf ×4 und ×15, Spirale ×6 mit Staub, Spirale mit Gaskarte. Glatt, Größen folgen der Dichte.
  FPS: 62–80 (×15 mit Gaskarte 63).


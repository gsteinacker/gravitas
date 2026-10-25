---
id: T-069
title: Gas- und Staubwolken beim Zoomen glatt und in richtiger Größe
status: todo
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

- [ ] Sprite-Größe folgt der Glättungslänge (2h) ohne obere Pixelgrenze; Kernel-Form statt hartem Rand
- [ ] Staubkarte/Abschwächung bleibt beim Zoomen glatt (Auflösung an Zoom anpassen oder Staub auf der GPU)
- [ ] Wolken unterschiedlicher Masse/Größe sehen unterschiedlich groß aus (Screenshot nah und fern)
- [ ] Leistung beim Hineinzoomen bleibt ≥ 60 FPS (große Sprites: Überzeichnung prüfen)

## Notizen

Rückmeldung des Nutzers vom 2026-10-25.

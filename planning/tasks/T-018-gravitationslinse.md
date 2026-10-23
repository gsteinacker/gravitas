---
id: T-018
title: Gravitationslinsen-Shader
status: done
milestone: M4
priority: 3
estimate: M
depends: [T-016]
---

## Ziel

Optischer Wow-Effekt um Schwarze Löcher.

## Akzeptanzkriterien

- [x] Shader verzerrt das Bild (Sterne und Hintergrund) nach der Punktlinsengleichung, θ_E = 5 r_s auf dem Bildschirm, bis 16 Löcher

## Notizen
- Visuelle Näherung: Es wird alles verzerrt, auch Sterne vor dem Loch, weil es keine Tiefentrennung gibt. θ_E ist eine Designwahl (der echte Wert hängt von den Abständen ab). Taste `l`.
- Opaker Sternhimmel als Hintergrund, damit die Linse sichtbar wird.

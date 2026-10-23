---
id: T-019
title: Bahnspuren
status: done
milestone: M4
priority: 3
estimate: S
depends: [T-011]
---

## Ziel

Bahnen sichtbar machen.

## Akzeptanzkriterien

- [x] Fade-Out für alle: max(0,93 · Spur − 1, Sterne) pro Frame

## Notizen
- Maximum-Blending statt Addition, sonst sättigen langsame, dichte Bereiche (Faktor ~14).
- Konstanter Abzug von 1/255 pro Frame, weil multiplikatives Verblassen in 8 Bit nie 0 erreicht und einen grauen Schleier hinterlässt. Taste `t`, standardmäßig aus.

---
id: T-015
title: Lichtgeschwindigkeit im UI einstellbar
status: done
milestone: M3
priority: 2
estimate: S
depends: [T-012]
---

## Ziel

Mit c spielen können.

## Akzeptanzkriterien

- [x] Tasten für c: `<` `>` (×1,25, Bereich 1…1000), Startparameter `-c`
- [x] BHs in Horizontgröße r_s, Ring an der Photonensphäre (1,5 r_s), Mindestgröße 3 px

## Notizen
- HUD zeigt c und r_s der nächsten BH-Masse. Default c = 10 in allen Szenen.
- Hinweis: Alles wird ohne Tiefensortierung gezeichnet, das BH verdeckt also auch Sterne vor ihm. Lensing und Tiefe sind Sache von T-018.

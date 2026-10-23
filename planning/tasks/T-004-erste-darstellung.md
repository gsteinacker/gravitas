---
id: T-004
title: Erste Darstellung mit Ebitengine
status: done
milestone: M0
priority: 1
estimate: M
depends: [T-003]
---

## Ziel

Ein Fenster zeigt die laufende Simulation als Punktwolke.

## Akzeptanzkriterien

- [x] `make run` öffnet ein Fenster mit ≥ 5k Sternen
- [x] Kamera: Zoom (Mausrad), Pan (Ziehen), Rotation
- [x] Simulation läuft entkoppelt vom Rendering

## Notizen
- Screenshot-Modus `-shot <png> -shot-after <s>` für die visuelle Prüfung durch den Agenten.
- Steuerung: Linksziehen rotiert, Shift- oder Rechtsziehen verschiebt, Mausrad zoomt.

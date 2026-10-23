---
id: T-007
title: HUD: FPS, N, Schritte/s, Energie-Drift
status: todo
milestone: M1
priority: 2
estimate: S
depends: [T-004]
---

## Ziel

Sichtbares Feedback über Zustand und Performance.

## Akzeptanzkriterien

- [x] Overlay ein-/ausblendbar (H)
- [x] Hilfe-Overlay mit Tastenbelegung

## Notizen
- FPS, N, Schritte/s und Zeit sind sichtbar. Es fehlt noch die Energie-Drift, die O(N²) kostet: periodisch im Sim-Goroutine berechnen.

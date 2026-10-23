---
id: T-007
title: HUD: FPS, N, Schritte/s, Energie-Drift
status: done
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
- [x] FPS, N, Schritte/s, Simulationszeit
- [x] Energie-Drift seit letzter Szenen-Änderung

## Notizen
- Drift wird alle 2 s auf der Sim-Goroutine gemessen. Neue Basis bei jeder Epoche (`Simulation.Epoch`: Reset oder hinzugefügte Körper), weil sich E dann legitim ändert.
- N ≤ 30.000: exakt (paralleles O(N²), ~70 ms). Darüber: Baum-Schätzung mit θ = 0,3, ~95 ms bei 100k, also ~5 % Durchsatz.
- Messung (N = 3000, 2000 Schritte), Drift exakt gegenüber Baum: θ = 0,6 folgt der Drift **nicht** (Vorzeichen teils falsch), θ = 0,3 liegt etwa ±30 % daneben. Darum zeigt das HUD die Schätzung mit `~` und „tree estimate“. Sie taugt nur als „explodiert es?“-Indikator.
- Genauere Baum-Energie bräuchte Quadrupol-Momente (siehe Notiz in T-008).
- Tastenbelegung nebenbei auf eingegebene Zeichen umgestellt (deutsches Layout), siehe CLAUDE.md.

---
id: T-017
title: Doppler- und Gravitations-Rotverschiebung (visuell)
status: done
milestone: M4
priority: 3
estimate: S
depends: [T-016]
---

## Ziel

Relativistische Effekte sichtbar machen.

## Akzeptanzkriterien

- [x] Farbverschiebung nach Radialgeschwindigkeit zur Kamera, inkl. transversalem Doppler, Gravitations-Rotverschiebung √(1 − r_s/r) und Beaming ∝ f³ (begrenzt)

## Notizen
- `render/shift.go`, getestet: ±0,6c ergibt Faktor 2 bzw. 0,5, transversal 1/γ = 0,8. Taste `d`.
- Gut sichtbar bei geneigter Scheibe: Die zulaufende Seite ist blau, die ablaufende rot (c = 10, v ≈ 0,1–0,5c).
- Snapshots enthalten dafür Koordinatengeschwindigkeiten v = u/γ.

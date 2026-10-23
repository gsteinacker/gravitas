---
id: T-012
title: Geschwindigkeitslimit über Eigengeschwindigkeit
status: done
milestone: M3
priority: 1
estimate: S
depends: [T-003]
---

## Ziel

Kein Körper wird schneller als c.

## Akzeptanzkriterien

- [x] Integration von u = γv
- [x] Test: |v| < c auch bei extremen Beschleunigungen

## Notizen
- `sim/relativity.go`: Kick auf u, Drift mit u/γ. Tests: konstante Beschleunigung 10⁶ (|v| < c in jedem Schritt, u = a·t exakt), Newtonscher Grenzfall (c = 10⁹ entspricht c = 0 auf 10⁻⁹), relativistischer Orbit bei 0,4 c (H-Drift < 10⁻⁵, Newtonsche Energie driftet messbar → Test ist trennscharf).

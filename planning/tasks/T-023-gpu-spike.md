---
id: T-023
title: Spike: GPU-Solver (WebGPU aus Go) für 10⁶ Sterne
status: todo
milestone: M2
priority: 2
estimate: M
depends: [T-008]
---

## Ziel

Klären, ob 10⁶ Sterne interaktiv möglich sind.

## Akzeptanzkriterien

- [ ] Prototyp oder begründete Absage per ADR

## Notizen
- Messung 2026-10-23: GPU-Direktsumme mit 185 Mrd. Paaren/s ist ~30× schneller als die Go-CPU, reicht aber für 1M nicht (5,3 s/Schritt). Nötig ist ein GPU-Baumcode. Siehe docs/analysis/01-tech-stack.md, Abschnitt 6.
- Kandidaten: cogentcore.org/core/gpu + gosl (Go→WGSL), github.com/gogpu/wgpu (reines Go). Fallback: Metal über purego.
- Erstes Ziel des Spikes: direkte Summe auf der GPU aus Go (100k in < 60 ms), danach Machbarkeit eines LBVH/Octree-Builds.

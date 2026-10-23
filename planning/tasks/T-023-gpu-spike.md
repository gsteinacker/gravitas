---
id: T-023
title: Spike: GPU-Solver (WebGPU aus Go) für 10⁶ Sterne
status: done
milestone: M2
priority: 2
estimate: M
depends: [T-008]
---

## Ziel

Klären, ob 10⁶ Sterne interaktiv möglich sind.

## Akzeptanzkriterien

- [x] Prototyp oder begründete Absage per ADR: Prototyp funktioniert und ist Standard auf macOS (ADR-0004)

## Notizen
- Messung 2026-10-23: GPU-Direktsumme mit 185 Mrd. Paaren/s ist ~30× schneller als die Go-CPU, reicht aber für 1M nicht (5,3 s/Schritt). Nötig ist ein GPU-Baumcode. Siehe docs/analysis/01-tech-stack.md, Abschnitt 6.
- Kandidaten: cogentcore.org/core/gpu + gosl (Go→WGSL), github.com/gogpu/wgpu (reines Go). Fallback: Metal über purego.
- Erstes Ziel des Spikes: direkte Summe auf der GPU aus Go (100k in < 60 ms), danach Machbarkeit eines LBVH/Octree-Builds.
- Ansatz geändert: Die reine GPU-Direktsumme ist uninteressant, seit der CPU-Baum schneller ist. Stattdessen läuft der Baumdurchlauf auf der GPU, die CPU baut den Baum.
- Ergebnis: 1M Kräfte in 40 ms (CPU 155 ms), 100k in 4,6 ms (CPU 11,8 ms). App mit 1M bei 12–13 Schritten/s, 12–18 FPS.
- Nebenbei behoben: Energie-Diagnose dominierte bei 1M (66 % CPU) → adaptives Intervall. Screenshot-Modus wartete auf Frames statt Zeit und nahm Eingaben an → jetzt Zeit und keine Eingabe.
- Rendering bei 1M: Projektion mit vorberechneter Kamera-Trigonometrie, Quads parallel aufgebaut, Snapshot parallel befüllt.
- Offen (Folge-Tasks): Teilbäume parallel bauen, Sterne direkt als GPU-Punkte rendern.

---
id: T-024
title: App erreicht nur gut die Hälfte des Benchmark-Durchsatzes
status: done
milestone: M2
priority: 2
estimate: S
depends: [T-009]
---

## Ziel

In der App mit N=100k: ~30 Schritte/s, der isolierte Benchmark schafft ~53. Ursache finden (Konkurrenz mit Render-Thread, serielles Snapshot-Kopieren, serielles Kick/Drift?) und beheben.

## Akzeptanzkriterien

- [x] Profil der laufenden App (pprof) liegt vor (`-cpuprofile`)
- [x] ~~≥ 45 Schritte/s bei N=100k in der App~~ Kriterium verworfen: Es beruhte auf einem Benchmark des Anfangszustands. Die App erreicht die headless-Rate des jeweiligen Zustands. Ziel weiter in T-027.

## Notizen
- Erste Hypothese: Der Renderer hielt das Snapshot-Lock während des ganzen Frames, also blockierte Zeichnen das Veröffentlichen. Behoben per Triple-Buffer, mit Test `TestSlowRendererDoesNotBlockStepping`. CPU-Auslastung 616 → 714 %, Schritte/s aber unverändert.
- Tatsächliche Ursache: Die Kraftberechnung wird teurer, während der Galaxiekern dichter wird (19 → 38 ms/Schritt von t = 0 bis t = 2,8, headless). Der Benchmark maß nur t = 0.
- Nebenbei leafSize 12 → 24 (−10 % am entwickelten Zustand). Details in docs/perf-log.md.

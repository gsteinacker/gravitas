---
id: T-024
title: App erreicht nur gut die Hälfte des Benchmark-Durchsatzes
status: todo
milestone: M2
priority: 2
estimate: S
depends: [T-009]
---

## Ziel

In der App mit N=100k: ~30 Schritte/s, der isolierte Benchmark schafft ~53. Ursache finden (Konkurrenz mit Render-Thread, serielles Snapshot-Kopieren, serielles Kick/Drift?) und beheben.

## Akzeptanzkriterien

- [ ] Profil der laufenden App (pprof) liegt vor
- [ ] ≥ 45 Schritte/s bei N=100k in der App

## Notizen

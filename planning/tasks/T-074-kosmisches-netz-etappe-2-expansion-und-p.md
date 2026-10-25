---
id: T-074
title: Kosmisches Netz Etappe 2: Expansion und Particle-Mesh
status: todo
milestone: M11
priority: 3
estimate: L
depends: [T-073]
---

## Ziel

Struktur wächst vor den Augen von z = 50 bis heute: mitbewegte Koordinaten, Friedmann-Expansion (ΛCDM) und
periodische Particle-Mesh-Kräfte (Etappe 2 aus `docs/analysis/04-universe.md`).

## Akzeptanzkriterien

- [ ] ADR „Kosmologischer Modus“: mitbewegte Koordinaten, Kick/Drift-Faktoren (Quinn et al. 1997), PM periodisch
- [ ] PM mit CIC und FFT (128³), parallel; Benchmark und Eintrag in `docs/perf-log.md`
- [ ] Tests: lineares Wachstum einer Einzelmode folgt D(a); Layzer-Irvine-Bilanz; Impuls im periodischen Würfel
- [ ] Anzeige von z und kosmischer Zeit; c, Löcher, Linse im Modus aus
- [ ] Lernmodus: Expansion, Dunkle Energie, Rotverschiebung

## Notizen

Aus der Analyse T-062. Danach optional Etappe 3 (TreePM, Friends-of-Friends).

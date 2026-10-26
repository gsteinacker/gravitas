---
id: T-074
title: Kosmisches Netz Etappe 2: Expansion und Particle-Mesh
status: doing
milestone: M11
priority: 3
estimate: L
depends: [T-073]
---

## Ziel

Struktur wächst vor den Augen von z = 50 bis heute: mitbewegte Koordinaten, Friedmann-Expansion (ΛCDM) und
periodische Particle-Mesh-Kräfte (Etappe 2 aus `docs/analysis/04-universe.md`).

## Akzeptanzkriterien

- [x] ADR „Kosmologischer Modus“: mitbewegte Koordinaten, Kick/Drift-Faktoren (Quinn et al. 1997), PM periodisch
- [x] PM mit CIC und FFT (128³), parallel; Benchmark und Eintrag in `docs/perf-log.md`
- [x] Tests: lineares Wachstum einer Einzelmode folgt D(a); Layzer-Irvine-Bilanz; Impuls im periodischen Würfel
- [x] Anzeige von z und kosmischer Zeit; c, Löcher, Linse im Modus aus
- [x] Lernmodus: Expansion, Dunkle Energie, Rotverschiebung

## Notizen

Aus der Analyse T-062. Danach optional Etappe 3 (TreePM, Friends-of-Friends).

Stand 2026-10-26:
- ADR-0014, `internal/sim/comoving.go` (Integrator), `internal/cosmo/pm.go` (PM mit CIC-Entfaltung, parallele FFT).
- Tests: Einzelmode wächst 12,15 statt 12,42 (−2 %); Layzer-Irvine 0,4 %; Impuls; Szene 13 baut ihren Würfel
  (`TestStructureFormationGrows`); `-race` für `cosmo` und `sim` grün.
- 12 ms pro Schritt bei 50k (perf-log).
- Szene 13 „Structure formation“ (z = 50 → 0), Anzeige z und a im Inspektor, Lernthema „expansion“. Die
  Energiedrift-Anzeige ist im Modus aus, der Skalenfaktor wird im Speicherstand gesichert.
- Offen: Screenshot-Prüfung des wachsenden Netzes. Der Bildschirm war gesperrt.


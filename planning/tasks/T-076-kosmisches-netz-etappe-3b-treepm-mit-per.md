---
id: T-076
title: Kosmisches Netz Etappe 3b: TreePM mit periodischem Baum
status: blocked
milestone: M11
priority: 3
estimate: L
depends: [T-075]
---

## Ziel

Halos auflösen: Kräfte als TreePM (GADGET-2, Springel 2005). Particle-Mesh für große Abstände, der vorhandene
Barnes-Hut-Baum für kurze, getrennt mit einem erfc-Split. Etappe 3 aus `docs/analysis/04-universe.md`.

## Akzeptanzkriterien

- [ ] ADR „TreePM im kosmologischen Modus“ (Split-Skala r_s, Abschneideradius, periodische Bilder im Baum)
- [ ] Kurzreichweitiger Baumanteil mit erfc-Faktor und nächstem periodischem Bild; GPU-Pfad angepasst oder
      bewusst auf CPU begrenzt (`TestGPUWalkMatchesCPU`)
- [ ] Tests: Kraft zwischen zwei Teilchen folgt über alle Abstände 1/r² (Summe PM + Baum, Fehler < 1 %);
      Impuls im periodischen Würfel; Layzer-Irvine-Bilanz; lineares Wachstum D(a) unverändert
- [ ] Benchmark vorher/nachher (`BenchmarkAccelerationsEvolved`), Eintrag in `docs/perf-log.md`
- [ ] Halo-Massenfunktion (T-075) näher an Tinker et al. 2008 als mit reinem PM
- [ ] Lernmodus: Physik-Stufe nennt den Split und seine Grenzen

## Notizen

Physik-Änderung: physics-reviewer einsetzen.

- 2026-11-04: Blockiert, die Szenen zum kosmischen Netz sind entfernt (T-094). Nur aufnehmen, wenn sie zurückkommen.

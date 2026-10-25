---
id: T-073
title: Kosmisches Netz Etappe 1: Zel'dovich-Szene mit Dichtefärbung
status: todo
milestone: M11
priority: 2
estimate: M
depends: [T-062]
---

## Ziel

Eine Szene zeigt das kosmische Netz, also Filamente, Knoten und Voids, als Zel'dovich-Realisierung eines
ΛCDM-Leistungsspektrums (Etappe 1 aus `docs/analysis/04-universe.md`), ohne neues Physikmodell.

## Akzeptanzkriterien

- [ ] Szenen-Komponente `cosmicweb`: Würfel L (Mpc/h), Teilchen n³, Rotverschiebung z, Seed; P(k) nach Eisenstein &
      Hu 1998, normiert auf σ₈; Zel'dovich-Verschiebung per eigener 3D-FFT (Stdlib)
- [ ] Test: Varianz des Dichtefelds folgt dem Wachstumsfaktor D(z); Spektrum der ICs reproduziert P(k)
- [ ] Darstellung: Dunkle Materie dichtegefärbt (log ρ), Szene mit Mpc-Einheiten und Hinweis „ohne Expansion“
- [ ] Lernmodus-Thema „Kosmisches Netz“ (drei Stufen, Modellgrenze: Zel'dovich schmiert bei z = 0, kein Expansion)

## Notizen

Aus der Analyse T-062.

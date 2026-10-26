---
id: T-073
title: Kosmisches Netz Etappe 1: Zel'dovich-Szene mit Dichtefärbung
status: doing
milestone: M11
priority: 2
estimate: M
depends: [T-062]
---

## Ziel

Eine Szene zeigt das kosmische Netz, also Filamente, Knoten und Voids, als Zel'dovich-Realisierung eines
ΛCDM-Leistungsspektrums (Etappe 1 aus `docs/analysis/04-universe.md`), ohne neues Physikmodell.

## Akzeptanzkriterien

- [x] Szenen-Komponente `cosmicweb`: Würfel L (Mpc/h), Teilchen n³, Rotverschiebung z, Seed; P(k) nach Eisenstein &
      Hu 1998, normiert auf σ₈; Zel'dovich-Verschiebung per eigener 3D-FFT (Stdlib)
- [x] Test: Varianz des Dichtefelds folgt dem Wachstumsfaktor D(z); Spektrum der ICs reproduziert P(k)
- [ ] Darstellung: Dunkle Materie dichtegefärbt (log ρ), Szene mit Mpc-Einheiten und Hinweis „ohne Expansion“
- [x] Lernmodus-Thema „Kosmisches Netz“ (drei Stufen, Modellgrenze: Zel'dovich schmiert bei z = 0, kein Expansion)

## Notizen

Aus der Analyse T-062.

Stand 2026-10-26:
- Paket `internal/cosmo` (Stdlib): Radix-2-FFT (1D/3D), Planck-2018-ΛCDM, Wachstumsfaktor D(a) per Integral,
  f ≈ Ω_m^0,55, Eisenstein-Hu-Transferfunktion ohne Oszillationen, σ₈-Normierung, Zel'dovich-Feld.
- Tests: FFT gegen DFT, D(a) = a für EdS und D(z=1) ≈ 0,61 für ΛCDM, σ₈ exakt, Varianz des Feldes gleich Σ P/V,
  −∇·Ψ korreliert mit δ.
- Szene 12 „Cosmic web“: Würfel 100 Mpc/h, z = 0, Dunkle Materie, startet angehalten (`paused`); Masse
  Ω_m ρ_krit V getestet; Lernthema „cosmic-web“.
- Offen: Prüfung der Darstellung (DM-Nebel als Dichte) per Screenshot. Der Bildschirm war gesperrt.


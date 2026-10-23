---
id: T-037
title: Ionisation mit Schattenwurf und Emissionsleuchten
status: done
milestone: M6
priority: 1
estimate: M
depends: [T-036]
---

## Ziel

Junge heiße Sterne ionisieren Gas: ionisiertes Gas wird auf ~10⁴ K geheizt, leuchtet in Hα und
verdampft; dichte Klumpen werfen Schatten, hinter denen kaltes Gas überlebt (Säulen, Pferdekopf).

## Akzeptanzkriterien

- [x] Quellen über `Lum` (Szenen-Komponente `star`, `luminosity`)
- [x] Optische Tiefe τ = κ_n ∫ρ_n dl + κ_r ∫ρ_i² dl auf 64³-Gitter (Cloud-in-Cell, trilinear), Ion = max exp(−τ)
- [x] Photoheizung auf `UIon`, Schatten kühlen auf `UFloor`; Sternentstehung nur in neutralem Gas
- [x] Hα-Emission ∝ x² m ρ, hinter Staub absorbiert, eigene Belichtung; Gaskarte zeigt Ion
- [x] Tests: Schatten, Transparenz ohne Rekombination; Spielstände speichern Lum/Ion/H (Format 3)

## Notizen

- Gittergrenzen: Quantile (0,5–99,5 %) des optisch relevanten Gases (τ über die eigene Glättungslänge
  ≥ 0,1) plus 10 % Rand. Vorher spannte das Gitter über alles Gas; ins Vakuum strömendes heißes Gas
  (rmax bis 65 pc) machte die Zellen 0,7 pc groß, Schatten verschwanden. In dem Lauf trat auch ein NaN
  bei t ≈ 0,24 auf, mit den neuen Grenzen nicht mehr (Wächter in `tools/evolve`).
- Selbstausschluss nur noch eine halbe Zelle vor dem Ziel; vorher blieb eine ganze Zelle (0,27 pc)
  unabgeschirmt, dichte Schalen waren teilionisiert und überstrahlten alles.
- Grenzen: kein Photonenbudget, keine geometrische Verdünnung, keine diffuse Strahlung. Darstellung
  sättigt in 8 Bit → T-041.

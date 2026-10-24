---
id: T-040
title: Stellarer Bulge fuer Galaxien (Staubbaender vor hellem Kern)
status: done
milestone: M6
priority: 3
estimate: S
depends: [T-035]
---

## Ziel

Echte Spiralgalaxien haben einen kugelförmigen Bulge. Vor ihm werden Staubbänder markant (Sombrero), außerdem gehört er zu realistischen Rotationskurven.

## Akzeptanzkriterien

- [x] Komponente `bulgeFraction` (Hernquist-Sternsphäre, isotrope Jeans-Geschwindigkeiten)
- [x] Sichtprüfung: Staubband vor dem Bulge bei Kantenansicht

## Notizen

- Jeans-Dispersion numerisch im Gesamtpotential (Bulge, Scheibe sphärisch, Halo, Loch), Fluchtgeschwindigkeit
  aus demselben Potential. Test: 2K/|U| ≈ 1, Halbmassenradius über ~5 dynamische Zeiten stabil (±10 %).
- Bulges: Szenen 1, 2, 6, 7 (15–20 %, NGC 5195 40 %), Milchstraße 15 %, M31 30 %; M33, LMC, SMC keiner.
- Das Staubband war mit der alten binären Vor/Hinter-Aufteilung nicht zu sehen: In Kantenansicht lag die halbe
  Scheibe „vor“ der mittleren Staubtiefe und überstrahlte alles. Neu: Durchlässigkeit pro Stern aus dem Staub
  vor ihm (Gauß-Näherung der Tiefenverteilung, Bildpyramide), τ physikalisch aus den Szeneneinheiten
  (A_V aus N_H). Die Pferdekopf-Silhouette wird dadurch ebenfalls schärfer.
- Neues Lernthema `bulge`; `dust` beschreibt das neue Modell.

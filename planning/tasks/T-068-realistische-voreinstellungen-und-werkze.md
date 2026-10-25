---
id: T-068
title: Realistische Voreinstellungen und Werkzeug Galaxie
status: done
milestone: M10
priority: 1
estimate: L
depends: [T-063]
---

## Ziel

Die Werkzeuge sollen mit realistischen Werten starten, passend zum Maßstab der Szene, statt mit einer Einheitsmasse
von 0,5 (in der Spiralgalaxie 10¹⁰ M☉ für jedes Objekt). Dazu kommt eine ganze Galaxie als vierter Objekttyp.
Beim Doppelsystem zeigt der Tacho vorher, wie schnell das vorhandene Loch wird (Entscheidung des Nutzers, Variante 2).

## Akzeptanzkriterien

- [x] Voreinstellungen je Werkzeug nach Szenenmaßstab (Längeneinheit): Galaxien-Szenen Loch 4·10⁶ M☉, Haufen 10⁵ M☉,
      Wolke 10⁶ M☉, Galaxie 2·10¹⁰ M☉; pc-Szenen Loch 10 M☉, Haufen 10³ M☉, Wolke 10³ M☉; ohne Einheiten wie bisher
- [x] Jedes Werkzeug merkt sich seine Masse; Massenregler über einen physikalisch sinnvollen Bereich (ab ~1 M☉)
- [x] Radien realistisch: Sternhaufen r ≈ 3 pc·(M/10⁵ M☉)^⅓, Wolken R ≈ 20 pc·(M/10⁶ M☉)^½ (Larson)
- [x] Werkzeug Galaxie (Taste, Menü, Inspektor, Startrampe): Scheibe, Bulge, Gas, Halo, Zentralloch nach der
      Spiralgalaxie, skaliert mit M^⅓; Scheibe in der Startebene, Neigung einstellbar (0° gleichsinnig, 180°
      gegensinnig); nur in Szenen im Galaxienmaßstab
- [x] Gaswolke: Vorhersage im Werkzeug (gebunden → Kollaps oder zerstreut) aus Virialparameter mit Turbulenz und
      Temperatur (inkl. Temperaturuntergrenze der Szene)
- [x] Doppelsystem: Tacho zeigt die Geschwindigkeit, die das vorhandene Loch erhält
- [x] Tests: Voreinstellungen je Szene, Radien, Galaxie (Masse, Schwerpunkt, Drehsinn), Vorhersage

## Notizen

- Kern: Massen je Werkzeug (`defaultMasses`, `SetTool`), Radien nach Beobachtungsrelationen, `AddTemplateGalaxy`
  (Spiralgalaxie, Längen ∝ M^⅓, Einheiten der Zielszene), Ausrichtung in der Startebene mit Neigung,
  `CloudForecast` (α = 5σ²/… + thermischer Anteil, Temperaturuntergrenze der Szene), `HoleKick`.
- App: vierte Werkzeugwahl (Taste a, gesperrt in pc-Szenen), Neigungsregler, Massenregler ab ~1 M☉, Prognose und
  Loch-Geschwindigkeit im Tacho, Geist je Werkzeug.
- Screenshots: Galaxie (2·10¹⁰ M☉) neben der Spirale mit Gezeitenschweif; Wolke 10⁶ M☉ in der Spirale: „zerstreut
  sich (α = 2,0)“ wegen der 10⁴-K-Untergrenze.

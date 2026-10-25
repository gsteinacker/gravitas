---
id: T-068
title: Realistische Voreinstellungen und Werkzeug Galaxie
status: doing
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

- [ ] Voreinstellungen je Werkzeug nach Szenenmaßstab (Längeneinheit): Galaxien-Szenen Loch 4·10⁶ M☉, Haufen 10⁵ M☉,
      Wolke 10⁶ M☉, Galaxie 2·10¹⁰ M☉; pc-Szenen Loch 10 M☉, Haufen 10³ M☉, Wolke 10³ M☉; ohne Einheiten wie bisher
- [ ] Jedes Werkzeug merkt sich seine Masse; Massenregler über einen physikalisch sinnvollen Bereich (ab ~1 M☉)
- [ ] Radien realistisch: Sternhaufen r ≈ 3 pc·(M/10⁵ M☉)^⅓, Wolken R ≈ 20 pc·(M/10⁶ M☉)^½ (Larson)
- [ ] Werkzeug Galaxie (Taste, Menü, Inspektor, Startrampe): Scheibe, Bulge, Gas, Halo, Zentralloch nach der
      Spiralgalaxie, skaliert mit M^⅓; Scheibe in der Startebene, Neigung einstellbar (0° gleichsinnig, 180°
      gegensinnig); nur in Szenen im Galaxienmaßstab
- [ ] Gaswolke: Vorhersage im Werkzeug (gebunden → Kollaps oder zerstreut) aus Virialparameter mit Turbulenz und
      Temperatur (inkl. Temperaturuntergrenze der Szene)
- [ ] Doppelsystem: Tacho zeigt die Geschwindigkeit, die das vorhandene Loch erhält
- [ ] Tests: Voreinstellungen je Szene, Radien, Galaxie (Masse, Schwerpunkt, Drehsinn), Vorhersage

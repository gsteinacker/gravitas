---
id: T-063
title: Gas- und Staubwolken platzieren
status: done
milestone: M10
priority: 2
estimate: M
depends: [T-058, T-064]
---

## Ziel

Neben Schwarzen Löchern und Sternhaufen soll man auch Gas- bzw. Staubwolken in die Szene setzen können: eine
Molekülwolke, die unter Eigengravitation kollabiert, Sterne bildet, von heißen Sternen ionisiert wird und Licht
dahinter abschwächt.

## Akzeptanzkriterien

- [x] Werkzeug „Gaswolke“ (Menü, Inspektor, Taste) neben Loch und Sternhaufen
- [x] Parameter: Masse, Radius, Temperatur, Turbulenz, ggf. Rotation; in physikalischen Einheiten der Szene, sonst
      Simulationseinheiten; sinnvolle Standardwerte je Szene (Galaxie vs. Pferdekopf-Maßstab)
- [x] Aufbau wie `gasclump` der Szenen (`addGasCloud`), Teilchenzahl an Sternzahl/Profil angepasst, h konsistent
- [x] Wolke wirkt mit SPH, Kühlung, Sternentstehung, Ionisation und Staub zusammen (Screenshot-Serie)
- [x] Platzierung nutzt das 3D-Werkzeug (T-064)
- [x] Lernmodus: Hinweis/Thema zur Wolke (Jeans-Instabilität)

## Notizen

- Werkzeug „Gaswolke“ (Taste w, Menü, Inspektor, Startrampe mit bläulichem Geist). Plummer-Wolke aus SPH-Teilchen
  (`scene.AddGasClump`, wie Szenen-`gasclump`), max(400, N/40) Teilchen, Radius 1,5× Sternhaufen gleicher Masse,
  30 % Dichteschwankungen. Temperatur 10–10⁴ K (Standard 100 K), Turbulenz als Anteil von √(GM/R) (Standard 0,3).
- Masse in M☉ (`Spec.MassUnit`: M = v²L/G), Temperatur über `Spec.SpecificEnergy`.
- Keine Rotation der Wolke (optional im Kriterium).
- Screenshot-Serie Spiralgalaxie, 6·10⁸ M☉ auf Kreisbahn (`-tool cloud -mass 0.03 -launch 0.62,0.5,1,1`): Wolke
  kollabiert, 456 neue Sterne nach t = 7 (Gas 2271 → 1815); Statuszeile zeigt jetzt „neue Sterne“.
- Hinweis: Die Spiralgalaxie hat eine Temperaturuntergrenze von 10⁴ K für das Gas; kältere Wolken werden dort
  auf diesen Wert gehoben. Neues Lernthema `gas-clouds` (Jeans-Kriterium, Virialparameter).

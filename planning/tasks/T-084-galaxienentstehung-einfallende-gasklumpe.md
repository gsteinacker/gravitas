---
id: T-084
title: Galaxienentstehung: einfallende Gasklumpen und länglicher Halo
status: done
milestone: M9
priority: 3
estimate: M
depends: [T-083]
---

## Ziel

Die Szene „Galaxy formation“ soll hierarchisches Wachstum im Kleinen zeigen: Einige gasreiche Unterhalos fallen
nacheinander ein, und der Halo ist länglich statt kugelförmig. Die Scheibe kippt, verbiegt sich (Warps) und
wächst von außen nach innen.

## Akzeptanzkriterien

- [x] Baustein für einen triaxialen Halo (Achsenverhältnisse aus ΛCDM, z. B. Allgazer/Jing & Suto 2002)
- [x] Einfallende Gasklumpen mit eigenem kleinen DM-Halo auf Bahnen mit Drehimpuls, aus demselben Teilchenbudget
- [x] Messung mit `tools/evolve`: Scheibe bildet sich weiter, Aufwand pro Schritt höchstens 20 % über T-083
- [x] Lernthema „galaxy-formation“ ergänzt, Screenshot-Prüfung

## Notizen

Aus der Bewertung vom 2026-10-29: Rechenaufwand mittel (dichte Klumpen und Schocks verlangen Gas-Teilschritte).
Nach T-085 deutlich günstiger.

Erledigt 2026-10-30:
- Komponente `halo` mit `axes`, `inclination` und `orientation` (`scene.ShapeHalo`): Die Plummer-Kugel wird gestreckt.
  Test `TestTriaxialHaloKeepsItsShape`: b/a 0,82 → 0,83 und c/a 0,61 → 0,63 über zwei Umläufe.
- Szene 14: Halo 1 : 0,8 : 0,6, um 25° gegen den Spin des Gases geneigt. Drei Unterhalos (je 0,3 DM + 0,05 Gas)
  starten in 35, 45 und 55 kpc mit 0,35 v_c auf geneigten Bahnen. Teilchenmassen wie im Haupthalo, die Gesamtmasse
  bleibt gleich.
- `tools/evolve`, 8000 Körper, t = 45:

| Stand | Abplattung Baryonen | Schritte/s |
|---|---|---|
| T-083 | 0,33 | 303,2 |
| T-084 | 0,38 | 303,8 |

  Die Scheibe bildet sich also weiter, der Aufwand ist gleich. Die Unterhalos bilden schon vor dem Einfall Sterne
  (dichtes Gas ab t ≈ 15) und sind im Screenshot bei t = 45 als kompakte Klumpen auf ihren Bahnen zu sehen.

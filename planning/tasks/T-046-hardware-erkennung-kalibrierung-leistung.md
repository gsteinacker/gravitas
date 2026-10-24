---
id: T-046
title: Hardware-Erkennung, Kalibrierung, Leistungsprofile
status: done
milestone: M7
priority: 1
estimate: M
depends: [T-042]
---

## Ziel

Die App soll auf einem MacBook Air genauso flüssig laufen wie auf einem M4 Pro. Dazu misst sie beim ersten Start,
was der Rechner kann, und wählt Sternzahl, GPU-Pfad und Effektqualität danach.

## Akzeptanzkriterien

- [x] `internal/perf.Probe()`: CPU-Name, P-/E-Kerne, Speicher (sysctl), Metal-Gerätename
- [x] `Calibrate()` ≤ 1 s: misst Kraftberechnung auf GPU und CPU, wählt den schnelleren Pfad, rechnet mit N log N die
      Sternzahl für ≤ 33 ms pro Schritt hoch; Ergebnis pro Hardware in den Prefs gecacht
- [x] Profile Automatisch/Niedrig/Mittel/Hoch/Maximal (Sternzahl, Effekte, Renderauflösung, θ-Bereich)
- [x] `-n` überschreibt das Profil (reproduzierbare Messungen); Screenshot-Zeile nennt das Profil
- [x] Test der Hochrechnung; auf dem M4 Pro landet „Automatisch“ bei ≥ 30 Schritten/s

## Notizen

Kalibrierung 50–160 ms, misst zusätzlich SPH (Gas kostet pro N·log N ≈ 6,6× Gravitation), Sternzahl pro Szene
nach Gasanteil. Messung mit `-n 0` nach 20 s (unter Fremdlast: parallel lief eine Galaxy-Instanz mit 100k und
~4 Kernen): Spiralgalaxie 100k → 68 Schritte/s, Kollision 120k → 53, Antennae 93k → 61. Pferdekopf 22k Gas → 8
Schritte/s, 4 Courant-Teilschritte statt der angenommenen 1,6 → Rückkopplung über die gemessene Schrittzeit in
T-047.

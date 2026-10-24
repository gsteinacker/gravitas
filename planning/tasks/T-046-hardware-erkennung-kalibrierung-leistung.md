---
id: T-046
title: Hardware-Erkennung, Kalibrierung, Leistungsprofile
status: todo
milestone: M7
priority: 1
estimate: M
depends: [T-042]
---

## Ziel

Die App soll auf einem MacBook Air genauso flüssig laufen wie auf einem M4 Pro. Dazu misst sie beim ersten Start,
was der Rechner kann, und wählt Sternzahl, GPU-Pfad und Effektqualität danach.

## Akzeptanzkriterien

- [ ] `internal/perf.Probe()`: CPU-Name, P-/E-Kerne, Speicher (sysctl), Metal-Gerätename
- [ ] `Calibrate()` ≤ 1 s: misst Kraftberechnung auf GPU und CPU, wählt den schnelleren Pfad, rechnet mit N log N die
      Sternzahl für ≤ 33 ms pro Schritt hoch; Ergebnis pro Hardware in den Prefs gecacht
- [ ] Profile Automatisch/Niedrig/Mittel/Hoch/Maximal (Sternzahl, Effekte, Renderauflösung, θ-Bereich)
- [ ] `-n` überschreibt das Profil (reproduzierbare Messungen); Screenshot-Zeile nennt das Profil
- [ ] Test der Hochrechnung; auf dem M4 Pro landet „Automatisch“ bei ≥ 30 Schritten/s

---
id: T-035
title: Staub als Darstellungseffekt (Absorption, Roetung)
status: done
milestone: M6
priority: 1
estimate: S
depends: [T-036]
---

## Ziel

Staub sichtbar machen: dunkle Bänder, Rötung. Nutzerfrage: „Müssen wir Gas und Staub anders einfärben, um sie zu sehen?“ Ja, kaltes Gas ist optisch unsichtbar; sichtbar sind Staubabsorption, Emission ionisierten Gases (T-037) und Reflexion.

## Akzeptanzkriterien

- [x] Staub aus dem Gas: Säulendichte je Kernel-Kern (2h), automatische Normierung
- [x] Absorption exp(−τ k) mit Rötung (k_B > k_G > k_R), Taste `u`, standardmäßig an
- [x] Nur Sterne **hinter** dem Staub werden abgeschwächt (massengewichtete Staubtiefe je Bildschirmzelle, 192×120)
- [x] Gaskarte (`o`) als Falschfarben-Ansicht

## Notizen
- Erste Version schwächte alle Sterne ab, auch die vor dem Staub (gleichmäßiger brauner Fleck). Jetzt mit Tiefenaufteilung.
- Markante Staubbänder brauchen einen hellen Bulge hinter der Scheibe → Folge-Task.
- Bei aktivierten Spuren entfällt die Tiefenaufteilung (Vereinfachung).

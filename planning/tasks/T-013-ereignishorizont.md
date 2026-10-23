---
id: T-013
title: Ereignishorizont und Akkretion
status: done
milestone: M3
priority: 1
estimate: M
depends: [T-012]
---

## Ziel

Sterne verschwinden im Schwarzen Loch, BHs verschmelzen.

## Akzeptanzkriterien

- [x] Absorption innerhalb r_s = 2GM/c²
- [x] Masse- und Impulserhaltung getestet
- [x] BH-BH-Verschmelzung

## Notizen
- `sim/accrete.go`. Exakte Energiebuchhaltung in `ExternalEnergy`: Test vergleicht E_nach − E_vor mit dem Gebuchten auf 10⁻¹² relativ.
- Demo: `-n 50000 -c 2.5`. Das zentrale Loch verschluckt in t = 1,8 gut 18k Sterne (r_s wächst mit M), die Drift bleibt bei ~10⁻³.
- Beschleunigungen werden nach Stern-Absorption nicht neu berechnet (Feldänderung ≈ eine Sternmasse), nach BH-Verschmelzung schon.
- Kosten: O(N_BH · N) Suche plus O(N) parallel pro Absorption. Bei 100k und c = 10 nicht messbar.

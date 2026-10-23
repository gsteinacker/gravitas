---
id: T-025
title: Hierarchische Zeitschritte nahe Schwarzer Löcher
status: done
milestone: M3
priority: 3
estimate: M
depends: [T-008]
---

## Ziel

Die Energie-Drift kommt fast vollständig von engen Vorbeiflügen an Schwarzen Löchern mit fester Schrittweite. Körper nahe BHs mit kleineren Unterschritten integrieren (Block-Zeitschritte), ohne die globale Rate zu senken.

## Akzeptanzkriterien

- [x] Energie-Drift im Galaxie-Test um ≥ 5× kleiner (direkt 630×, Barnes-Hut 300×)
- [x] Schritte/s bei N=100k höchstens 10 % schlechter (+1,7 %, relativ gemessen)

## Notizen
- Keine Block-Zeitschritte, sondern Kraft-Aufteilung (RESPA). Stern-Stern-Kräfte kommen aus dem Baum, einmal pro Schritt. Kräfte mit Löchern sind direkt summiert und billig, deshalb lassen sie sich oft auswerten. So muss der Baum nicht pro Unterschritt neu gebaut werden.
- **Anlass war größer als gedacht:** Seit T-014 (PW, c = 10) lag die korrekt gemessene Drift bei 0,3 in 500 Schritten. τ am zentralen Loch (~0,0035) war kürzer als dt. Der alte Drift-Test maß die Newtonsche Energie und hat das verdeckt. Test korrigiert (`modelEnergy`), absoluter Wächter ergänzt (Drift < 5·10⁻³).
- Löser-Vertrag geändert: Löser liefern nur Stern-Stern-Kräfte (`sim.Solver`). Genauigkeitstest misst den Fehler jetzt gegenüber der Gesamtkraft inkl. Löcher, wie vorher effektiv auch.
- HUD zeigt die Zahl der Unterschritte.

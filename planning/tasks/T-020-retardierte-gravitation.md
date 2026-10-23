---
id: T-020
title: Spike: retardierte Gravitation für BH-Quellen
status: done
milestone: M5
priority: 3
estimate: L
depends: [T-013]
---

## Ziel

Endliche Ausbreitungsgeschwindigkeit der Gravitation ausprobieren.

## Akzeptanzkriterien

- [x] Ringpuffer der BH-Positionen (8192 Schritte, mit Geschwindigkeiten, stabile IDs)
- [x] Bewertung in ADR (ADR-0005)

## Notizen
- Drei Modi: instant (Standard), retarded (naiv), retarded+v (extrapoliert). Messungen siehe ADR-0005: naiv zerfällt ein Doppel-Loch bei 0,1c innerhalb eines Umlaufs (+57 %), extrapoliert driftet ~25× langsamer, instant ist stabil.
- Gefundene Fehler beim Bau: (1) Ein neues Loch bekam zwei History-Einträge zur selben Zeit, die Zeitachse war verschoben. (2) Feste 4 Iterationen des Lichtkegel-Fixpunkts konvergieren bei 0,3c nicht (Fehler 0,3⁴), jetzt Iteration bis zur Toleranz. Beide durch Tests gegen exakte gleichförmige Bewegung gefunden.
- Mein erster Doppel-Loch-Test verglich Endwerte nach 20 Umläufen. Der naive Fall war da gesättigt (zerfallen), deshalb misst der Test jetzt nach einem Umlauf (linearer Bereich).
- Kosten: −23 % Schritte/s bei 100k. Toleranz 1e-7 statt 1e-10 und Histories pro Schritt gebunden statt Map-Lookup pro Paar.
- Möglicher Folge-Task: EIH-1PN-Gleichungen für Loch-Paare (siehe ADR-0005, Alternativen).

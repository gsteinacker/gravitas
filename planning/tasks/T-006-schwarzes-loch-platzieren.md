---
id: T-006
title: Interaktion: Schwarzes Loch platzieren, Zeitsteuerung
status: done
milestone: M1
priority: 1
estimate: M
depends: [T-004]
---

## Ziel

Der Kern des Spielspaßes.

## Akzeptanzkriterien

- [x] Klick platziert BH mit einstellbarer Masse in der Fokusebene
- [x] Ziehen gibt dem BH eine Anfangsgeschwindigkeit (Alt-/Option-Ziehen)
- [x] Pause (Leertaste), Zeitraffer (+ -), Einzelschritt (.)

## Notizen
- Klick platziert ein ruhendes BH. Alt-Ziehen startet es am Startpunkt mit v = 0,5 · Zugvektor in der Fokusebene, mit Pfeil-Vorschau und Geschwindigkeitsanzeige. Normales Ziehen dreht weiter die Kamera, damit die gewohnte Geste bleibt.
- `.` pausiert und führt genau einen Schritt aus. Runner-Test sichert, dass Befehle im Pause-Modus ausgeführt und veröffentlicht werden.

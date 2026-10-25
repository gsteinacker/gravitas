---
id: T-061
title: Erklärung per (i) an jeder Einstellung
status: todo
milestone: M10
priority: 2
estimate: L
depends: [T-058]
---

## Ziel

Jede Einstellung (Inspektor, Einstellungsfenster, Menü-Schalter) soll über ein (i)-Symbol eine Erklärung öffnen: was
die Einstellung steuert, was das physikalisch bzw. technisch bedeutet und welche Auswirkungen sie hat. Wo es das
Verständnis erleichtert, mit grafischer Visualisierung.

## Akzeptanzkriterien

- [ ] (i)-Knopf neben jeder Einstellung im Inspektor und im Einstellungsfenster, öffnet ein Popover
- [ ] Texte de/en, in drei Stufen wie der Lernmodus (Schüler, Interessierte, Physik), Stufe folgt der Lernmodus-
      Einstellung; Inhalte als Daten (wie `internal/learn/topics`), physikalisch korrekt mit Modellgrenzen
- [ ] Je Einstellung: was sie steuert, Bedeutung, Auswirkung (auf Bild, Physik und Leistung), Empfehlung/Standard
- [ ] Visualisierungen, wo sinnvoll, z. B.: c → Horizont- und ISCO-Größe über c; θ → Fehler und Rechenzeit über θ
      (Daten aus perf-log); Loch-Gravitation → Skizze retardierte Position; Staub → Abschwächung je Farbkanal;
      Leistungsprofil → Sterne und Schritte/s; Masse → Schwarzschild-Radius
- [ ] Verweis vom Popover ins passende Lernmodus-Thema
- [ ] Test: jede Einstellung hat eine Erklärung in beiden Sprachen und allen Stufen

## Notizen

Wunsch des Nutzers vom 2026-10-25. Grafiken mit Swift Charts bzw. SwiftUI-Canvas.

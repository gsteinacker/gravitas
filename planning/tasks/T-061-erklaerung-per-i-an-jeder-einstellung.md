---
id: T-061
title: Erklärung per (i) an jeder Einstellung
status: done
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

- [x] (i)-Knopf neben jeder Einstellung im Inspektor und im Einstellungsfenster, öffnet ein Popover
- [x] Texte de/en, in drei Stufen wie der Lernmodus (Schüler, Interessierte, Physik), Stufe folgt der Lernmodus-
      Einstellung; Inhalte als Daten (wie `internal/learn/topics`), physikalisch korrekt mit Modellgrenzen
- [x] Je Einstellung: was sie steuert, Bedeutung, Auswirkung (auf Bild, Physik und Leistung), Empfehlung/Standard
- [x] Visualisierungen, wo sinnvoll, z. B.: c → Horizont- und ISCO-Größe über c; θ → Fehler und Rechenzeit über θ
      (Daten aus perf-log); Loch-Gravitation → Skizze retardierte Position; Staub → Abschwächung je Farbkanal;
      Leistungsprofil → Sterne und Schritte/s; Masse → Schwarzschild-Radius
- [x] Verweis vom Popover ins passende Lernmodus-Thema
- [x] Test: jede Einstellung hat eine Erklärung in beiden Sprachen und allen Stufen

## Notizen

Wunsch des Nutzers vom 2026-10-25. Grafiken mit Swift Charts bzw. SwiftUI-Canvas.

Erledigt 2026-10-26:
- 22 Einstellungen in Inspektor, Startplanung und Einstellungsfenster haben ein (i). Die Texte stehen in
  `internal/learn/settings.json`, je drei Stufen auf Deutsch und Englisch. Die Stufe folgt der Zielgruppe des
  Lernmodus.
- Diagramme berechnet der Go-Kern (`engine.SettingInfo`, `galaxy_setting` als JSON), gezeichnet wird mit Swift
  Charts:
  - c: Horizont und ISCO des größten Lochs über c
  - θ: Kraftfehler und Rechenzeit (perf-log T-034)
  - Masse: Radius je Werkzeug, beim Loch r_s
  - Staub: Durchlass für R/G/B
  - Profil: Sterne je Profil auf diesem Rechner
  - Loch-Gravitation: Skizze sofort/retardiert/ret. + v
- Link „Im Lernmodus: …“ öffnet das Thema (`galaxy_show_topic`).
- Tests: `TestSettingsAreComplete`, `TestEverySettingInTheAppIsExplained` (gleicht die Swift-Quellen mit den Daten
  ab), `TestSettingInfoCharts`, `TestHorizonChartFollowsC`, Smoke-Test der C-API.
- Menü-Schalter haben kein (i), weil Menüs keine Popover zeigen. Dieselben Einstellungen stehen mit (i) im
  Inspektor.
- Screenshot-Flag `-info <id>` zeigt die Erklärung im Fenster.


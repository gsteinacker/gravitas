---
id: T-055
title: Bedienung vollständig in SwiftUI
status: done
milestone: M8
priority: 1
estimate: M
depends: []
---

## Ziel

Alles, was die Ebitengine-App bedienbar machte, soll auch die SwiftUI-App können.

## Akzeptanzkriterien

- [x] Menü: Schnell sichern/letzten Stand laden, Tastenübersicht (Hilfe), alle Effekte, Dunkle Materie zeigen
- [x] Startpfeil mit Geschwindigkeit beim Alt-Ziehen
- [x] Einstellungen: Sternzahl fest/automatisch, GPU, „Leistung neu messen“
- [x] Korrektur der Sternzahl über die gemessene Schrittzeit (wie `cmd/galaxy/tuning.go`) im Go-Kern
- [x] Optional einblendbare Statuszeile im Bild (HUD-Ersatz)

## Notizen

Engine: `SetStars`, `SetGPUMode`, `Recalibrate`, `LaunchSpeed`, Korrektur der Sternzahl über die gemessene
Schrittzeit (`engine/tuning.go`, Port von `cmd/galaxy/tuning.go`). Swift: Schnell sichern/laden (F5/F9),
Tastenübersicht als Sheet (?, F1), Statuszeile im Bild (h), Startpfeil als Layer beim ⌥-Ziehen, Einstellungen mit
Sternzahl, GPU, Messung und „Leistung neu messen“.

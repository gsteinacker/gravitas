---
id: T-042
title: Aktionsschicht und App-Controller
status: done
milestone: M7
priority: 1
estimate: M
depends: []
---

## Ziel

Alle Aktionen (Szene laden, Pause, Tempo, c, Effekte, Werkzeug, Sichern/Öffnen …) stecken heute im Tasten-`switch`
von `internal/ui/game.go`. Eine Aktionsschicht ohne Ebitengine erlaubt, dass Tasten, natives Menü, Inspektor und
später eine SwiftUI-App dieselben Wege benutzen.

## Akzeptanzkriterien

- [x] Paket `internal/app` (ohne Ebitengine-Import) mit `Action`-Konstanten für jede Taste aus der Hilfe
- [x] `Controller` hält Runner, Szene, n, Tempo, Werkzeug, Masse und Effekte; `Do(Action)` und `State()`-Snapshot
- [x] Threadsichere Warteschlange: von beliebigen Goroutinen einreihen, in `Game.Update` ausführen (`make race` grün)
- [x] `internal/ui.Game` übersetzt Tasten nur noch in Aktionen, Tastenbelegung unverändert (`AppendInputChars`)
- [x] Speicherstände liegen unter `~/Library/Application Support/Galaxy/saves` (`os.UserConfigDir`)
- [x] Test je Aktion auf die Zustandsänderung

## Notizen

Grundlage für T-043 bis T-050. Plan: `~/.claude/plans/k-nnen-wir-die-app-ancient-pine.md`.

Umgesetzt: `internal/app` (Action, Controller, State, Lernmodus-Zustand), `internal/view` (Kamera, Effekte aus
`render` herausgelöst). Test `TestNoEbitengine` sichert die Abhängigkeitsgrenze.

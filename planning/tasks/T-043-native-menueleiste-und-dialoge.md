---
id: T-043
title: Native Menüleiste und Dialoge
status: todo
milestone: M7
priority: 1
estimate: L
depends: [T-042]
---

## Ziel

Die App soll sich wie eine Mac-App bedienen lassen: eine vollständige Menüleiste mit allen Aktionen, Häkchen und
Kürzeln sowie native Öffnen-/Sichern-Dialoge statt F5/F9 und Kommandozeilen-Flags.

## Akzeptanzkriterien

- [ ] ADR-0009: cgo/Objective-C in `internal/mac` (Erweiterung von ADR-0004)
- [ ] `internal/mac` mit `darwin && cgo`-Implementierung und No-op-Stub für andere Plattformen
- [ ] Menüs Galaxy, Ablage, Simulation, Physik, Werkzeug, Darstellung, Fenster, Hilfe ersetzen Ebitengines Minimalmenü
- [ ] Häkchen und Radiogruppen spiegeln den Zustand (`validateMenuItem:`), auch nach Änderung per Taste
- [ ] Szene öffnen (JSON), Stand öffnen/sichern (`.galaxy`), Bild exportieren (PNG) über `NSOpenPanel`/`NSSavePanel`
- [ ] Fehler (kaputte Datei) als `NSAlert`
- [ ] Eine Taste löst ihre Aktion genau einmal aus (Menü-Kürzel und Spieltaste nicht doppelt)
- [ ] Screenshot-Modus unverändert, `make check` grün

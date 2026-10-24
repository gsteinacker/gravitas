---
id: T-043
title: Native Menüleiste und Dialoge
status: done
milestone: M7
priority: 1
estimate: L
depends: [T-042]
---

## Ziel

Die App soll sich wie eine Mac-App bedienen lassen: eine vollständige Menüleiste mit allen Aktionen, Häkchen und
Kürzeln sowie native Öffnen-/Sichern-Dialoge statt F5/F9 und Kommandozeilen-Flags.

## Akzeptanzkriterien

- [x] ADR-0009: cgo/Objective-C in `internal/mac` (Erweiterung von ADR-0004)
- [x] `internal/mac` mit `darwin && cgo`-Implementierung und No-op-Stub für andere Plattformen
- [x] Menüs Galaxy, Ablage, Simulation, Physik, Werkzeug, Darstellung, Fenster, Hilfe ersetzen Ebitengines Minimalmenü
- [x] Häkchen und Radiogruppen spiegeln den Zustand (`validateMenuItem:`), auch nach Änderung per Taste
- [x] Szene öffnen (JSON), Stand öffnen/sichern (`.galaxy`), Bild exportieren (PNG) über `NSOpenPanel`/`NSSavePanel`
- [x] Fehler (kaputte Datei) als `NSAlert`
- [x] Eine Taste löst ihre Aktion genau einmal aus (Menü-Kürzel und Spieltaste nicht doppelt)
- [x] Screenshot-Modus unverändert, `make check` grün

## Notizen

Geprüft per `GALAXY_MENU_SELFTEST=1`: Menü vollständig (de), Häkchen und Deaktivierung stimmen, `g`/`x` vom Menü
verbraucht und wirksam, `?`/`+` gehen ans Spiel. Dialog-Ergebnisse (Szene, Stand, Bild, Finder-Doppelklick,
kaputte Datei) per Unit-Test. Sheets und Alert selbst nur von Hand prüfbar (kein Bildschirmzugriff in der Sitzung).

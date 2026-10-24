# Galaxy

Interaktive N-Body-Simulation: Tausende Sterne, Schwarze Löcher per Mausklick, einstellbare Lichtgeschwindigkeit.
Go + Ebitengine, als native Mac-App mit Menüleiste, Inspektor und Einstellungen (Deutsch/Englisch).

## Installieren (macOS 12 oder neuer, Apple Silicon und Intel)

1. `Galaxy-<Version>.dmg` öffnen und **Galaxy** auf **Applications** (Programme) ziehen.
2. Beim ersten Start meldet macOS, dass die App nicht geprüft werden kann. Sie ist (noch) nicht bei Apple
   notarisiert. Einmalig freigeben:
   - **macOS 15 und neuer:** Galaxy starten und die Meldung schließen. Dann unter *Systemeinstellungen →
     Datenschutz & Sicherheit* bei „Galaxy wurde blockiert“ auf **Trotzdem öffnen** klicken.
   - **macOS 12–14:** Im Finder bei gedrückter ctrl-Taste auf Galaxy klicken, dann **Öffnen** wählen.
3. Beim ersten Start misst Galaxy etwa 0,1 s lang die Leistung des Rechners. Daraus wählt es die Sternzahl für
   flüssige 30 Schritte/s. Danach passt es Auflösung und Genauigkeit laufend an. In den *Einstellungen* (⌘,)
   lassen sich das Profil, eine feste Sternzahl oder die Grafikkarte wählen, dort gibt es auch „Leistung neu
   messen“.

Speicherstände (`.galaxy`) öffnen sich per Doppelklick. Einstellungen und Stände liegen unter
`~/Library/Application Support/Galaxy`.

**Bauen:**
- `make app` erzeugt `bin/Galaxy.app` (Universal, ad-hoc signiert).
- `make dmg` erzeugt das Disk-Image.
- Mit einem Apple-Developer-Zertifikat signiert und notarisiert
  `make notarize SIGN_ID="Developer ID Application: …" NOTARY_PROFILE=…`. Das Profil wird vorher einmal mit
  `xcrun notarytool store-credentials` angelegt.

## Entwickeln

```sh
make run                       # Start (Sternzahl nach Leistungsprofil, Spiralgalaxie)
go run ./cmd/galaxy -n 50000 -c 2.5        # kleines c: das zentrale Loch frisst die Galaxie
go run ./cmd/galaxy -preset 5               # Lokale Gruppe in echten Abständen (x/v: Dunkle Materie)
go run ./cmd/galaxy -scene meine.json        # eigene Szene (docs/scenes.md), -load ~/Library/Application\ Support/Galaxy/saves/….galaxy
go run ./cmd/galaxy -preset 4 -c 3         # Doppel-Loch mit Linseneffekt
go run ./cmd/galaxy -preset 2 -fx glow,doppler,trails   # Kollision mit Bahnspuren
go run ./cmd/galaxy -n 100000 -preset 2   # Barnes-Hut (Default), -solver direct für Referenz
make plan                      # Fortschritt im Backlog
make check                     # fmt, vet, tests, Backlog-Validierung
```

Alle Aktionen stehen auch in der Menüleiste und im Inspektor (⌥⌘I).

| Eingabe | Aktion |
|---------|--------|
| `1`–`8` | Szene: Spirale, Kollision, Sternhaufen, BH-Doppel, **Lokale Gruppe, Antennen, Whirlpool, S-Sterne um Sgr A\*** ([Szenen](docs/scenes.md)) |
| `b` `k` `Tab` | Werkzeug: Schwarzes Loch / Sternhaufen |
| Klick / Alt-Ziehen | Objekt platzieren / mit Geschwindigkeit losschicken (`m` `n` ändern die Masse) |
| Ziehen / Shift-Ziehen / Rad | Rotieren / Verschieben / Zoom |
| `<` `>` | Lichtgeschwindigkeit verringern / erhöhen (kleines c → große Horizonte) |
| `Leertaste` `.` `+` `-` `r` | Pause, Einzelschritt, Tempo, Neustart |
| `e` | **Lernmodus**: Einführung zur Szene und Erklärungen (←/→ Thema, ↑/↓ Zielgruppe: Schüler, Interessierte, Physik) |
| `x` `v` | Dunkle Materie: Gravitation an/aus, sichtbar machen |
| `F5` `F9` | Spielstand speichern / neuesten laden (`~/Library/Application Support/Galaxy/saves`) |
| `o` `u` | Gaskarte (Falschfarben) / Staubabsorption |
| `p` | Gravitation der Löcher: instantan / retardiert / retardiert + v (Ausbreitung mit c) |
| `g` `t` `l` `d` | Effekte: Glow, Spuren, Gravitationslinse, Doppler-Farben |
| `h` `?`/`F1` `f` `c` | HUD, Hilfe, Vollbild, Ansicht zentrieren |

Doku: [Stack-Analyse](docs/analysis/01-tech-stack.md) · [Architektur](docs/architecture.md) ·
[Physik](docs/physics.md) · [ADRs](docs/adr) · [Roadmap](planning/ROADMAP.md)

# Galaxy

Interaktive N-Body-Simulation: Tausende Sterne, Schwarze Löcher per Mausklick, einstellbare Lichtgeschwindigkeit.
Go + Ebitengine.

```sh
make run                       # Start (50.000 Sterne, Spiralgalaxie)
go run ./cmd/galaxy -n 50000 -c 2.5        # kleines c: das zentrale Loch frisst die Galaxie
go run ./cmd/galaxy -preset 5               # Lokale Gruppe in echten Abständen (x/v: Dunkle Materie)
go run ./cmd/galaxy -scene meine.json        # eigene Szene (docs/scenes.md), -load saves/….galaxy
go run ./cmd/galaxy -preset 4 -c 3         # Doppel-Loch mit Linseneffekt
go run ./cmd/galaxy -preset 2 -fx glow,doppler,trails   # Kollision mit Bahnspuren
go run ./cmd/galaxy -n 100000 -preset 2   # Barnes-Hut (Default), -solver direct für Referenz
make plan                      # Fortschritt im Backlog
make check                     # fmt, vet, tests, Backlog-Validierung
```

| Eingabe | Aktion |
|---------|--------|
| `1`–`8` | Szene: Spirale, Kollision, Sternhaufen, BH-Doppel, **Lokale Gruppe, Antennen, Whirlpool, S-Sterne um Sgr A\*** ([Szenen](docs/scenes.md)) |
| `b` `k` `Tab` | Werkzeug: Schwarzes Loch / Sternhaufen |
| Klick / Alt-Ziehen | Objekt platzieren / mit Geschwindigkeit losschicken (`m` `n` ändern die Masse) |
| Ziehen / Shift-Ziehen / Rad | Rotieren / Verschieben / Zoom |
| `<` `>` | Lichtgeschwindigkeit verringern / erhöhen (kleines c → große Horizonte) |
| `Leertaste` `.` `+` `-` `r` | Pause, Einzelschritt, Tempo, Neustart |
| `x` `v` | Dunkle Materie: Gravitation an/aus, sichtbar machen |
| `F5` `F9` | Spielstand speichern / neuesten laden (`saves/`) |
| `p` | Gravitation der Löcher: instantan / retardiert / retardiert + v (Ausbreitung mit c) |
| `g` `t` `l` `d` | Effekte: Glow, Spuren, Gravitationslinse, Doppler-Farben |
| `h` `?`/`F1` `f` `c` | HUD, Hilfe, Vollbild, Ansicht zentrieren |

Doku: [Stack-Analyse](docs/analysis/01-tech-stack.md) · [Architektur](docs/architecture.md) ·
[Physik](docs/physics.md) · [ADRs](docs/adr) · [Roadmap](planning/ROADMAP.md)

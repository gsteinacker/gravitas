# Galaxy

Interaktive N-Body-Simulation: Tausende Sterne, Schwarze Löcher per Mausklick, einstellbare Lichtgeschwindigkeit.
Go + Ebitengine.

```sh
make run                       # Start (8000 Sterne, Spiralgalaxie)
go run ./cmd/galaxy -n 12000 -preset 2
make plan                      # Fortschritt im Backlog
make check                     # fmt, vet, tests, Backlog-Validierung
```

| Eingabe | Aktion |
|---------|--------|
| `1`–`4` | Szene: Spirale, Kollision, Sternhaufen, BH-Doppel |
| Klick | Schwarzes Loch platzieren (`[` `]` ändern die Masse) |
| Ziehen / Shift-Ziehen / Rad | Rotieren / Verschieben / Zoom |
| `Leertaste` `+` `-` `r` | Pause, Tempo, Neustart |
| `h` `?` `f` `c` | HUD, Hilfe, Vollbild, Ansicht zentrieren |

Doku: [Stack-Analyse](docs/analysis/01-tech-stack.md) · [Architektur](docs/architecture.md) ·
[Physik](docs/physics.md) · [ADRs](docs/adr) · [Roadmap](planning/ROADMAP.md)

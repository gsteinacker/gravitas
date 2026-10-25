---
id: T-056
title: Kommandozeile und Screenshot-Modus der SwiftUI-App
status: done
milestone: M8
priority: 2
estimate: S
depends: []
---

## Ziel

Die Arbeitsweise im Repo (CLAUDE.md) beruht auf Kommandozeilen-Flags und reproduzierbaren Screenshots. Die
SwiftUI-App braucht dieselben Möglichkeiten, bevor die Go-App gelöscht wird.

## Akzeptanzkriterien

- [x] Argumente wie `cmd/galaxy`: -preset -n -scene -load -c -gravity -fx -explain -lang -scale -adapt
      -hud -shot -shot-after
- [x] Screenshot-Modus reproduzierbar: ohne Einstellungen, Faktor 1, 50 000 Sterne ohne -n, gleiche Messzeile
- [x] `make shot` nutzt die SwiftUI-App

## Notizen

`engine.ParseOptions` (Go `flag`, gleiche Namen wie `cmd/galaxy`, unbekannte macOS-Argumente werden ignoriert, relative
Pfade gegen `$PWD`), `engine.Start`; Swift reicht nur `CommandLine.arguments` an `galaxy_start`. Die Sprache
entscheidet der Go-Kern (`-lang` oder System), Swift übernimmt sie. Messzeile identisch zur Go-App (`ShotLine`).
`-hud`, `-help` neu. `make shot` nutzt `bin/swift/GalaxySwift`.

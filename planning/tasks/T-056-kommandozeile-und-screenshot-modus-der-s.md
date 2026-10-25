---
id: T-056
title: Kommandozeile und Screenshot-Modus der SwiftUI-App
status: todo
milestone: M8
priority: 2
estimate: S
depends: []
---

## Ziel

Die Arbeitsweise im Repo (CLAUDE.md) beruht auf Kommandozeilen-Flags und reproduzierbaren Screenshots. Die
SwiftUI-App braucht dieselben Möglichkeiten, bevor die Go-App gelöscht wird.

## Akzeptanzkriterien

- [ ] Argumente wie `cmd/galaxy`: -preset -n -scene -load -c -gravity -fx -explain -lang -scale -adapt
      -hud -shot -shot-after
- [ ] Screenshot-Modus reproduzierbar: ohne Einstellungen, Faktor 1, 50 000 Sterne ohne -n, gleiche Messzeile
- [ ] `make shot` nutzt die SwiftUI-App

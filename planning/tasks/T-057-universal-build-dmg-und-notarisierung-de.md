---
id: T-057
title: Universal-Build, DMG und Notarisierung der SwiftUI-App
status: todo
milestone: M8
priority: 1
estimate: M
depends: [T-056]
---

## Ziel

Die SwiftUI-App soll sich wie die bisherige verteilen lassen: Universal Binary, DMG, Notarisierung vorbereitet.

## Akzeptanzkriterien

- [ ] Go-Bibliothek und Swift-Binary für arm64 und x86_64, mit `lipo` vereint (ohne Xcode)
- [ ] `make dmg` und `make notarize` für die SwiftUI-App
- [ ] x86_64-Binary läuft unter Rosetta (Screenshot)

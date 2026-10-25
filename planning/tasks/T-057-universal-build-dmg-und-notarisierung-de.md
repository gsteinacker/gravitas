---
id: T-057
title: Universal-Build, DMG und Notarisierung der SwiftUI-App
status: done
milestone: M8
priority: 1
estimate: M
depends: [T-056]
---

## Ziel

Die SwiftUI-App soll sich wie die bisherige verteilen lassen: Universal Binary, DMG, Notarisierung vorbereitet.

## Akzeptanzkriterien

- [x] Go-Bibliothek und Swift-Binary für arm64 und x86_64, mit `lipo` vereint (ohne Xcode)
- [x] `make dmg` und `make notarize` für die SwiftUI-App
- [x] x86_64-Binary läuft unter Rosetta (Screenshot)

## Notizen

`make lib` baut die Go-Bibliothek für arm64 und amd64 (`lipo`), `swift build --arch arm64 --arch x86_64`. DMG über
`make dmg-swift`, Notarisierung `make notarize-swift` (gemeinsame Ziele `package-dmg`/`notarize-dmg`). x86_64 unter
Rosetta geprüft (Screenshot mit Metal). Hinweis von Xcode: x86_64 läuft mit macOS 27 aus; Mindestziel bleibt 14.

---
id: T-050
title: SwiftUI-App mit Metal-Renderer (Branch swiftui-app)
status: todo
milestone: M7
priority: 2
estimate: L
depends: [T-049]
---

## Ziel

Zweite Variante zum Vergleich: eine echte SwiftUI-App mit nativem Inspektor, Settings und Lernmodus-Seitenleiste;
Rendering in Metal statt Ebitengine.

## Akzeptanzkriterien

- [ ] ADR-0010: Swift als zweiter Stack (Versuch auf Branch)
- [ ] Swift Package `macos/GalaxySwift`, `make app-swift` → „Galaxy (SwiftUI).app“, eigene Bundle-ID
- [ ] MTKView: Sterne (additiv), Glow, dann Linse, Doppler, Spuren, Staub; Screenshot-Vergleich mit Variante 1
- [ ] SwiftUI: Menü (`Commands`), `Settings`, `.inspector`, Lernmodus-Seitenleiste, Datei-Import/-Export, de/en
- [ ] Adaptiver Regler aus Go wird genutzt (Swift meldet Frame-Zeit)

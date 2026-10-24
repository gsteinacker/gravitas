---
id: T-050
title: SwiftUI-App mit Metal-Renderer (Branch swiftui-app)
status: done
milestone: M7
priority: 2
estimate: L
depends: [T-049]
---

## Ziel

Zweite Variante zum Vergleich: eine echte SwiftUI-App mit nativem Inspektor, Settings und Lernmodus-Seitenleiste;
Rendering in Metal statt Ebitengine.

## Akzeptanzkriterien

- [x] ADR-0010: Swift als zweiter Stack (Versuch auf Branch)
- [x] Swift Package `macos/GalaxySwift`, `make app-swift` → „Galaxy (SwiftUI).app“, eigene Bundle-ID
- [x] MTKView: Sterne (additiv, HDR), Glow, Linse, Doppler, Spuren, Hα-Emission; Screenshot-Vergleich mit Variante 1. **Staub, DM-/Gaskarte nicht portiert**
- [x] SwiftUI: Menü (`Commands`), `Settings`, `.inspector`, Lernmodus-Seitenleiste, Datei-Import/-Export, de/en
- [x] Adaptiver Regler aus Go wird genutzt (Swift meldet Frame-Zeit)

## Notizen

- SwiftUI-Teile:
  - Menü per `Commands`; Spieltasten über typisierte Zeichen in `SimulationView.keyDown`;
  - Inspektor per `.inspector`, Lernmodus als `NavigationSplitView`-Seitenleiste, `Settings`;
  - `fileImporter` und `NSSavePanel`, Finder-Öffnen über `application(_:open:)`.
- Texte de/en über `L()` statt String Catalogs, die Xcode bräuchten.
- Werkzeug-Grenzen der Command Line Tools: Statt `@State` (Makro-Plugin fehlt) ein `ObservableObject`, Shader
  aus Quelltext, Build nur für arm64.
- Prüfung ohne Bildschirm: `-shot x.png` schreibt das Metal-Bild und `x.window.png` (die SwiftUI-Bedienung);
  `-explain` öffnet die Seitenleiste.

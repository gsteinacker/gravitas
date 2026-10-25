# ADR-0011: SwiftUI als einzige Oberfläche, Go-Kern als Bibliothek

- Status: accepted
- Datum: 2026-10-25
- Tasks: T-052 bis T-058
- Ersetzt: die Oberflächen-Teile von ADR-0001 (Ebitengine), ADR-0008 (Go-Schriften) und ADR-0009 (AppKit-Hülle);
  bestätigt ADR-0010

## Kontext
T-051 hat zwei Mac-Oberflächen verglichen:
- **A:** Ebitengine mit AppKit-Hülle.
- **B:** SwiftUI mit Metal-Renderer auf dem Go-Kern als c-archive.

Der Nutzer hat am 2026-10-25 entschieden, nur noch B weiterzuverfolgen und A zu löschen, sobald B alles kann. Mit
T-052 bis T-057 ist das erreicht: Staub, Schleier, Gaskarte, Bedienung, Kommandozeile, Screenshot-Modus,
Universal-Build und DMG.

## Entscheidung
Galaxy ist eine SwiftUI-App mit einem Go-Kern.

- **Go** (`internal/*`, `cmd/libgalaxy` als c-archive): Physik, Szenen, Controller (`app`), Kalibrierung und
  Regler (`perf`), Lerntexte, Kommandozeile sowie alles Grafikfreie an der Darstellung (`engine.Prepare`):
  Belichtung, Emissionsnormierung, Schleier, Staubkarte, Löcher.
- **Swift** (`macos/GalaxySwift`): Fenster, Menü, Inspektor, Einstellungen, Lern-Seitenleiste, Eingabe. Dazu der
  Metal-Renderer (`Shaders/Galaxy.metal`, vorkompiliert zu `Galaxy.metallib`).
- **Gelöscht:** `cmd/galaxy`, `internal/ui`, `internal/render`, `internal/mac`; Ebitengine fliegt aus `go.mod`.
- **Bundle:** `Galaxy.app` mit der ID `de.steinacker.galaxy` (`make app`/`dmg`/`notarize`).
- **Werkzeuge:** Xcode wird zum Bauen gebraucht (Metal-Compiler, SwiftUI-Makros, Universal-Build).

## Alternativen
- **A behalten:** Sie bietet weniger native Bedienung und rendert in 8 Bit statt HDR; zudem müsste jede
  Darstellungsänderung doppelt gebaut werden (T-051).
- **Rendering ganz in Go mit Metal über cgo:** Das wäre ein weiterer eigener Renderer ohne SwiftUI-Vorteile.

## Konsequenzen
- Nur noch ein Renderer. Darstellungsregeln, die nicht zeichnen, gehören in den Go-Kern, damit sie testbar
  bleiben.
- Screenshots kommen aus der Swift-App (`make shot`, `bin/swift/GalaxySwift -shot …`) mit derselben Messzeile wie
  zuvor.
- Die Glyphenprüfung der Go-Schrift entfällt. SwiftUI nutzt die Systemschrift, die alle Zeichen der Texte hat.
- CI unter Linux prüft weiter den Go-Kern (`make check`). Die Swift-App baut nur auf macOS mit Xcode.

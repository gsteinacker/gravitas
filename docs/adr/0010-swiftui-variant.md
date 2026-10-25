# ADR-0010: SwiftUI-Variante mit Go-Kern als Bibliothek (Versuch auf Branch `swiftui-app`)

- Status: proposed (Vergleichsversuch, Entscheidung in T-051)
- Datum: 2026-10-24
- Tasks: T-049, T-050, T-051

## Kontext
Mit ADR-0009 bekommt die Ebitengine-App eine native AppKit-Hülle: Menü, Dialoge und ein Inspektor, der aus
einer Go-Beschreibung gebaut wird. Offen ist, ob eine echte SwiftUI-App besser zu einer verteilbaren Mac-App
passt, etwa wegen Seitenleisten, `.inspector`, `Settings`, Bedienungshilfen und Fenster-Tabs. Beide Wege sollen
ausprobiert und verglichen werden.

## Entscheidung
Auf dem Branch `swiftui-app` entsteht eine zweite App:
- **Go-Kern als c-archive:** `cmd/libgalaxy` über `internal/engine`, bestehend aus Controller, Runner,
  Kalibrierung und Regler, ohne Ebitengine.
- **Swift Package** `macos/GalaxySwift`:
  - SwiftUI für Menü (`Commands`), `Settings`, `.inspector` und die Lern-Seitenleiste
    (`NavigationSplitView`);
  - `MTKView` mit eigenen Metal-Shadern, als MSL-Quelltext zur Laufzeit kompiliert.
- **Renderer:** Er portiert `internal/render`, also Projektion, Sternfarben, Doppler, Hα-Emission, Spuren,
  Linse, Bloom, Löcher und Auto-Belichtung. Er rendert in float16 (HDR) mit weichem Tonemapping.
- **Bundle:** „Galaxy SwiftUI.app“ (`de.steinacker.galaxy.swiftui`), neben der Hauptvariante installierbar.

Swift ist damit ein zweiter Stack im Repo. Diese Ausnahme gilt nur auf dem Branch, bis T-051 entscheidet.

## Alternativen
- **Ebitengine in einer SwiftUI-View einbetten:** Das geht nicht, weil Ebitengine sein Fenster und seinen
  Run-Loop selbst besitzt.
- **SwiftUI nur für die Fenster um die Ebitengine-App:** Das ergibt zwei Prozesse oder zwei Run-Loops und ist
  keine echte SwiftUI-App.

## Konsequenzen
- Der Renderer existiert doppelt: Kage/Go und MSL/Swift. Jede Darstellungsänderung müsste zweimal gebaut werden.
- Nicht portiert sind Staub, Dunkle-Materie- und Gas-Karte sowie die Gravitationsrotverschiebung im Doppler.
  Ebenfalls fehlen „Leistung neu messen“ und die Korrektur der Sternzahl über die gemessene Schrittzeit.
- **Werkzeuge:** Xcode ist seit 2026-10-25 installiert, samt Metal-Toolchain. Damit stehen zur Verfügung:
  - `@Observable` und `@State`;
  - vorkompilierte Shader (`Galaxy.metallib` aus `Shaders/Galaxy.metal`);
  - ein Universal-Build per `swift build --arch arm64 --arch x86_64`, die Go-Bibliothek per `lipo`.

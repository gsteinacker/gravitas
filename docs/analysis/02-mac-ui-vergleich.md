# Mac-Oberfläche: AppKit-Hülle um Ebitengine gegen SwiftUI-App

Stand 2026-10-24, T-051. Verglichen werden:
- **Variante A** (`main`): Ebitengine-App mit nativer AppKit-Hülle aus Menü, Dialogen, Inspektor-Panel und
  Einstellungen (ADR-0009).
- **Variante B** (Branch `swiftui-app`): SwiftUI-App mit Metal-Renderer auf dem Go-Kern als c-archive (ADR-0010).

Beide nutzen dieselbe Physik, Kalibrierung und denselben Regler.

## Messung

M4 Pro (8P+4E), ohne Fremdlast, Externer Monitor 2560×1440 (Renderfaktor 1). Beide Apps liefen abwechselnd, mit
denselben Startparametern, und wurden nach 20 s gemessen.

| | A: AppKit + Ebitengine | B: SwiftUI + Metal |
|---|---|---|
| Spiralgalaxie 100k | 60 FPS, 81 / 92 / 100 Schritte/s | 60 FPS, 85 / 88 / 107 Schritte/s |
| Kollision 100k | 60 FPS, 76 / 82 Schritte/s | 60 FPS, 86 / 89 Schritte/s |
| Spiralgalaxie 500k | 59–60 FPS, 24–25 Schritte/s | 63–64 FPS, 27–28 Schritte/s (θ 0,95 durch den Regler) |
| Start bis zum ersten Bild (50k, inkl. Kalibrierung) | 0,45–0,47 s | 0,42–0,45 s |
| Bundle | 27 MB universal (arm64 allein 12,8 MB) | 6,1 MB, nur arm64 |

Beide Varianten halten die Bildrate. Die SwiftUI-Variante gewinnt bei der Simulation 5–10 %, weil die Sterne auf
der GPU projiziert und eingefärbt werden. Variante A baut die Sprites auf der CPU, dort konkurriert das mit der
Physik. Bei 500k ist der Vergleich nicht ganz fair, weil der Regler in B θ auf 0,95 angehoben hat.

## Bild

- **A:** 8-Bit-Bild mit Auto-Belichtung und Clipping, warme Farben, dichter Kern sättigt.
- **B:** float16 (HDR) mit weichem Tonemapping. Spiralarme und Kern bleiben besser getrennt; das ist das Ziel von
  T-041, das A noch vor sich hat. Die Farben sind etwas kühler.
- **Nur in A:** Staub (Absorption und Rötung, Silhouette des Pferdekopfs), Karten für Dunkle Materie und Gas,
  Gravitationsrotverschiebung, Startpfeil beim Alt-Ziehen, HUD-Text im Bild.
- **Einmal beobachtet, nicht reproduziert:** ein zusätzlicher Lochring in B während einer Akkretion. Die
  exportierten Daten zeigten dabei nur ein Loch.

## Bedienung

| | A | B |
|---|---|---|
| Menü | vollständig, Häkchen, Spieltasten als Kürzel | vollständig über `Commands`, Toggles und Picker |
| Inspektor | schwebendes HUD-Panel, aus Go-Beschreibung gebaut | in das Fenster integriert (`.inspector`), gruppiertes Formular |
| Lernmodus | Panel im Bild, Go-Schrift, nicht markierbar, keine Bedienungshilfen | Seitenleiste mit Themenliste, Zielgruppen-Umschalter, markierbarer Text, VoiceOver |
| Einstellungen | Fenster aus Go-Beschreibung | Standard-`Settings`-Scene |
| Dateien | `NSOpenPanel`/`NSSavePanel`, Finder-Doppelklick | `fileImporter`, `NSSavePanel`, Finder-Doppelklick |
| Vollbild, Fenster-Tabs, Systemdarstellung | über Ebitengine, eingeschränkt | nativ |
| Sprache | de/en im Code (`i18n.T`) | de/en im Code (`L()`); String Catalogs bräuchten Xcode |

B fühlt sich spürbar mehr wie eine Mac-App an. Seitenleiste, Inspektor und Simulation teilen sich ein Fenster,
Texte sind auswählbar, und alles folgt der Systemdarstellung. A ist vollständiger, wirkt aber wie ein Spiel mit
nativen Anbauten. Der Lernmodus ist der größte Unterschied.

## Aufwand und Wartung

| | A | B |
|---|---|---|
| Neuer Code | `internal/mac` 1 549 Zeilen (Go + Objective-C) + 383 Test | Swift 1 240 Zeilen (davon Renderer und Shader 663) + `engine`/`libgalaxy` 508 |
| Renderer | einer (Go/Kage) | zweiter in MSL; jede Darstellungsänderung doppelt, solange A lebt |
| Werkzeuge | Command Line Tools reichen | CLT gehen mit Einschränkungen: kein `@State`-Makro, kein Shader-Offline-Compiler, kein Universal-Build. Xcode empfohlen |
| Tests | Menü- und Formularmodell, Dialogpfade, Selbsttest der Cocoa-Seite | C-Smoke-Test der Bibliothek; SwiftUI nur über Shot-Modus und Fenster-Schnappschuss |
| Vertrieb | `make app`/`dmg`/`notarize` fertig | `make app-swift`; DMG und Notarisierung analog, aber noch nicht gebaut |

## Empfehlung

**Kurzfristig A ausliefern, mittelfristig auf B umsteigen.**

- **A** ist fertig, vollständig im Funktionsumfang, universal gebaut und paketiert. Damit lässt sich die App sofort
  an andere geben.
- **Für ein Produkt** ist B die bessere Basis: echte Mac-Oberfläche, Bedienungshilfen, HDR-Bild, weniger
  CPU-Last für die Darstellung und ein späterer Weg in den App Store über ein Xcode-Projekt.
- **Für den Umstieg nötig:**
  - Staub sowie DM- und Gaskarte in MSL portieren;
  - die Korrektur der Sternzahl über die gemessene Schrittzeit in `internal/engine` übernehmen;
  - Xcode für Universal-Build, String Catalogs und vorkompilierte Shader.

  Danach entfällt der Ebitengine-Renderer. Physik, Kalibrierung, Regler und Lerntexte bleiben in Go und werden
  von B unverändert genutzt.

Die Entscheidung liegt beim Nutzer. Bis dahin bleibt B auf dem Branch, der von `main` regelmäßig nachgezogen
werden kann.

# ADR-0009: Native macOS-Hülle über cgo und Objective-C (`internal/mac`)

- Status: accepted
- Datum: 2026-10-24
- Tasks: T-043, T-044, T-047, T-048

## Kontext
Galaxy soll an andere weitergegeben werden und sich wie eine Mac-App bedienen lassen: mit Menüleiste, Häkchen,
Öffnen- und Sichern-Dialogen, Warnungen, einem Inspektor und einem Einstellungsfenster. Ebitengine erzeugt nur
ein Minimalmenü (Über, Ausblenden, Beenden, Fenster) und bietet keine nativen Bedienelemente. Für das Rendering
und die Eingabe im Spielfeld bleibt Ebitengine gesetzt (ADR-0001).

ADR-0004 erlaubt cgo bisher nur für Metal in `internal/sim/bh/gpu_darwin.go`.

## Entscheidung
Die Ausnahme aus ADR-0004 wird auf das Paket `internal/mac` erweitert. Dort spricht Objective-C über cgo AppKit
an (Frameworks Cocoa und UniformTypeIdentifiers).

- **Aufteilung:**
  - `mac_darwin.m`: Objective-C; enthält Definitionen, weil eine cgo-Präambel mit `//export` das nicht darf.
  - `mac_darwin.go`: dünne Brücke.
  - `menu.go`, `shell.go`: plattformneutrale Logik, also Menübeschreibung, Häkchen-Zustand und Befehle,
    headless testbar.
  - `mac_other.go`: No-op-Stub für `!(darwin && cgo)`.
- **Threading:**
  - Go ruft AppKit nur per `dispatch_async` auf den Main-Thread auf und wartet nie darauf. Ebitengines Main-Thread
    kann auf den Spielthread warten, ein synchrones Warten könnte also verklemmen.
  - Callbacks von AppKit gehen über `app.Controller.Post` in die Update-Schleife.
  - Häkchen liest `validateMenuItem:` aus dem atomar veröffentlichten `app.State`.
- **Menü:** Go beschreibt es als Zeilenliste, Objective-C baut daraus das `NSMenu`. Neue Einträge brauchen also
  kein Objective-C.
- **Tasten:** Spieltasten ohne Modifier stehen als Key-Equivalent im Menü. AppKit bietet jedes Key-Down zuerst
  dem Menü an; das Menü verbraucht die Taste, oder sie erreicht das Spiel. So wirkt jede Taste genau einmal.
  Zeichen, die je nach Layout Shift brauchen (`? < > +`), und die Leertaste bleiben beim Spiel, das Zeichen
  statt Tastenpositionen liest. Ein Test stellt sicher, dass Menütaste und Spieltaste dieselbe Aktion auslösen.
- **Dokumente aus dem Finder:** `application:openURLs:` wird per `class_addMethod` an Ebitengines App-Delegate
  angehängt, sobald `NSApplicationWillFinishLaunchingNotification` kommt. So gehen auch die Dateien nicht
  verloren, mit denen die App gestartet wurde.
- **Prüfung ohne Bildschirmzugriff:** `GALAXY_MENU_SELFTEST=1` gibt das Menü samt Häkchen aus und spielt
  Key-Equivalents durch.

Der adaptive Regler (T-047) darf θ im Bereich 0,7–1,0 verändern. Das ist keine Änderung des Physikmodells: Der
Bereich liegt innerhalb der gemessenen Fehlerbudgets (perf-log T-028/T-034), und der Standard bleibt 0,9.

## Alternativen
- **purego statt cgo:** Ebitengine nutzt purego ohnehin. Blocks (Completion-Handler der Panels), eigene Klassen
  mit `validateMenuItem:` und Konstanten wie `NSAboutPanelOptionCredits` sind darüber aber umständlich und
  fehleranfällig. cgo ist für Metal ohnehin nötig.
- **Eigene Widgets in Ebitengine:** Sie sehen nicht nativ aus, haben keine Systemdialoge, keine Bedienungshilfen
  und keine Menüsuche.
- **SwiftUI-App mit Go als Bibliothek:** Das wird auf dem Branch `swiftui-app` als Vergleich gebaut (T-049 bis
  T-051), mit eigener Entscheidung danach.

## Konsequenzen
- Ein macOS-Build braucht die Command Line Tools (wie bei ADR-0004). `CGO_ENABLED=0` und andere Plattformen
  bauen weiter, dann ohne native Hülle.
- Das Menü ist sprachabhängig (`internal/i18n`, Systemsprache).
- Native Teile lassen sich nur teilweise automatisch testen: Modell und Zustand per Unit-Test, die Cocoa-Seite
  per Selbsttest. Sheets und Dialoge werden von Hand geprüft.

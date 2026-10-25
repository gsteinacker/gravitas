# Galaxy – interaktive N-Body-Simulation

Spielerische Simulation von 10⁴–10⁵ Sternen mit Schwarzen Löchern und einstellbarer Lichtgeschwindigkeit.
Stack: **Go 1.27** für Physik und Kern (CPU, Barnes-Hut-Baumdurchlauf optional auf Metal), **SwiftUI + Metal** als
Mac-App (`macos/GalaxySwift`), verbunden über den Go-Kern als C-Bibliothek (`cmd/libgalaxy`, ADR-0011). Bauen der
App braucht Xcode.

## Orientierung
- `docs/analysis/01-tech-stack.md`: warum dieser Stack
- `docs/architecture.md`: Paketstruktur, Datenmodell, Threading, Performance-Budget
- `docs/physics.md`: Physikmodell, Einheiten (G=1), Vereinfachungen, Test-Invarianten
- `docs/adr/`: Architekturentscheidungen (Template: `0000-template.md`)
- `planning/ROADMAP.md` + `planning/tasks/`: Meilensteine und Backlog

## Befehle
- `make check`: gofmt-Prüfung, vet, Tests, Backlog-Validierung. **Vor jedem Commit grün.**
- `make test` / `make race` / `make bench`
- `make build`: Go-Bibliothek, Shader und Swift-App nach `bin/swift/GalaxySwift`. `make run ARGS="-preset 2"`
  startet sie. `make app` / `make dmg` bauen `bin/Galaxy.app` und das DMG.
- Der Screenshot-Modus gibt eine Zeile `shot …: bodies= fps= steps_per_s= substeps= gas_substeps= t= profile= auto_stars= gpu=` aus. Damit lassen sich
  Messungen ohne Bildauswertung machen. Vorher prüfen, dass keine andere App läuft (`pgrep -fl Galaxy`).
- Effekte für Screenshots über `-fx glow,lens,doppler,trails` wählen (Tasten gehen im Screenshot-Modus nicht).
- Screenshots sind fest (50 000 Sterne, Faktor 1, ohne Einstellungen). `-adapt` misst wie die App (Kalibrierung,
  Regler), `-scale 2` rendert Retina, `-lang de|en` wählt die Sprache, `-hud`/`-help`/`-explain level:topic` zeigen
  Statuszeile, Tastenübersicht, Lernmodus, `-launch x,y[,preset]` plant einen Start (Anteile der Bildgröße).
  Flags: `internal/engine/options.go`.
- Neben `<png>` entsteht `<png>.window.png` (die SwiftUI-Bedienelemente; das Metal-Bild bleibt darin leer) und bei
  offenem Sheet `<png>.sheet.png`. So lässt sich die Oberfläche ohne Bildschirmzugriff prüfen.
- `make shot`: startet die App, speichert nach einigen Sekunden `bin/shot.png` und beendet sich.
  Direkt: `./bin/swift/GalaxySwift -preset 2 -n 8000 -shot <png> -shot-after 10`. **Visuelle Änderungen immer per
  Screenshot prüfen** (PNG mit dem Read-Tool ansehen), bevor sie als erledigt gelten.
- `make plan` (= Board), `make next`, `go run ./tools/plan <list|show|set|new|check>`

## Arbeitsweise
1. Arbeit immer an einem Task aus `planning/tasks/` ausrichten (`/next-task`). Fehlt einer, mit `/new-task` anlegen.
2. Status pflegen: `go run ./tools/plan set T-XXX doing` beim Start, `done` erst wenn alle Akzeptanzkriterien
   abgehakt sind (Checkboxen in der Task-Datei aktualisieren).
3. Kleine, in sich geschlossene Commits pro Task: `T-XXX: <was>`.
4. Abweichungen vom Physikmodell oder von der Architektur brauchen einen ADR (`/adr`).
5. Performance-relevante Änderungen: vorher/nachher benchmarken (`/bench`), Ergebnis in `docs/perf-log.md`.
   **Vorher `pgrep -fl Galaxy` prüfen:** Eine laufende App belegt mehrere Kerne und halbiert die
   Benchmark-Zahlen. Läuft sie, nur relativ messen (alter Commit per `git worktree` gegen neuen, abwechselnd).
   Benchmarks auch am entwickelten Zustand prüfen (`BenchmarkAccelerationsEvolved`).

## Code-Konventionen
- Code, Bezeichner, Kommentare und Commit-Messages auf **Englisch**. Doku und Tasks auf **Deutsch**.
- Swift-Code (Kommentare, Bezeichner) ebenfalls Englisch. Swift zeichnet und bedient nur. Regeln der Darstellung,
  die nicht zeichnen (Belichtung, Staub, Schleier, Löcher), gehören in den Go-Kern (`engine.Prepare`), mit Tests.
- `internal/sim` und Unterpakete bleiben headless testbar; Go-Pakete haben keine Oberfläche.
- Hot Loops: Struct-of-Arrays, `float64`, keine Allokationen, keine Interfaces pro Körper, keine Maps.
  Mit `testing.AllocsPerRun` oder `b.ReportAllocs()` absichern.
- Parallelität über Chunking auf `runtime.GOMAXPROCS(0)` Worker. Jede parallele Änderung braucht `make race`.
- Physik-Änderungen brauchen einen Invarianten-Test aus `docs/physics.md` (Energie-Drift, Impuls, |v| < c).
- Go: nur Standardbibliothek (plus `x/image` für `tools/icon`). Neue Abhängigkeiten nur per ADR. Ausnahme per
  ADR-0004: Metal über cgo in `internal/sim/bh/gpu_darwin.go`. Jede GPU-Änderung braucht `TestGPUWalkMatchesCPU`.
  Swift: nur Apple-Frameworks.
- Neue Bedienfunktion: `app.Action` (Go), Name in `cmd/libgalaxy` (`actions`), Menü/Inspektor in Swift
  (`GalaxyApp.swift`, `Views.swift`), ggf. Taste (`app.ActionForChar`). Neue C-Funktionen in `cmd/libgalaxy`
  brauchen `make lib-test` grün. Nutzertexte zweisprachig: Go `i18n.T("deutsch", "english")`, Swift `L(…)`; die
  Sprache entscheidet der Go-Kern.
- Simulationseinheiten: G = 1. `c` ist ein Parameter und keine Konstante.
- Szenen sind Daten (`internal/scene/scenes/*.json`, Format in `docs/scenes.md`). Neue Strukturen als JSON
  anlegen, mit Quellen und Annahmen in `note` und physikalischen Einheiten in `info` und `yearsPerUnit`.
- Lernmodus (`internal/learn/topics/*.json`, Szenen-`intro`/`topics`): jedes neue Konzept bekommt ein Thema mit
  drei Stufen (school, lay, physics), physikalisch korrekt, mit Modellgrenzen in der Physik-Stufe.
- Körperarten: Star, BlackHole, DarkMatter, Gas. „Kein Loch“ heißt `!= BlackHole`, **nicht** `== Star`.
- Tastenbelegung über eingegebene Zeichen (`NSEvent.characters` → `galaxy_key`), **nicht** über Tastencodes, die
  US-Positionen sind. Ausnahme: layoutunabhängige Tasten (Leertaste, F-Tasten, Pfeile). Der Nutzer hat ein
  deutsches Layout.

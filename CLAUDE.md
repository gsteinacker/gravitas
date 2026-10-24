# Galaxy – interaktive N-Body-Simulation

Spielerische Simulation von 10⁴–10⁵ Sternen mit Schwarzen Löchern und einstellbarer Lichtgeschwindigkeit.
Stack: **Go 1.27 + Ebitengine v2**, Physik auf der CPU (siehe `docs/adr/0001-go-ebitengine.md`).

## Orientierung
- `docs/analysis/01-tech-stack.md`: warum dieser Stack
- `docs/architecture.md`: Paketstruktur, Datenmodell, Threading, Performance-Budget
- `docs/physics.md`: Physikmodell, Einheiten (G=1), Vereinfachungen, Test-Invarianten
- `docs/adr/`: Architekturentscheidungen (Template: `0000-template.md`)
- `planning/ROADMAP.md` + `planning/tasks/`: Meilensteine und Backlog

## Befehle
- `make check`: gofmt-Prüfung, vet, Tests, Backlog-Validierung. **Vor jedem Commit grün.**
- `make test` / `make race` / `make bench`
- `make run`: startet die App
- Der Screenshot-Modus gibt eine Zeile `shot …: bodies= fps= steps_per_s= substeps= t=` aus. Damit lassen sich
  Messungen ohne Bildauswertung machen. Vorher prüfen, dass keine andere App läuft (`pgrep -fl bin/galaxy`).
- Effekte für Screenshots über `-fx glow,lens,doppler,trails` wählen (Tasten gehen im Screenshot-Modus nicht).
- `make shot`: startet die App, speichert nach einigen Sekunden `bin/shot.png` und beendet sich.
  Direkt: `./bin/galaxy -preset 2 -n 8000 -shot <png> -shot-after 10`. **Visuelle Änderungen immer per
  Screenshot prüfen** (PNG mit dem Read-Tool ansehen), bevor sie als erledigt gelten.
- `make plan` (= Board), `make next`, `go run ./tools/plan <list|show|set|new|check>`

## Arbeitsweise
1. Arbeit immer an einem Task aus `planning/tasks/` ausrichten (`/next-task`). Fehlt einer, mit `/new-task` anlegen.
2. Status pflegen: `go run ./tools/plan set T-XXX doing` beim Start, `done` erst wenn alle Akzeptanzkriterien
   abgehakt sind (Checkboxen in der Task-Datei aktualisieren).
3. Kleine, in sich geschlossene Commits pro Task: `T-XXX: <was>`.
4. Abweichungen vom Physikmodell oder von der Architektur brauchen einen ADR (`/adr`).
5. Performance-relevante Änderungen: vorher/nachher benchmarken (`/bench`), Ergebnis in `docs/perf-log.md`.
   **Vorher `pgrep -fl bin/galaxy` prüfen:** Eine laufende App belegt mehrere Kerne und halbiert die
   Benchmark-Zahlen. Läuft sie, nur relativ messen (alter Commit per `git worktree` gegen neuen, abwechselnd).
   Benchmarks auch am entwickelten Zustand prüfen (`BenchmarkAccelerationsEvolved`).

## Code-Konventionen
- Code, Bezeichner, Kommentare und Commit-Messages auf **Englisch**. Doku und Tasks auf **Deutsch**.
- `internal/sim` und Unterpakete importieren **niemals** Ebitengine. Physik bleibt headless testbar.
- Hot Loops: Struct-of-Arrays, `float64`, keine Allokationen, keine Interfaces pro Körper, keine Maps.
  Mit `testing.AllocsPerRun` oder `b.ReportAllocs()` absichern.
- Parallelität über Chunking auf `runtime.GOMAXPROCS(0)` Worker. Jede parallele Änderung braucht `make race`.
- Physik-Änderungen brauchen einen Invarianten-Test aus `docs/physics.md` (Energie-Drift, Impuls, |v| < c).
- Nur Standardbibliothek plus Ebitengine. Neue Abhängigkeiten nur per ADR. Ausnahme per ADR-0004: Metal über
  cgo in `internal/sim/bh/gpu_darwin.go`. Jede GPU-Änderung braucht `TestGPUWalkMatchesCPU`. Per ADR-0009:
  AppKit über cgo in `internal/mac` (Objective-C in `mac_darwin.m`, Logik plattformneutral in Go). AppKit nur per
  `dispatch_async` aufrufen, nie synchron warten. Prüfung ohne Bildschirm: `GALAXY_MENU_SELFTEST=1 ./bin/galaxy`.
- Aktionen laufen über `internal/app` (ohne Ebitengine): neue Bedienfunktion = neue `app.Action`, im Menü
  (`internal/mac/menu.go`) und ggf. als Taste. Nutzertexte zweisprachig mit `i18n.T("deutsch", "english")`.
- Simulationseinheiten: G = 1. `c` ist ein Parameter und keine Konstante.
- Szenen sind Daten (`internal/scene/scenes/*.json`, Format in `docs/scenes.md`). Neue Strukturen als JSON
  anlegen, mit Quellen und Annahmen in `note` und physikalischen Einheiten in `info` und `yearsPerUnit`.
- Lernmodus (`internal/learn/topics/*.json`, Szenen-`intro`/`topics`): jedes neue Konzept bekommt ein Thema mit
  drei Stufen (school, lay, physics), physikalisch korrekt, mit Modellgrenzen in der Physik-Stufe. Nur Zeichen,
  die die Go-Schrift hat (Test `TestLearningTextsUseOnlyAvailableGlyphs`).
- Körperarten: Star, BlackHole, DarkMatter, Gas. „Kein Loch“ heißt `!= BlackHole`, **nicht** `== Star`.
- Tastenbelegung über `ebiten.AppendInputChars` (eingegebene Zeichen), **nicht** über `ebiten.Key*`-Konstanten,
  die US-Positionen sind. Ausnahme: layoutunabhängige Tasten (Leertaste, F-Tasten, Pfeile). Der Nutzer hat ein
  deutsches Layout.

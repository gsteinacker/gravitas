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
- Nur Standardbibliothek plus Ebitengine. Neue Abhängigkeiten nur per ADR.
- Simulationseinheiten: G = 1. `c` ist ein Parameter und keine Konstante.
- Tastenbelegung über `ebiten.AppendInputChars` (eingegebene Zeichen), **nicht** über `ebiten.Key*`-Konstanten,
  die US-Positionen sind. Ausnahme: layoutunabhängige Tasten (Leertaste, F-Tasten, Pfeile). Der Nutzer hat ein
  deutsches Layout.

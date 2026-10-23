# Architektur

> Lebendes Dokument. Beschreibt den **Soll-Zustand**. Abweichungen werden hier oder per ADR nachgezogen.

## Paketstruktur (geplant)

```
cmd/galaxy/          Einstiegspunkt der interaktiven App (Ebitengine)
internal/vec/        Vec3, minimal, inline-freundlich
internal/sim/        Zustand (Struct-of-Arrays), Integrator, Diagnose, Solver-Interface
internal/sim/direct/ Referenzlöser O(N²) (Tests, kleine N)
internal/sim/bh/     Barnes-Hut-Octree, parallel
internal/relativity/ Eigengeschwindigkeit, Horizont, Paczyński-Wiita
internal/scene/      Anfangsbedingungen: Plummer-Kugel, Scheibengalaxie, Kollision, Presets
internal/render/     Kamera, Punkt-Batching, Shader (Kage)
internal/ui/         Eingabe, Werkzeuge (BH platzieren, Zeit steuern), HUD
tools/plan/          Projektplanungs-CLI (Backlog in planning/)
```

Abhängigkeiten zeigen nur nach unten: `cmd → ui → render, scene → sim → vec`.
**`sim` kennt Ebitengine nicht.** Die Physik ist headless test- und benchmarkbar.

## Kern-Datenmodell

```go
// Struct-of-Arrays: cache-freundlich, allokationsfrei im Hot Loop.
type State struct {
    N          int
    PosX, PosY, PosZ []float64
    VelX, VelY, VelZ []float64 // bzw. Eigengeschwindigkeit u = γv ab M3
    AccX, AccY, AccZ []float64
    Mass       []float64
    Kind       []Kind // Star, BlackHole
    Alive      []bool // absorbierte Körper werden periodisch kompaktiert
}

type Solver interface {
    // Accelerations füllt s.Acc* für alle lebenden Körper.
    Accelerations(s *State, p Params)
}
```

## Threading

- Die Simulation läuft in einer eigenen Goroutine mit fester Schrittweite (`sim.Runner`), das Rendering im
  Ebitengine-Loop.
- Übergabe per **Double-Buffer-Snapshot** (`float32`-Positionen). Gelockt wird nur beim Tausch der Buffer
  und während der Renderer liest, nie im Hot Loop.
- Alle Mutationen (Szene laden, BH platzieren) laufen als Closures über `Runner.Do` und werden zwischen zwei
  Schritten auf der Sim-Goroutine ausgeführt. Kein geteilter, veränderlicher Zustand.
- Kraftberechnung: Körper in Chunks über `GOMAXPROCS` Worker verteilt; Baumaufbau sequenziell oder parallel
  nach Oktanten.

## Performance-Budget (Ziel M2, M4 Pro, N = 100k)

| Phase | Budget |
|-------|--------|
| Baumaufbau | ≤ 8 ms |
| Kräfte (parallel) | ≤ 20 ms |
| Integration | ≤ 2 ms |
| Render-Upload | ≤ 4 ms |

Benchmarks liegen neben dem Code (`*_bench_test.go`), Ergebnisse in [perf-log.md](perf-log.md).

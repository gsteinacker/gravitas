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
- Übergabe per **Triple-Buffer-Snapshot** (`float32`-Positionen): Die Simulation füllt `writing`, der
  Renderer liest `reading`, und `latest` ist der neueste fertige Stand. Das Lock schützt nur Zeigertausche,
  keine Seite wartet je auf die Arbeit der anderen (T-024).
- Alle Mutationen (Szene laden, BH platzieren) laufen als Closures über `Runner.Do` und werden zwischen zwei
  Schritten auf der Sim-Goroutine ausgeführt. Kein geteilter, veränderlicher Zustand.
- Kraftberechnung: Körper in Chunks über `GOMAXPROCS` Worker verteilt; Baumaufbau sequenziell oder parallel
  nach Oktanten.

## Performance-Budget (M4 Pro, N = 100k, θ = 0,6)

| Phase | Budget | gemessen (2026-10-23) | Benchmark |
|-------|--------|-----------------------|-----------|
| Baumaufbau | ≤ 8 ms | 3,3 ms ✓ | `bh.BenchmarkBuild` |
| Kräfte, Anfangszustand | ≤ 20 ms | 17,5 ms inkl. Build ✓ | `bh.BenchmarkAccelerations/N=100000` |
| Kräfte, entwickelte Galaxie (t ≈ 2,4) | ≤ 33 ms (30 Schritte/s) | 35,1 ms ✗ knapp | `bh.BenchmarkAccelerationsEvolved` |
| Energie-Diagnose (alle 2 s) | ≤ 5 % | 50 ms / 2 s = 2,5 % ✓ | `bh.BenchmarkPotentialEnergy` |
| Integration | ≤ 2 ms | < 1 % im Profil ✓ | – |
| Rendern (Ebitengine-Thread) | ≤ 16 ms | ~60 FPS ✓ | Profil der App |

**Wichtig:** Die Kosten wachsen, während sich die Galaxie entwickelt, weil ihr Kern dichter wird
(t = 0: 19 ms, t = 2,4: 36 ms). Performance-Aussagen immer auch am entwickelten Zustand prüfen.
Nächster Hebel: Gruppen-Walk (T-027).

Benchmarks liegen neben dem Code (`*_bench_test.go`), Ergebnisse in [perf-log.md](perf-log.md).

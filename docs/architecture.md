# Architektur

> Lebendes Dokument. Beschreibt den **Soll-Zustand**. Abweichungen werden hier oder per ADR nachgezogen.

## Paketstruktur (geplant)

```
cmd/galaxy/          Einstiegspunkt der interaktiven App (Ebitengine)
internal/vec/        Vec3, minimal, inline-freundlich
internal/sim/        Zustand (Struct-of-Arrays), Integrator, Diagnose, Solver-Interface
internal/sim/direct/ Referenzlöser O(N²) (Tests, kleine N)
internal/sim/bh/     Barnes-Hut-Octree, parallel
internal/sim/        … inkl. relativity.go (u = γv) und accrete.go (Horizont, Verschmelzung).
                     Kein eigenes Paket: Drift und Kick sind Hot Loops und gehören zum Integrator.
internal/scene/      Anfangsbedingungen: Plummer-Kugel, Scheibengalaxie, Kollision, Presets
internal/render/     Kamera, Punkt-Batching, Shader (Kage)
internal/ui/         Eingabe, Werkzeuge (BH platzieren, Zeit steuern), HUD
tools/plan/          Projektplanungs-CLI (Backlog in planning/)
```

Abhängigkeiten zeigen nur nach unten: `cmd → ui → render, scene → sim → vec`.
**`sim` kennt Ebitengine nicht.** Die Physik ist headless test- und benchmarkbar.

## Rendering (M4)

`render.Pipeline` pro Frame: Sterne additiv in ein Offscreen-Bild (Farbe nach Masse, optional Doppler-Verschiebung)
→ optional Spuren (max-Blending mit verblassendem Vorbild) → über opaken Sternhimmel → optional
Gravitationslinse (Kage-Shader, Punktlinse) → Bildschirm → optional Bloom (Downsample, 2× Gauß, additiv)
→ Löcher, Ringe, HUD. Effekte per Taste (`g t l d`) oder `-fx`. Shader-Ausgaben verwenden `BlendCopy`,
der Hintergrund ist opak. Sonst akkumulieren Bilder über Frames.

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

// Solver liefert nur Stern-Stern-Kräfte. Alles mit Schwarzen Löchern übernimmt der Integrator
// (sim/split.go: direkt summiert, mit Unterschritten nahe den Löchern).
type Solver interface {
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

## Performance-Budget (M4 Pro, N = 100k, Quadrupol, θ = 0,9)

| Phase | Budget | gemessen (2026-10-23, T-028) | Benchmark |
|-------|--------|-----------------------|-----------|
| Baumaufbau | ≤ 8 ms | 3,3 ms ✓ | `bh.BenchmarkBuild` |
| Kräfte, Anfangszustand | ≤ 20 ms | 13,4 ms inkl. Build ✓ | `bh.BenchmarkAccelerations/N=100000` |
| Kräfte, entwickelte Galaxie (t ≈ 2,4) | ≤ 33 ms (30 Schritte/s) | 20,7 ms ✓ | `bh.BenchmarkAccelerationsEvolved` |
| Ganzer Schritt inkl. Loch-Unterschritte | ≤ 33 ms | 28 ms ✓ | `bh.BenchmarkStep` |
| Energie-Diagnose (alle 2 s) | ≤ 5 % | 20 ms / 2 s = 1 % ✓ | `bh.BenchmarkPotentialEnergy` |
| Integration | ≤ 2 ms | < 1 % im Profil ✓ | – |
| Rendern (Ebitengine-Thread) | ≤ 16 ms | ~60 FPS ✓ | Profil der App |

**Wichtig:** Die Kosten wachsen, während sich die Galaxie entwickelt, weil ihr Kern dichter wird
(t = 0: 13 ms, t = 2,4: 21 ms). Performance-Aussagen immer auch am entwickelten Zustand prüfen.
Die Laufzeit ist proportional zu den Wechselwirkungen pro Stern (T-027). Der Gruppen-Walk war deshalb langsamer,
Quadrupole (T-028) haben geholfen.

Benchmarks liegen neben dem Code (`*_bench_test.go`), Ergebnisse in [perf-log.md](perf-log.md).

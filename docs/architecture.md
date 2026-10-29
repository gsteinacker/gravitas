# Architektur

> Lebendes Dokument. Beschreibt den **Soll-Zustand**. Abweichungen werden hier oder per ADR nachgezogen.

## Paketstruktur

```
cmd/libgalaxy/       Der Go-Kern als C-Bibliothek (c-archive) für die App: Start, Aktionen, Zustand, Bilddaten
internal/engine/     Kern ohne Oberfläche: Start mit Kommandozeile, Regler, Kalibrierung, Bildstatistik
                     (Belichtung, Emission, Schleier, Staubkarte, Löcher: engine.Prepare)
internal/app/        Aktionen und Controller: Zustand der Bedienung, Szenen, Speichern, Lernmodus
internal/perf/       Hardware, Kalibrierung, Leistungsprofile, Laufzeit-Regler
internal/view/       Kamera und Effektauswahl
internal/prefs/      Einstellungen (JSON in Application Support)
internal/i18n/       Deutsch/Englisch
internal/learn/      Lerntexte (topics/*.json)
internal/scene/      Anfangsbedingungen und Szenen (scenes/*.json)
internal/save/       Speicherstände (.galaxy)
internal/sim/        Zustand (Struct-of-Arrays), Integrator, Relativität, Akkretion, Diagnose
internal/sim/direct/ Referenzlöser O(N²) (Tests, kleine N)
internal/sim/bh/     Barnes-Hut-Octree, parallel, Baumdurchlauf optional auf Metal
internal/sph/        Gas (SPH)
internal/vec/        Vec3
macos/GalaxySwift/   SwiftUI-App: Fenster, Menü, Inspektor, Einstellungen, Lern-Seitenleiste, Eingabe;
                     Metal-Renderer (Shaders/Galaxy.metal)
tools/               plan (Backlog), evolve (Szenen vorrechnen), icon (App-Icon), gpubench
```

Abhängigkeiten zeigen nur nach unten: `libgalaxy → engine → app, perf, view → scene → sim → vec`. Kein Go-Paket
hat eine Oberfläche (ADR-0011). Die Swift-App ruft den Kern nur aus dem Main-Thread auf: einmal pro Bild
`galaxy_tick` (führt eingereihte Arbeit aus, regelt, veröffentlicht den Zustand), dann `galaxy_frame` (kopiert den
neuesten Stand in einen Metal-Puffer) und `galaxy_prepare` (Bildstatistik, Staubabschwächung pro Körper, Löcher).
Die Physik läuft in eigenen Goroutinen und ist headless test- und benchmarkbar.

## GPU (ADR-0004)

Auf macOS führt Metal den Baumdurchlauf aus (`bh.Solver.GPU`, Standard in der App). Der Baumaufbau bleibt auf
der CPU und ist vollständig parallelisiert: Sammeln, Morton-Keys, stabiler paralleler Radix-Sort,
sortierte Kopien und die Knoten (Teilbäume parallel, T-029). Das Ergebnis ist bitidentisch zum seriellen
Aufbau.

## Rendering

Metal in der Swift-App, in float16 (HDR) mit weichem Tonemapping am Ende. Pro Bild:
1. Staubbild aus den Gas-Sprites;
2. Sterne und Hα-Emission additiv (Farbe nach Masse, Doppler und Gravitationsrotverschiebung, Abschwächung durch
   Staub pro Körper aus dem Go-Kern);
3. optional Spuren;
4. über den Sternhimmel, der durch das Staubbild abgeschwächt wird;
5. optional Gravitationslinse;
6. optional Bloom;
7. Tonemapping und Löcher.

Die Belichtung ist automatisch (`engine.Prepare`): Das Licht wird in einem 96×60-Raster gesammelt, und die hellsten
10 % der belegten Zellen erreichen den Zielwert, geglättet über die Bilder. Effekte per Menü, Inspektor, Taste oder
`-fx`.

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
  Draw-Callback der `MTKView` auf dem Main-Thread der Swift-App, der den Kern per `galaxy_tick`/`galaxy_frame`
  aufruft.
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
| Rendern (Metal) | ≤ 16 ms | ~60 FPS ✓ | Profil der App |

**Wichtig:** Die Kosten wachsen, während sich die Galaxie entwickelt, weil ihr Kern dichter wird
(t = 0: 13 ms, t = 2,4: 21 ms). Performance-Aussagen immer auch am entwickelten Zustand prüfen.
Die Laufzeit ist proportional zu den Wechselwirkungen pro Stern (T-027). Der Gruppen-Walk war deshalb langsamer,
Quadrupole (T-028) haben geholfen.

Benchmarks liegen neben dem Code (`*_bench_test.go`), Ergebnisse in [perf-log.md](perf-log.md).

## Anpassung an den Rechner (T-046, T-047)

Die App soll auf jedem Mac flüssig laufen, nicht nur auf dem M4 Pro, auf dem das Budget oben gemessen ist.

- **Erkennung** (`perf.Probe`): CPU-Name, P- und E-Kerne, Speicher (sysctl), Metal-Gerät.
- **Kalibrierung** (`perf.Calibrate`, 50–150 ms beim ersten Start, gecacht pro Hardware in den Einstellungen):
  - eine Kraftauswertung der Standardszene mit 20k Körpern, auf CPU und GPU; der schnellere Pfad gewinnt;
  - eine SPH-Auswertung der Gasszene mit 10k Teilchen.
- **Hochrechnung** auf eine Sternzahl: Kosten ≈ 2,1 · k·M log M + 1,6 · k_g·G log G, mit M Körpern und G Gasteilchen.
  - Der Faktor 2,1 ist das Verhältnis „voller Schritt im entwickelten Zustand“ zu „Kraftauswertung im
    Anfangszustand“ (Tabelle oben).
  - Der Faktor 1,6 entspricht den SPH-Auswertungen pro Schritt (perf-log T-038).
  - Die Sternzahl wird pro Szene aus ihrem Gasanteil bestimmt.
- **Profile** in den Einstellungen:

  | Profil | Ziel | Renderauflösung |
  |---|---|---|
  | Automatisch (Standard) | 30 Schritte/s | voll |
  | Niedrig | 60 Schritte/s | ohne Retina |
  | Mittel | 45 Schritte/s | mittel |
  | Hoch | 30 Schritte/s | voll |
  | Maximal | 12 Schritte/s | voll |

  Eine feste Sternzahl in den Einstellungen oder `-n` überschreibt das Profil. Screenshots ohne `-n` nutzen
  50 000 und ignorieren die Einstellungen, damit Messungen reproduzierbar bleiben.
- **Grenze des Modells:** Kurze Gasschritte wachsen mit der Auflösung des Gases. Seit T-085 (ADR-0018) zahlen
  dafür nur die betroffenen Gasteilchen, nicht mehr das ganze Gas und die Gravitation.
  Der Laufzeit-Regler (T-047) misst die echte Schrittzeit und korrigiert damit die Sternzahl der nächsten
  Szene.

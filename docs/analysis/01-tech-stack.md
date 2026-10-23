# Analyse: Projekt-Setup für die Sternensystem-Simulation

Stand: 2026-10-23 · Ergebnis: [ADR-0001](../adr/0001-go-ebitengine.md)

## 1. Anforderungen

| # | Anforderung | Konsequenz für den Stack |
|---|-------------|--------------------------|
| A1 | Tausende bis möglichst viele Massepunkte | Rechenleistung pro Schritt ist der Engpass. Der **Algorithmus** zählt mehr als die Sprache |
| A2 | Grafische Echtzeit-Darstellung | Rendering von 10⁵+ Punkten pro Frame, gebündelte Draw-Calls |
| A3 | Spaß, Interaktivität („Schwarzes Loch platzieren“) | Schnelle Iteration, Maus/Tastatur, sofortiges Feedback, 30–60 FPS |
| A4 | Vereinfachte Physik, aber endliche Lichtgeschwindigkeit | Physikmodell mit einstellbarem `c` (siehe [physics.md](../physics.md)) |
| A5 | Entwicklung mit AI-Harness | Sprache mit schnellem Compile/Test-Zyklus, guten Standard-Tools und wenig „Magie“ |

## 2. Was kostet die Physik?

Ein Paar-Wechselwirkungsterm kostet ca. 20 FLOPs (Differenz, r², Softening, 1/r³, Akkumulation).

| Verfahren | N = 10⁴ | N = 10⁵ | N = 10⁶ |
|-----------|---------|---------|---------|
| Direkte Summe O(N²) | 10⁸ Paare | 10¹⁰ | 10¹² |
| Barnes-Hut O(N log N), θ≈0,7 | ~10⁷ | ~1,5·10⁸ | ~2·10⁹ |

Grobe Durchsätze auf dem Entwicklungsrechner (M4 Pro):

- **CPU, skalar, 12 Kerne:** ca. 1–3·10⁹ Paare/s. Direkte Summe reicht für 10⁴ bei ~20 Schritten/s,
  Barnes-Hut für **10⁵ bei interaktiven Raten** und 10⁶ als „Offline“-Modus (~1 s/Schritt).
- **GPU (Compute-Shader):** ca. 10¹¹ Paare/s. Damit wird schon die direkte Summe bei 10⁵ interaktiv.

Daraus folgt: Mit einer CPU-Lösung ist **~100.000 Sterne interaktiv** ein realistisches Ziel. Will man
darüber hinaus, braucht es GPU-Compute. Das ist die zentrale Weiche der Analyse.

## 3. Kandidaten

### Go + Ebitengine (CPU-Physik)
- **+** Goroutinen machen die Parallelisierung der Kraftberechnung trivial (12 Kerne nutzen ist eine Schleife).
- **+** Sehr schneller Build/Test-Zyklus, `go test -bench`, `pprof`, `go vet`, Race-Detector sind Standard.
  Das ist ideal für einen Agenten-Loop: ändern → testen → benchmarken → profilen.
- **+** Einfache, explizite Sprache. AI-generierter Code ist gut lesbar und reviewbar.
- **+** Ebitengine: ausgereifte 2D-Engine, `DrawTriangles32` für gebündeltes Rendering von 10⁵+ Punkten,
  Kage-Shader (Glow, Gravitationslinsen-Effekt), Builds für macOS/Linux/Windows **und WebAssembly**
  (Szenen im Browser teilen).
- **−** GPU-Compute nur über junge Bibliotheken ([Cogent Core `gpu`/`gosl`](https://www.cogentcore.org/lab/gosl):
  Go→WGSL über WebGPU, oder [`gogpu/wgpu`](https://pkg.go.dev/github.com/gogpu/wgpu@v0.21.2): reines Go). Physik startet auf der CPU.
- **−** SIMD nur experimentell und nicht für arm64. Wir bleiben bei skalarem Code mit cache-freundlichem
  Speicherlayout (Struct-of-Arrays).
- **−** GC: kein Problem, wenn die Hot Loops allokationsfrei sind (vorallokierte Slices, Node-Pools für den Baum).

### Rust + wgpu
- **+** Höchste Performance-Obergrenze (SIMD, GPU-Compute, kein GC).
- **−** Langsamere Iteration (Compile-Zeiten, Borrow-Checker bei Baumstrukturen), höherer Aufwand pro Feature.
  Für ein Spaßprojekt ist der Overhead spürbar.

### TypeScript + WebGPU (Browser)
- **+** GPU-Compute verfügbar, kein Install, leicht teilbar.
- **−** Barnes-Hut auf der CPU in JS ist deutlich langsamer. Ohne GPU bleibt man bei kleinen N.
  Zwei Welten (WGSL + TS), schwieriger zu testen.

### C++ / Zig
- **+** Maximale Kontrolle, SIMD.
- **−** Viel Infrastruktur (Build, Fenster, Abhängigkeiten) bevor es Spaß macht. Speichersicherheit hilft
  einem Agenten-Loop nicht gerade.

### Python (NumPy/Taichi/JAX)
- **+** Super für Prototypen, Taichi kann GPU.
- **−** Interaktive App und Distribution sind mühsam, Performance hängt an der Bibliothek.

## 4. Bewertung

Gewichtung: Spaß/Iterationsgeschwindigkeit hoch, maximale N mittel, Teilbarkeit niedrig.

| Kriterium (Gewicht) | Go+Ebiten | Rust+wgpu | TS+WebGPU | C++ | Python |
|---------------------|:---------:|:---------:|:---------:|:---:|:------:|
| Iterationsgeschwindigkeit (3) | 5 | 2 | 4 | 2 | 4 |
| Max. N interaktiv (2) | 3 | 5 | 4 | 5 | 3 |
| Tooling/Testbarkeit für AI-Loop (3) | 5 | 4 | 3 | 2 | 3 |
| Rendering/Interaktion (2) | 4 | 4 | 4 | 3 | 2 |
| Teilbarkeit (1) | 4 (WASM) | 3 | 5 | 2 | 1 |
| **Summe** | **46** | 38 | 40 | 29 | 30 |

## 5. Empfehlung

**Go + Ebitengine, Physik auf der CPU mit parallelem Barnes-Hut, Struct-of-Arrays in `float64`.**

- Zielgröße: **100.000 Sterne bei ≥ 30 Schritten/s** auf dem M4 Pro, 10⁶ als Offline-Modus.
- Die Physik bleibt hinter einem `Solver`-Interface. Eine spätere GPU-Variante wäre so ein Austausch des
  Backends und kein Neubau. Ein Spike dazu ist im Backlog (Meilenstein M5).
- Physik in **3D** mit Projektion auf eine rotierbare 2D-Kamera. Scheibengalaxien sehen gekippt toll aus,
  und die Mechanik ist dieselbe wie in 2D. Siehe [ADR-0002](../adr/0002-3d-physics-2d-rendering.md).

### Re-Evaluierungs-Trigger
Wir überdenken die Entscheidung, wenn nach M2 (Skalierung) gilt:
- 100k Sterne erreichen trotz Profiling < 15 Schritte/s, **oder**
- der Wunsch nach ≥ 10⁶ interaktiven Sternen wird konkret.

## 6. Nachprüfung: Wäre Faktor 10 (≈ 1 Mio. Sterne interaktiv) mit anderer Sprache drin?

Gemessen am 2026-10-23 auf dem M4 Pro (16 GPU-Kerne):

| Messung | Ergebnis |
|---------|----------|
| Go, CPU, 12 Kerne, float64, direkte Summe (`make bench`) | **6,1 Mrd. Paare/s** |
| Metal-Compute, GPU, float32, direkte Summe, getiled (`tools/gpubench`) | **185 Mrd. Paare/s** (Faktor ~30) |

| Ansatz | 100k Sterne | 1 Mio. Sterne | Faktor 10 erreichbar? |
|--------|-------------|---------------|-----------------------|
| Go CPU, Barnes-Hut (Schätzung, T-008) | ~25–40 ms | ~0,3–0,5 s | – (Referenz) |
| Rust/C++ CPU mit NEON-SIMD, Barnes-Hut | ~10–15 ms | ~0,1–0,2 s | **nein**, nur ~2–3× |
| GPU, direkte Summe (gemessen) | 54 ms | 5,3 s | **nein**, O(N²) frisst den Vorsprung |
| GPU, Baumcode (Bonsai-artig, Schätzung) | ~5 ms | ~40–100 ms | **knapp ja**, 10–25 Schritte/s |

Schätzung GPU-Baumcode: Bonsai erreichte 2012 ~2,8 Mio. Teilchen/s auf einer GPU mit ~1/3 der heutigen
Rechenleistung ([Bédorf et al.](https://ar5iv.arxiv.org/html/1204.2280)). Hochgerechnet ergibt das ~8 Mio./s,
mit gröberem Öffnungswinkel (θ ≈ 0,75, für „Spaß“ ausreichend) eher 15–25 Mio./s.

**Fazit:**
1. Ein **Sprachwechsel allein bringt keinen Faktor 10.** Rust/C++ holen auf der CPU über SIMD etwa 2–3× heraus.
2. Faktor 10 geht nur mit **GPU *und* O(N log N)-Verfahren auf der GPU** (Baumaufbau über Morton-Sortierung
   und Baumtraversierung im Shader). Das ist der aufwendigste Teil des ganzen Projekts, egal in welcher Sprache.
3. Das ist eine Frage des **Solver-Backends, nicht der Sprache**. Aus Go ist WebGPU-Compute erreichbar (siehe oben).
   Das `Solver`-Interface ist dafür vorbereitet.
4. Konsequenz: Go bleibt. Der GPU-Spike (T-023) wird vorgezogen und direkt nach Barnes-Hut auf der CPU (T-008)
   bearbeitet, damit früh klar ist, ob die Go-WebGPU-Bindings tragen. Wenn nicht: GPU-Solver als
   Metal-Shader über purego/cgo, nur für macOS.

# ADR-0004: Baumdurchlauf auf der GPU (Metal über cgo), Baumaufbau auf der CPU

- Status: accepted
- Datum: 2026-10-23
- Tasks: T-023 (Spike), Vorarbeit T-028

## Kontext
ADR-0001 hat die Physik bewusst auf der CPU gestartet. Faktor 10 (≈ 1 Mio. Sterne interaktiv) erfordert laut
Analyse (01-tech-stack.md, Abschnitt 6) die GPU *und* ein O(N log N)-Verfahren. Die reine GPU-Direktsumme
(185 Mrd. Paare/s gemessen) ist inzwischen langsamer als der CPU-Baum (100k: 54 ms gegen 13 ms).

## Entscheidung
Hybrid: Die CPU baut den Octree (Morton-Sortierung, Quadrupole). Die GPU führt den Baumdurchlauf aus,
einen Thread pro Stern. Die vorhandene Skip-Pointer-Traversierung braucht keinen Stack und passt daher
unverändert in einen Metal-Kernel. Knoten und Sterne werden als float32 in gemeinsam genutzte Buffer
(`MTLResourceStorageModeShared`, Unified Memory) geschrieben, es wird nichts kopiert. Anbindung per cgo und
Objective-C im Präambel (`internal/sim/bh/gpu_darwin.go`, Build-Tag `darwin && cgo`). Andere Plattformen
nutzen den CPU-Pfad (Stub). In der App ist das der Standard, mit `-gpu=false` abschaltbar.

## Messungen (M4 Pro, ohne Last)

| | CPU (12 Kerne) | GPU-Hybrid | davon Kernel |
|---|---|---|---|
| 100k | 11,8 ms | 4,6 ms | 1,2 ms |
| 1M | 155 ms | 40 ms | 16 ms |

Abweichung GPU gegenüber CPU-Walk: Median 3·10⁻⁷, Max 1,7·10⁻⁵ (float32), weit unter dem Baumfehler (5·10⁻⁴).
App mit 1M Sternen und allen Effekten: 12–13 Schritte/s, 12–18 FPS.

## Alternativen
- **purego + Objective-C-Laufzeit** (ohne cgo): möglich, da purego ohnehin über Ebitengine im Modul ist.
  Aber `dispatchThreads` nimmt `MTLSize` als Struct per Wert, das ist über purego fragil. cgo ist robuster,
  und auf macOS ist ein C-Compiler mit den Command Line Tools vorhanden.
- **WebGPU aus Go** (cogentcore `gpu`/`gosl`, `gogpu/wgpu`): plattformübergreifend, aber neue, große bzw.
  junge Abhängigkeiten. Bleibt eine Option, falls Linux/Windows-GPU gewünscht wird. Der Kernel ist klein
  und ließe sich nach WGSL übertragen.
- **Voller GPU-Baum** (Radix-Sort und Karras-Build im Shader): deutlich aufwendiger. Nach der
  Parallelisierung bleibt der CPU-Aufbau mit ~20 ms bei 1M der größte Posten, es lohnt sich erst später.

## Konsequenzen
- cgo auf macOS: Build braucht die Xcode Command Line Tools. `CGO_ENABLED=0` baut weiterhin (CPU-Pfad).
- Bei 1M sind jetzt CPU-Aufbau (~20 ms), Loch-Kräfte und Rendering (4M Vertices pro Frame) die Engpässe.
  Nächste Hebel: Teilbäume parallel bauen, Sterne direkt als GPU-Punkte rendern statt CPU-Quads.
- Die Energie-Diagnose läuft weiter auf der CPU, mit adaptivem Intervall (≤ 5 % Rechenzeit).

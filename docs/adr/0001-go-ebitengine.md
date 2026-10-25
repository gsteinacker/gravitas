# ADR-0001: Go + Ebitengine, Physik auf der CPU

- Status: accepted (Oberfläche ersetzt durch ADR-0011)
- Datum: 2026-10-23
- Tasks: T-001

## Kontext
Interaktive N-Body-Simulation mit möglichst vielen Sternen, Fokus auf Spaß und schnelle Iteration,
entwickelt mit einem AI-Harness. Ausführliche Analyse: [01-tech-stack.md](../analysis/01-tech-stack.md).

## Entscheidung
Go (≥ 1.27) mit Ebitengine v2 für Fenster, Eingabe und Rendering. Physik auf der CPU, parallelisiert über
Goroutinen, mit Barnes-Hut. Daten als Struct-of-Arrays in float64.

## Alternativen
- Rust + wgpu: höhere Performance-Obergrenze, aber langsamere Iteration.
- TypeScript + WebGPU: GPU-Compute im Browser, aber schwächer bei CPU-Baumcodes und beim Testen.
- C++/Zig, Python: siehe Analyse.

## Konsequenzen
- Realistisches Ziel ca. 10⁵ Sterne interaktiv. Darüber hinaus wäre GPU-Compute nötig.
- Physik hinter `Solver`-Interface, damit ein späteres GPU-Backend möglich bleibt.
- Re-Evaluierung nach M2, falls 100k Sterne < 15 Schritte/s.
- Re-Evaluierung nach M2 (2026-10-23, T-010): 100k Sterne laufen mit 27–46 Schritten/s, je nach
  Entwicklungsstand der Galaxie. Das Abbruchkriterium (< 15 Schritte/s) ist klar verfehlt, **Go bleibt**.
- Nachprüfung 2026-10-23 (Analyse, Abschnitt 6): Ein anderer Sprachstack brächte auf der CPU nur ~2–3×. Faktor 10 erfordert einen GPU-Baumcode. Das ist eine Backend-Frage und lässt sich auch aus Go lösen (T-023).

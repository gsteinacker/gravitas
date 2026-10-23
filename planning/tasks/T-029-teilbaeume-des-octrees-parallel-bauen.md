---
id: T-029
title: Teilbaeume des Octrees parallel bauen
status: done
milestone: M2
priority: 3
estimate: M
depends: [T-023]
---

## Ziel

Der Baumaufbau ist bei 1M mit ~20 ms der größte CPU-Posten im GPU-Pfad. Die Rekursion über die Knoten (~10 ms) läuft seriell: Teilbäume ab einer Ebene (z. B. 64 Oktanten) parallel in eigene Knoten-Slices bauen, danach Indizes verschieben und die oberen Knoten von unten nach oben abschließen.

## Akzeptanzkriterien

- [x] Ergebnis bitidentisch zum seriellen Aufbau (Test, Kräfte und Knotenzahl, 40k und 300k)
- [x] `BenchmarkBuild` bei 1M mindestens 2× schneller: 46,7 → 8,8 ms (5,3×)
- [x] `BenchmarkGPUvsCPU/N=1000000/GPU` ≤ 30 ms: 27,3 ms (vorher 40,2)

## Notizen
- Obere Ebenen seriell bis Teilbäume ≤ max(8·leafSize, N/(8·Worker)), Teilbäume parallel in eigene Slices, Zusammenführen parallel über Präfixsummen, obere Knoten rückwärts abschließen (`finish`).
- Bitidentisch, weil die Walk-Reihenfolge nur über first/next definiert ist und die Summen in derselben Reihenfolge gebildet werden. Das Potential summiert Teilergebnisse in Abschlussreihenfolge und war schon vorher nur bis auf Rundung reproduzierbar (Test prüft 1e-12).
- Auch die float32-Umrechnung und das Rücklesen im GPU-Pfad sind parallel.

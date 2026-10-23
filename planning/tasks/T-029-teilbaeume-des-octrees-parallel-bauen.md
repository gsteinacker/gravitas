---
id: T-029
title: Teilbaeume des Octrees parallel bauen
status: todo
milestone: M2
priority: 3
estimate: M
depends: [T-023]
---

## Ziel

Der Baumaufbau ist bei 1M mit ~20 ms der größte CPU-Posten im GPU-Pfad. Die Rekursion über die Knoten (~10 ms) läuft seriell: Teilbäume ab einer Ebene (z. B. 64 Oktanten) parallel in eigene Knoten-Slices bauen, danach Indizes verschieben und die oberen Knoten von unten nach oben abschließen.

## Akzeptanzkriterien

- [ ] Ergebnis bitidentisch zum seriellen Aufbau (Test)
- [ ] `BenchmarkBuild` bei 1M mindestens 2× schneller
- [ ] `BenchmarkGPUvsCPU/N=1000000/GPU` ≤ 30 ms

## Notizen

# Performance-Log

Chronologisches Log relevanter Benchmark-Ergebnisse. Neueste oben. Format:

```
## YYYY-MM-DD · <Task-ID> · <kurze Beschreibung>
Maschine: M4 Pro, GOMAXPROCS=12, Go 1.27.x
<benchstat- oder go test -bench Ausgabe>
Fazit: …
```

## 2026-10-23 · T-008/T-009 · Barnes-Hut, parallel

Maschine: M4 Pro (8P+4E), Go 1.27.1, θ = 0,6, Scheibe + 2 BHs

```
BenchmarkAccelerations/N=10000-12       1.35 ms/op   27 allocs/op
BenchmarkAccelerations/N=100000-12     18.8  ms/op   27 allocs/op   (seriell: 137.6 ms → 7.3×)
BenchmarkAccelerations/N=1000000-12   279    ms/op   27 allocs/op
BenchmarkBuild-12 (N=100k)              3.6  ms/op    0 allocs/op
```

Budget (N=100k): Build 3,6 ms ≤ 8 ✓, Kräfte ~15 ms ≤ 20 ✓.
In der App: 100k Sterne bei ~30 Schritten/s und 58–60 FPS (Ziel ≥ 30 ✓). Die Lücke zum Benchmark (~53/s) ist T-024.
Gegenüber der direkten Summe ist N=100k ~85× schneller.

## 2026-10-23 · T-003 · Direkter Löser, parallel

Maschine: M4 Pro (8P+4E), GOMAXPROCS=12, Go 1.27.1

```
BenchmarkAccelerations/N=1000-12     282 µs/op   3.5 Gpairs/s   97 allocs/op
BenchmarkAccelerations/N=10000-12   16.4 ms/op   6.1 Gpairs/s   97 allocs/op
```

In der App: N=8000, Spiralgalaxie, ~100 Schritte/s bei 60 FPS (Renderer kein Engpass).
Fazit: Für M0/M1 reicht die direkte Summe. 100k Sterne bräuchten ~1,6 s/Schritt, dafür kommt Barnes-Hut (T-008).

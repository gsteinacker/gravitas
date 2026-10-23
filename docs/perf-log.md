# Performance-Log

Chronologisches Log relevanter Benchmark-Ergebnisse. Neueste oben. Format:

```
## YYYY-MM-DD · <Task-ID> · <kurze Beschreibung>
Maschine: M4 Pro, GOMAXPROCS=12, Go 1.27.x
<benchstat- oder go test -bench Ausgabe>
Fazit: …
```

## 2026-10-23 · T-003 · Direkter Löser, parallel

Maschine: M4 Pro (8P+4E), GOMAXPROCS=12, Go 1.27.1

```
BenchmarkAccelerations/N=1000-12     282 µs/op   3.5 Gpairs/s   97 allocs/op
BenchmarkAccelerations/N=10000-12   16.4 ms/op   6.1 Gpairs/s   97 allocs/op
```

In der App: N=8000, Spiralgalaxie, ~100 Schritte/s bei 60 FPS (Renderer kein Engpass).
Fazit: Für M0/M1 reicht die direkte Summe. 100k Sterne bräuchten ~1,6 s/Schritt, dafür kommt Barnes-Hut (T-008).

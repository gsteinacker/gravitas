# Performance-Log

Chronologisches Log relevanter Benchmark-Ergebnisse. Neueste oben. Format:

```
## YYYY-MM-DD · <Task-ID> · <kurze Beschreibung>
Maschine: M4 Pro, GOMAXPROCS=12, Go 1.27.x
<benchstat- oder go test -bench Ausgabe>
Fazit: …
```

## 2026-10-23 · T-027 · Gruppen-Walk – verworfen

Ohne Last, 100k (alter Walk gegen Gruppen-Walk mit Blättern als Gruppen):

```
Accelerations t=0:     17.3 ms → 25.0 ms   (+45 %)
AccelerationsEvolved:  29.0 ms → 41.0 ms   (+41 %)
Wechselwirkungen/Stern: pro Stern 530 | Gruppen à 4: 634 | à 8: 693 | à 24: 757
```

Fazit: Die Laufzeit ist proportional zu den Wechselwirkungen. Das konservative Gruppenkriterium kostet mehr, als
die gesparte Traversierung bringt (skalarer Code). Nicht übernommen. Nächster Hebel: Quadrupol-Momente (T-028).

## 2026-10-23 · T-025 · Mehrfach-Zeitschritte nahe Schwarzer Löcher

Relativ gemessen (App lief parallel), 100k-Galaxie mit 2 BHs, vollständiger `Step`:

```
alter Commit (fester dt):      50.97 ms/step (Mittel aus 6)
neu (16 Unterschritte):        51.82 ms/step (Mittel aus 6)   → +1.7 %
```

Energie-Drift nach 500 Schritten (N = 2000, c = 10): direkt 0,297 → 0,00047, Barnes-Hut 0,222 → 0,00075.

## 2026-10-23 · T-024/T-010 · Triple-Buffer, Blattgröße, entwickelter Zustand

Profil der laufenden App (100k, 20 s): `walk` 78 %, Rendern 8 %, Energie-Diagnose 5 %.
CPU-Auslastung 616 % → 714 % nach dem Triple-Buffer. Die Schritte/s blieben aber gleich (~25 bei t ≈ 2,5).

Ursache der Lücke zum Benchmark: Der Benchmark maß nur t = 0. Headless, ohne App:

```
t = 0.4: 21.6 ms/step   t = 1.2: 29.0   t = 2.0: 35.6   t = 2.8: 38.1
```

Die App ist also so schnell wie die nackte Simulation. Blattgröße am entwickelten Zustand (t = 2,4):

```
leafSize  4: 44.4 ms   8: 39.9   12: 40.5   16: 37.6   24: 36.0   32: 35.7
```

→ leafSize = 24. Danach:

```
BenchmarkAccelerations/N=100000-12      17.5 ms/op    27 allocs/op
BenchmarkAccelerations/N=1000000-12    260   ms/op
BenchmarkAccelerationsEvolved-12        35.1 ms/op    (100k, t = 2.4)
BenchmarkPotentialEnergy-12             50   ms/op    (θ = 0.3)
BenchmarkBuild-12                        3.3 ms/op     0 allocs/op
```

Fazit: Die App ist nicht der Engpass, sondern die Kraftberechnung bei dichtem Kern. Nächster Hebel ist T-027 (Gruppen-Walk).

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

# Performance-Log

Chronologisches Log relevanter Benchmark-Ergebnisse. Neueste oben. Format: `## Datum · Task · Thema`, dann
Maschine und Last, die Zahlen in einem Codeblock und eine kurze Einordnung.

## 2026-10-29 · T-087 · SPH-Nachbarsuche bei gemischten Glättungslängen

Messung (Szene 14, 43 000, 12 400 Gas): Dichtes Scheibengas prüfte im Kraftteil rund 700 Kandidaten für 60
Nachbarn. Ursache sind wenige weit herausgeschleuderte Halo-Teilchen mit h bis 20, die über die symmetrische
Bedingung r < 2·max(h_i, h_j) formal die ganze Scheibe erreichen. Jedes Blatt mit so einem Teilchen wurde
komplett (8 Körper) geprüft. M4 Pro, keine Fremdlast, T-085-Stand und neuer Stand abwechselnd.

```
Gasauswertung Szene 14 / 43 000                     20,3 ms → 17,5 ms (1 Kern: 145 → 130 ms, 7,4× auf 12)
Szene 14 dichter Zustand, 43 000                     38,5 → 46,0 Schritte/s
Szene 14 dichter Zustand, 8000                        168 → 175 Schritte/s
Pferdekopfnebel frisch, 60k                          12,7 → 13,3 Schritte/s
```

Änderungen: Die Kandidaten der Gruppensuche sind Blätter statt Bereiche; jeder Körper verwirft Blätter, deren
Box außer Reichweite liegt (Dichte: 787 → 118 geprüfte Kandidaten pro Körper). Die Körper eines Blatts sind nach
h absteigend sortiert; liegt das Blatt außerhalb des eigenen Radius, bricht die Prüfung beim ersten Körper ab,
der nicht mehr hinreicht. `boxDist2` verzweigungsfrei (inline). Rest: Die Halo-Teilchen sind echte Partner der
symmetrischen Formulierung; weniger Arbeit gäbe es nur mit einer Obergrenze für h (Physikänderung, nicht gemacht).

## 2026-10-29 · T-085 · Eigene Zeitschritte für Gas

Anlass: Nutzer sah die Kerne weiter nur zur Hälfte ausgelastet. Profil Szene 14 (8000): Jeder Gas-Teilschritt
wartete auf eine volle GPU-Gravitationsrechnung (6 ms Latenz), und das ganze Gas rechnete im Takt der dichtesten
Teilchen. M4 Pro, keine Fremdlast, alter Stand (`f2cdca0`) und neuer Stand abwechselnd (`tools/evolve`).

```
BenchmarkHorsehead (eine Gasauswertung, 60k)          41–48 ms → 40 ms, 120 343 → 161 allocs/op
Pferdekopfnebel frisch, 60k, 21 Schritte                7,9 → 14,9 Schritte/s (Gasauswertungen 2,1 → 2,2)
Pferdekopfnebel entwickelt (50k Gas)                    9,6 → 17,5 Schritte/s
Szene 14 dichter Zustand, 8000                           71 → 168 Schritte/s
Szene 14 dichter Zustand, 43 000 (12 400 Gas)            21 → 38 Schritte/s
Szene 14 ganze 550 Myr, 8000 (Mittel)                   106 → 198 Schritte/s
Spirale 400 Myr, 20k (kein Gas-Teilschritt)             243 → 236 Schritte/s (Verwaltung, im Rauschen)
Gasauswertung Szene 14 / 43 000 nach Kernen     1: 145 ms, 4: 41 ms, 8: 27 ms, 12: 20 ms (7,1×)
```

Änderungen: Gravitation einmal pro Schritt, parallel zur letzten Gasauswertung; Gas in Blockzeitschritten, nur
aktive Teilchen werden ausgewertet (ADR-0018). Die Dichte-Closure legte pro Teilchen Speicher an (seit T-086
unbemerkt), jetzt allokationsfrei. Der Rest-Engpass bei 43 000 ist die symmetrische Nachbarsuche im Kraftteil
(Scan der Kandidaten, 50 % der Zeit) in Gas mit sehr unterschiedlichen Glättungslängen.

## 2026-10-29 · T-083, T-086 · Gasrechnung: Kernauslastung, Gruppensuche, Sink

Anlass: Nutzer sah bei Gasszenen 5 Schritte/s und kaum ausgelastete Kerne. Profil (Pferdekopfnebel, 50 000 Gas):
87 % der Zeit in der SPH-Nachbarsuche, der Baumaufbau (Vergleichssortierung, 13,5 ms) seriell, `updateGas`
seriell, die CPU wartet während der GPU-Gravitation. M4 Pro, keine Fremdlast, alter Commit (Worktree) und neuer
Stand abwechselnd.

```
BenchmarkHorsehead (eine Gasauswertung, 60k)      66 ms  →  48 ms
  davon Baumaufbau                                13,5 ms →  3,0 ms (Radix-Sort, keine Allokationen)
Pferdekopfnebel, ganzer Schritt, 50k             4,4–4,7 → 5,6–5,9 Schritte/s
Szene 14 dichter Zustand (8000)                   25,5 → 59 Schritte/s (mit Sink, ADR-0017)
Szene 14 ganze 550 Myr (8000, Mittel)               41 → 76 Schritte/s
Spirale 400 Myr (20k)                              159 → 225 Schritte/s, neue Sterne 70 → 74
```

Änderungen: Nachbarsuche je Baumblatt (8 Teilchen) statt je Teilchen, kleinere Arbeitspakete (dichte Kerne
blockierten einen Kern), Radix-Sort und parallele Schlüssel im Baumaufbau, Gas parallel zur Gravitation
(eigene Beschleunigungspuffer, danach addiert), `updateGas` parallel. Dichte und Kräfte skalieren 8–10× auf
12 Kernen (8P + 4E). Der nächste Hebel sind eigene Zeitschritte für Gas (T-085).

## 2026-10-29 · T-080, T-081 · Supernovae, Gaszufluss, höhere Sternentstehung

`tools/evolve -preset 1 -n 20000 -t 15` (200 Myr), M4 Pro, keine Fremdlast. Abwechselnd gemessen: neuer Stand,
alter Commit (Worktree), neuer Stand.

```
neu (Supernovae, Zufluss, ε 0,066)   167,9 / 168,1 Schritte/s   1055 Gas, 35 neue Sterne
T-080 (nur Supernovae)               176,3 Schritte/s           1012 Gas, 14 neue Sterne
```

Etwa 5 % langsamer. Gründe: 4 % mehr Gas und eine zusätzliche Kraftauswertung je neuem Gasteilchen (alle
~60 Schritte bei 20k). Über 1 Gyr sinkt die Rate von 170 auf 130 Schritte/s, weil das Gas wächst und dichter
wird (ADR-0016). Hinnehmbar; falls nötig, ließe sich die neue Kraftauswertung durch eine Beschleunigung nur für
das neue Teilchen ersetzen.

## 2026-10-26 · T-074 · Particle-Mesh im kosmologischen Modus

Szene 13 (100 Mpc/h, z = 50), M4 Pro, keine Fremdlast, 20 Schritte nach einem Aufwärmschritt:

```
50 000 Teilchen, Gitter 64³     12,4 ms/Schritt
262 144 Teilchen, Gitter 64³    29,3 ms/Schritt
```

Vier FFTs (Dichte hin, drei Gradienten zurück), parallel über die Zeilen jeder Achse. z = 50 → 0 sind etwa 390
Schritte, also Sekunden bis Minuten.

## 2026-10-26 · T-072 · Frame-Dragging

Szene 4 (Doppel-Loch im Haufen), 50k, M4 Pro, keine Fremdlast, je 30 Schritte nach einem Aufwärmschritt, zwei
Läufe:

```
ohne Spin            30,26 / 30,73 ms/Schritt
beide Löcher a* 0,9  30,46 / 30,94 ms/Schritt   (+0,7 %)
```

Die Lense-Thirring-Rotation kostet pro Kick O(N · Löcher) wie die Lochkräfte selbst. Ohne Spin bleibt nur eine
Prüfung über die Löcher.

## 2026-10-24 · T-046/T-047 · Anpassung an den Rechner

Maschine: M4 Pro (8P+4E). **Unter Fremdlast:** Parallel lief eine Galaxy-Instanz mit 100k Sternen und ~4 Kernen
(PID des Nutzers, nicht beendet). Die absoluten Zahlen sind deshalb zu niedrig; der Vergleich fest/adaptiv lief
unter gleicher Last.

Schwacher Rechner, nachgestellt mit `GOMAXPROCS=2 -gpu=false`, Spiralgalaxie, Screenshot nach 25 s:

```
fest, -n 50000                       50 001 Körper   11,9 Schritte/s   57 FPS   θ 0,90
fest, -n 50000, -adapt (Regler)      50 001 Körper   15,4 Schritte/s   58 FPS   θ 1,00
adaptiv (-adapt, Kalibrierung CPU)   17 001 Körper   34,7 Schritte/s   57 FPS   θ 0,90
```

M4 Pro mit GPU, `-n 0` (Profil „Automatisch“), nach 20 s:
- Spiralgalaxie 100k: 68 Schritte/s;
- Kollision 120k: 53 Schritte/s;
- Antennae 93k: 61 Schritte/s;
- Pferdekopf 22k Gas: 8 Schritte/s (4 Courant-Teilschritte; die Korrektur über die gemessene Schrittzeit greift
  beim nächsten Laden der Szene).

Retina (`-scale 2`, 30k): 60 FPS wie bei Faktor 1. Bloom-Verkleinerung wächst mit dem Faktor, ihr Aufwand bleibt
gleich.

## YYYY-MM-DD · <Task-ID> · <kurze Beschreibung>
Maschine: M4 Pro, GOMAXPROCS=12, Go 1.27.x
<benchstat- oder go test -bench Ausgabe>
Fazit: …
```

## 2026-10-24 · T-038 · SPH-Nachbarsuche (Pferdekopf, 60k Gas)

Maschine: M4 Pro (8P+4E), GOMAXPROCS=12, Go 1.27.1

```
BenchmarkHorsehead-12 (eine Hydro-Auswertung)
  vorher                                   151.9 ms/op
  Morton-Bitgrenzen statt Anzahl-Teilung    69.1 ms/op
  sortierte Kopien, Morton-Reihenfolge      53.9 ms/op
  h-Iteration pro Teilchen (Toleranz 5 %)   39.5 ms/op
```

In der App: 4,5 → 15–18 Schritte/s. Fazit: Blätter waren 4× größer als h, weil Morton-Abschnitte nach
Anzahl geteilt wurden; jetzt kompakte Oktree-Zellen. Rest: symmetrische Kraftsuche (~60 %).

## 2026-10-24 · T-034 · θ auf DM-dominierter Galaxie

Szene „Spiral galaxy“ mit Hernquist-Halo, Kraftfehler der Sterne gegenüber direkter Summe (6000) und Zeit (150k, CPU):

```
θ=0.5  median 7.8e-4  max 7.7e-3  193 ms
θ=0.6  median 1.5e-3  max 1.5e-2  126 ms
θ=0.7  median 2.5e-3  max 3.1e-2   90 ms
θ=0.8  median 3.9e-3  max 6.4e-2   51 ms
θ=0.9  median 5.7e-3  max 8.0e-2   33 ms   ← Standard
```

Fazit: In DM-dominierten Systemen ist der relative Fehler ~10× größer als im lochdominierten Testsystem. Er liegt
noch im üblichen Rahmen und unter dem Diskretheitsrauschen der schweren DM-Teilchen.

## 2026-10-23 · T-030 · Rendering bei 1M

```
1M, alle Effekte, ruhende Kamera:   12–18 FPS → 28 FPS (DisableMipmaps, 16-px-Sprite) → 61 FPS (Sternbild-Cache)
1M Schritte/s: 12–13 → 15        100k: 60 FPS, ~100 → 113 Schritte/s
Gegentests (vor dem Cache): ohne Post-Effekte 28 FPS, mit CPU-Simulation 12 FPS
```

## 2026-10-23 · T-029 · Paralleler Baumaufbau (Teilbäume)

```
BenchmarkBuild N=100k:  serial 4.05 ms → parallel 1.39 ms
BenchmarkBuild N=1M:    serial 46.7 ms → parallel 8.8 ms   (5.3×)
GPUvsCPU N=1M/GPU:      40.2 → 27.3 ms (Kernel 16.1 ms)
GPUvsCPU N=1M/CPU:      155  → 142  ms
AccelerationsEvolved (CPU, 100k): 20.7 → 18.1 ms
```

## 2026-10-23 · T-023 · GPU-Baumdurchlauf (Metal) + paralleler Baumaufbau

Ohne Last:

```
                    CPU        GPU-Hybrid   Kernel
N=100k            11.8 ms       4.6 ms     1.2 ms
N=1M             155   ms      40   ms    16   ms
Build 100k:  3.3 → 2.1 ms (parallel: collect, keys, radix sort, copies)
Build 1M Phasen (seriell, vorher): collect 8.7, keys 5.5, sort 10.7, copy 6.2, nodes 9.6 ms
```

App mit 1M: Zuerst dominierte die Energie-Diagnose (`walkPotential` 66 % CPU), weil eine Messung länger dauerte als
ihr 2-s-Intervall. Danach adaptives Intervall (≤ 5 %) und Diagnose-θ 0,9 (Fehler 1,2e-6, halbe Kosten).
Ergebnis: 1M Sterne mit 12–13 Schritten/s und 12–18 FPS, alle Effekte.

## 2026-10-23 · T-028 · Quadrupol-Momente, θ 0,6 → 0,9

Ohne Last, 100k. θ-Scan am entwickelten Zustand (Median / Max des relativen Kraftfehlers, ms pro Kraftberechnung):

```
θ=0.60 Monopol    2.3e-3 / 3.5e-2   35.3 ms
θ=0.60 Quadrupol  1.3e-4 / 1.3e-3   40.0 ms
θ=0.75 Quadrupol  2.5e-4 / 2.9e-3   26.7 ms
θ=0.90 Quadrupol  4.6e-4 / 5.7e-3   20.7 ms   ← neuer Default
θ=1.00 Quadrupol  6.3e-4 / 2.3e-2   18.0 ms
```

Benchmarks danach: Accelerations t=0 17.3 → 13.4 ms, Evolved 29.0 → 20.7 ms (−29 %),
PotentialEnergy (Diagnose θ 0.3 → 0.6) 50 → 20 ms bei einem Fehler von 4e-4 → 1.5e-6, Step 28 ms.

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

---
id: T-038
title: Szene Pferdekopfnebel
status: done
milestone: M6
priority: 2
estimate: M
depends: [T-037]
---

## Ziel

Szene Barnard 33 / IC 434: Rand der Molekülwolke L1630, von σ Ori beleuchtet; ein dichter Klumpen wird
durch Photoverdampfung zum Kopf mit Hals geformt, vor dem roten Leuchten der HII-Region.

## Akzeptanzkriterien

- [x] Szene `9-horsehead.json` mit Einführung (3 Stufen) und Themen (Ionisation, Staub, Gas, Kühlung, Sternentstehung)
- [x] Stabil über ≥ 0,5 ZE (750 kyr), keine NaN (headless mit `tools/evolve` geprüft)
- [x] Dunkle Säule mit Kopf und hellem Rand vor IC 434 (Screenshot t = 0,5)
- [x] ≥ 10 Schritte/s bei 60k Teilchen (vorher 4,5; jetzt 15–18)

## Notizen

- SPH-Nachbarsuche 152 → 40 ms (BenchmarkHorsehead): Baum teilt an Morton-Bitgrenzen statt nach Anzahl
  (Blätter waren 4× größer als h), sortierte Positionskopien, Abfragen in Morton-Reihenfolge, h-Iteration
  pro Teilchen (meist 1 Durchlauf statt 3).
- IC 434: 20 cm⁻³ in 16×4×10 pc, damit die freie Kante den Kopf erst nach ~0,5 Myr erreicht; κ_r = 2000
  (Strömgren-Abschätzung: τ ≈ 0,3 bis zum Kopf).
- Effektive Temperatur 100 K statt 20 K: ersetzt die fehlende magnetische/turbulente Stütze. Mit 20 K
  kollabierte die ganze Wolke in 0,5 ZE und bildete überall Sterne; mit 100 K bleibt sie stabil, der
  Druck der HII-Region ist ~3× höher (statt 18×). Sternentstehung ist aktiv, bis t = 0,5 aber keine
  (ρ_max 28–36 < 50), passend zu den wenigen jungen Objekten in B33.
- `tools/evolve`: Szene headless rechnen, Spielstand mit Kamera schreiben, `-in` zum Fortsetzen.
- Folge: T-041 (HDR-Emission, Sättigung/Flecken).

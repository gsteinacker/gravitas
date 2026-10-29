---
id: T-087
title: Schnellere SPH-Nachbarsuche bei stark unterschiedlichen Glättungslängen
status: done
milestone: M9
priority: 2
estimate: M
depends: [T-085]
---

## Ziel

Nach T-085 ist in klumpigen Gasszenen die SPH-Nachbarsuche der Engpass: In Szene 14 (43 000 Teilchen) entfällt
die Hälfte der Gasrechnung auf das Prüfen von Kandidaten im Kraftteil. Gas mit sehr unterschiedlichen
Glättungslängen (dünner Halo neben dichter Scheibe) bläht die Kandidatenlisten der Gruppensuche auf. Ziel: weniger
geprüfte Kandidaten pro gefundenem Nachbarn, bei gleichem Ergebnis.

## Akzeptanzkriterien

- [x] Kandidaten pro Nachbar gemessen (vorher/nachher), Engstelle benannt
- [x] Ergebnis der Nachbarsuche unverändert (`TestNeighborSearchMatchesBruteForce`, Gasauswertung bitgleich oder
      nur in der Summationsreihenfolge verschieden)
- [x] Benchmarks vorher/nachher in `docs/perf-log.md` (`BenchmarkHorsehead`, Szene 14 dicht), `make race`

## Notizen

Nutzer am 2026-10-29: „ja, Nachbarsuche auch noch optimieren“.

Erledigt 2026-10-29: Engstelle sind Halo-Teilchen mit sehr großem h (bis 20), die über die symmetrische Bedingung
die ganze Scheibe erreichen und ihre Blätter in jede Kandidatenliste bringen. Kandidaten pro Körper gefiltert,
Blätter nach h sortiert mit frühem Abbruch. Gasauswertung Szene 14 / 43 000: 20,3 → 17,5 ms, Szene dicht
38,5 → 46 Schritte/s. Test `TestGroupScanMatchesBruteForce`. Ergebnis nur in der Summationsreihenfolge anders.

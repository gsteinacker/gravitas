---
id: T-010
title: Benchmarks und Performance-Budget
status: done
milestone: M2
priority: 2
estimate: S
depends: [T-009]
---

## Ziel

Messbarkeit gegen das Budget in docs/architecture.md.

## Akzeptanzkriterien

- [x] `make bench` für alle Phasen (Build, Kräfte t = 0 und entwickelt, Energie, direkt)
- [x] Eintrag in docs/perf-log.md, Budget-Tabelle mit Messwerten in docs/architecture.md
- [x] Re-Evaluierung von ADR-0001 dokumentiert (Go bleibt)

## Notizen
- Budget „30 Schritte/s am entwickelten Zustand“ wird knapp verfehlt (35,1 ms statt ≤ 33 ms) → T-027.
- `make bench` dauert durch das Vorlaufen der Galaxie für den entwickelten Zustand etwa eine Minute.

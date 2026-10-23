---
id: T-031
title: Dunkle Materie als unsichtbare Koerperart
status: done
milestone: M5
priority: 1
estimate: M
depends: [T-020]
---

## Ziel

Galaxien brauchen Halos aus Dunkler Materie für realistische Rotationskurven und Verschmelzungen (Lokale Gruppe). Nutzerwunsch: schaltbar und optional sichtbar.

## Akzeptanzkriterien

- [x] Körperart DarkMatter, gravitativ wie Sterne, unsichtbar
- [x] Zur Laufzeit abschaltbar (`x`): Testteilchen ohne Quellmasse, Test: Sterne spüren exakt nichts
- [x] Sichtbar machbar (`v`, Effekt `dark`)
- [x] ADR-0006

## Notizen
- Gefundener Fehler beim Einbau: `PairRS` gab einem Stern-DM-Paar einen Schwarzschild-Radius, weil es nur auf „verschiedene Arten“ prüfte.
- 7 Stellen setzten „kein Loch“ mit `== Star` gleich. Die Konvention steht jetzt in CLAUDE.md.

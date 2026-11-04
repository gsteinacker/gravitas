---
id: T-075
title: Kosmisches Netz Etappe 3a: Friends-of-Friends und beschriftete Haufen
status: blocked
milestone: M11
priority: 3
estimate: M
depends: [T-074]
---

## Ziel

Die Haufen im kosmischen Würfel werden als Objekte erkennbar: ein Friends-of-Friends-Halo-Finder (Etappe 3 aus
`docs/analysis/04-universe.md`, erster Teil) findet die gebundenen Klumpen, die größten werden mit ihrer Masse
beschriftet. Ohne neues Kraftgesetz, deshalb vor TreePM.

## Akzeptanzkriterien

- [ ] FoF mit Verknüpfungslänge 0,2 × mittlerer Teilchenabstand, periodisch, ohne Allokationen pro Körper im
      Hot Loop, parallel (`make race`); Benchmark bei 64³/128³ in `docs/perf-log.md`
- [ ] Tests: zwei getrennte Plummer-Klumpen ergeben zwei Halos mit richtiger Masse; Klumpen über die Würfelkante
      wird als ein Halo erkannt; Gleichverteilung ergibt keine großen Halos
- [ ] Beschriftung der N größten Halos in M☉ (Go liefert Positionen und Texte, Swift zeichnet nur)
- [ ] Halo-Massenfunktion als (i)-Diagramm, qualitativ gegen Tinker et al. 2008 (Auflösungsgrenze genannt)
- [ ] Lernmodus: Thema „Halos“ in drei Stufen, Modellgrenze (PM glättet unterhalb der Gitterzelle)
- [ ] Screenshot-Prüfung der Szene 13 mit Beschriftung

## Notizen

Aus T-062. Davis et al. 1985 (FoF). Bis TreePM (T-076) sind kleine Halos durch das PM-Gitter verwaschen; die
Beschriftung gilt daher nur oberhalb einer Mindestteilchenzahl (z. B. 32).

- 2026-11-04: Blockiert, die Szenen zum kosmischen Netz sind entfernt (T-094). Nur aufnehmen, wenn sie zurückkommen.

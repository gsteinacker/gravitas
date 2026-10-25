---
id: T-060
title: Intro im Stil der Star-Wars-Lauftexte
status: todo
milestone: M10
priority: 2
estimate: M
depends: [T-058]
---

## Ziel

Beim Start soll ein Intro im Stil des Star-Wars-Lauftexts erklären, was Galaxy simuliert und was daran wirklich cool
ist: Text in Gelb auf Sternenhimmel, perspektivisch geneigt, der in die Tiefe davonläuft. Beim ersten Klick (oder
einer Taste) verschwindet es, und die Simulation erscheint.

## Akzeptanzkriterien

- [ ] Lauftext perspektivisch geneigt (3D-Rotation), läuft nach oben in die Ferne und blendet aus; vorher kurze
      Einblendung „Vor langer Zeit in einer Galaxie …“-Anspielung ohne Markenzitate (eigener Text, kein Logo)
- [ ] Inhalt de/en: N-Körper-Physik mit 10⁴–10⁶ Sternen, Schwarze Löcher, einstellbare Lichtgeschwindigkeit,
      Gas, Staub, Dunkle Materie, echte Strukturen (Lokale Gruppe, Antennen, Pferdekopf), Lernmodus; was davon
      besonders ist (z. B. relativistische Effekte sichtbar machen, Galaxien kollidieren in Sekunden)
- [ ] Erster Klick oder Taste beendet das Intro sofort (sanftes Ausblenden); Simulation läuft im Hintergrund schon
- [ ] Einstellung „Intro beim Start zeigen“; nach dem ersten Mal standardmäßig aus, im Hilfe-Menü wieder abspielbar
- [ ] Screenshot-Modus und `-shot` zeigen kein Intro; Prüfung per Fenster-Schnappschuss
- [ ] Text physikalisch korrekt (CLAUDE.md, Lernmodus-Regeln), keine Übertreibungen

## Notizen

Wunsch des Nutzers vom 2026-10-25. Umsetzung in SwiftUI (`rotation3DEffect` + Animation), nach T-058.

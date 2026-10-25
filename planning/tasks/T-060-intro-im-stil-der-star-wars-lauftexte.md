---
id: T-060
title: Intro im Stil der Star-Wars-Lauftexte
status: done
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

- [x] Lauftext perspektivisch geneigt (3D-Rotation), läuft nach oben in die Ferne und blendet aus; vorher kurze
      Einblendung „Vor langer Zeit in einer Galaxie …“-Anspielung ohne Markenzitate (eigener Text, kein Logo)
- [x] Inhalt de/en: N-Körper-Physik mit 10⁴–10⁶ Sternen, Schwarze Löcher, einstellbare Lichtgeschwindigkeit,
      Gas, Staub, Dunkle Materie, echte Strukturen (Lokale Gruppe, Antennen, Pferdekopf), Lernmodus; was davon
      besonders ist (z. B. relativistische Effekte sichtbar machen, Galaxien kollidieren in Sekunden)
- [x] Erster Klick oder Taste beendet das Intro sofort (sanftes Ausblenden); Simulation läuft im Hintergrund schon
- [x] Einstellung „Intro beim Start zeigen“; nach dem ersten Mal standardmäßig aus, im Hilfe-Menü wieder abspielbar
- [x] Screenshot-Modus und `-shot` zeigen kein Intro; Prüfung per Fenster-Schnappschuss
- [x] Text physikalisch korrekt (CLAUDE.md, Lernmodus-Regeln), keine Übertreibungen

## Notizen

Wunsch des Nutzers vom 2026-10-25. Umsetzung in SwiftUI (`rotation3DEffect` + Animation), nach T-058.

Erledigt 2026-10-26 (`macos/GalaxySwift/Sources/GalaxySwift/IntroView.swift`):
- Erst die Einblendung „Vor nicht allzu langer Zeit, in genau dieser Galaxie …“, dann Titel und sechs Absätze in
  Gelb, um 50° gekippt mit Perspektive, die in 70 s nach oben in die Ferne laufen und oben ausblenden.
- Ein Klick oder eine beliebige Taste beendet das Intro (0,6 s Ausblenden). Die Taste wirkt dabei nicht in der
  Simulation, und die Simulation läuft dahinter schon.
- Beim ersten Start wird das Intro gezeigt, danach nur, wenn „Intro beim Start zeigen“ eingeschaltet ist. Es lässt
  sich auch über Hilfe › Intro abspielen und in den Einstellungen abspielen.
- Ohne `-intro` zeigt der Screenshot-Modus kein Intro. `-intro` schreibt `<png>.intro.png` (ein Standbild per
  ImageRenderer, da der Fenster-Schnappschuss keine 3D-Transformationen kann).
- Text geprüft: Teilchenzahl 10⁴–10⁶ (Einstellung bis 1 000 000), Teilchen als Sterngruppen, Barnes-Hut, Horizont
  und Doppler bei kleinerem c, Retardierung, Sternentstehung in den Armen, Szenen Lokale Gruppe, Antennen, S-Sterne
  und Pferdekopf.


---
id: T-089
title: 'Bug: Anzeige bleibt leer, wenn die App bei schlafendem Bildschirm startet'
status: done
milestone: M10
priority: 1
estimate: S
depends: []
---

## Ziel

Wird die App gestartet, während der Bildschirm schläft (z. B. per `open` aus einem Skript), bleibt das Fenster nach
dem Aufwachen leer. Bei wachem Bildschirm gestartet, zeichnet sie normal (Rückmeldung des Nutzers vom 2026-10-30).
Auch der Screenshot-Modus hing in diesem Zustand.

## Akzeptanzkriterien

- [x] Zeichenschleife wird neu angestoßen, wenn der Bildschirm aufwacht oder das Fenster den Bildschirm bzw. die
      Sichtbarkeit wechselt (`SimulationView.restartDrawing`)
- [x] Render-Ziele werden dabei neu angelegt, und die lange Pause zählt nicht als langsames Bild für den Regler
- [ ] Prüfung mit schlafendem Bildschirm: vom Nutzer nicht gewünscht, steht noch aus

## Notizen

Während der Bildschirm schläft, ruft MTKView `draw(in:)` nicht auf. Mit geweckter Anzeige (`caffeinate -u`) laufen
Screenshots und die App normal. Ursache vermutlich: Die Anzeigesynchronisierung des MTKView bindet sich beim Start
an keinen aktiven Bildschirm und wird danach nicht neu verbunden. `isPaused` aus- und wieder einschalten legt sie
neu an.

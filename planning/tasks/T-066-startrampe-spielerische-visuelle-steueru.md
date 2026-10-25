---
id: T-066
title: Startrampe: spielerische visuelle Steuerung beim Platzieren
status: done
milestone: M10
priority: 1
estimate: L
depends: [T-064]
---

## Ziel

Das Platzieren soll sich spielerisch und visuell anfühlen statt über Inspektor-Regler (Rückmeldung zu T-064). Vorbilder
(Recherche 2026-10-25): Universe Sandbox (Modi Ruhe/Orbit/Doppelsystem/Start, Gitter, Bahnvorschau), Kerbal Space
Program (Manöverknoten mit sechs farbigen Griffen prograd/radial/normal und Live-Bahn), „Orbit – Playing with
Gravity“ (Schleuder mit Vorschau), Homeworld (Scheibe plus Höhenlot), Blender/Unity (Griffe, Einrasten mit Modifier).

## Akzeptanzkriterien

- [x] Klick setzt ein halbtransparentes Geisterobjekt auf die Scheibenebene mit lokalem Koordinatensystem:
      Kompassscheibe (beschriftet: mit der Drehung, außen, Zentrum), Lot zur Ebene, Höhengriff ↕
- [x] Sechs farbige Griffe (prograd/retrograd, radial außen/innen, normal oben/unten): Ziehen ändert das Tempo in
      diese Richtung, Bahnvorschau live
- [x] Pfeilspitze frei ziehbar; Schleuder: Geisterobjekt zurückziehen und loslassen startet entgegengesetzt
- [x] Tacho am Objekt: Bogen mit Marken Kreisbahn- und Fluchtgeschwindigkeit, Nadel, km/s; magnetisches Einrasten von
      Tempo (Kreisbahn, Flucht) und Richtung (Achsen); ⌘ beim Ziehen ohne Einrasten
- [x] Mini-Leiste am Objekt: Ruhe, Kreisbahn, Doppelsystem (beide umkreisen den gemeinsamen Schwerpunkt, Impuls
      erhalten), Flucht, Zentrum, Los!, ✕
- [x] Kamera bleibt beim Planen frei drehbar; Inspektor bleibt als genaue Eingabe und ist synchron
- [x] Griff-Treffer, Einrasten und Doppelsystem im Go-Kern mit Tests; Prüfung per Fenster-Schnappschuss

## Notizen

- Go: `internal/engine/gizmo.go` (Griffe in fester Bildschirmgröße am lokalen System prograd/radial/normal,
  Treffertest, Ziehen je Griff im Maßstab des Pfeils, Schleuder, Ausblenden von Achsen, die auf die Kamera zeigen),
  `launch.go` (Einrasten 4 % bzw. 6°, Doppelsystem mit Impulsaufteilung nach Masse).
- Tests: Treffer, Griffziehen (Δv = v_K/2,4 je Grifflänge), Einrasten, Schleuder (Klick startet nicht), Impuls im
  Doppelsystem.
- Swift: Griffe/Geist als Layer, Cursor-Rückmeldung, Overlay mit Kompassbeschriftung, Tacho (Marken Kreisbahn grün,
  Flucht rot) und Mini-Leiste (Ruhe, Kreisbahn, Doppelsystem, Flucht, Zentrum, Los!, ✕). Inspektor bleibt synchron.
- Geprüft per Fenster-Schnappschuss (`-launch 0.68,0.5,1`); die Maus-Bedienung selbst nur über die Kern-Tests.

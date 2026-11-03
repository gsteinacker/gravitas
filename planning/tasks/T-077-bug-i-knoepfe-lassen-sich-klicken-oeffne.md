---
id: T-077
title: Bug: (i)-Knöpfe lassen sich klicken, öffnen aber nichts
status: done
milestone: M10
priority: 1
estimate: S
depends: []
---

## Ziel

Die (i)-Knöpfe neben den Einstellungen (T-061) reagieren auf den Klick, zeigen aber kein Popover. Die Erklärungen
sind damit in der App unerreichbar, obwohl sie im Screenshot-Modus (`-info`) funktionieren.

## Akzeptanzkriterien

- [x] Ursache gefunden und behoben, für alle (i) im Inspektor, in den Einstellungen und auf der Startrampe
- [x] Popover bleibt offen, bis man daneben klickt, auch während die Simulation läuft
- [x] Der Link „Im Lernmodus“ öffnet das Thema
- [x] Prüfung per echtem Klick (nicht nur `-info`)

## Notizen

Stand 2026-10-27:
- Go-Kern liefert für alle Einstellungen in allen 14 Szenen gültiges JSON (Test über `json.Marshal`).
- Programmatisch geöffnet (`shown = true`) erscheint das Popover sichtbar (`_NSPopoverWindow`, on screen) und
  bleibt offen. Der Zustand (`engine.state`) wird im Lauf nicht neu zugewiesen.
- Also liegt der Fehler im Klickweg: entweder kommt die Aktion nicht an, oder das Popover schließt sofort.
  Synthetische Klicks (CGEvent) blockiert macOS ohne Bedienungshilfen-Freigabe.
- Eingeschleuster Klick im eigenen Prozess (`NSApp.postEvent` mit Mausereignissen am (i)) öffnet das Popover
  im Screenshot- und im normalen Modus, für Tempo, c und θ. Am θ liegt per `hitTest` ein `SelectionTextField`
  (Textauswahl-Ebene von SwiftUI) über dem Knopf, der Klick kommt trotzdem an.
- Offen: Klick des Nutzers im Diagnose-Build (erster Versuch landete vermutlich in der installierten App).

Gelöst 2026-11-03:
- **Ursache:** Werte, die sich während des Laufs ändern, lagen im langsamen `Engine.state`. Daran hängen Menüs und
  Inspektor. Bei jeder Änderung baut SwiftUI sie neu auf, und dabei schließen sich offene Popovers und Untermenüs.
  Zuletzt waren das die Balkenwerte aus T-090 (alle 0,5 s, zusätzlich flackerte „Neu aus Szene“).
- **Fix:** Die Werte werden in der langsamen Kopie genullt und aus `live` gelesen. Hinweis an der C-Struktur und in
  `Engine.refresh`.
- **Nachweis:** Diagnoseprotokoll (Felder von `state`, die sich ändern) und echte Klicks des Nutzers. Menü und (i)
  funktionieren, das Popover bleibt offen, bis man daneben klickt.

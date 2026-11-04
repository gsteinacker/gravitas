---
id: T-093
title: Bug: (i)-Knöpfe schließen sich bei Meldungen und Live-Werten wieder
status: doing
milestone: M10
priority: 1
estimate: S
depends: []
---

## Ziel

Die (i)-Knöpfe im Inspektor öffnen ihr Popover und es bleibt offen, bis man daneben klickt (Rückfall nach
T-077). Ursache: SwiftUI baut Ansichten neu auf, die sich ändernde Werte lesen, und dabei schließen sich Popovers.
`StarStats` las `engine.live` (alle Werte, mehrmals pro Sekunde), `ContentView` las `engine.message` (jede Meldung
baute den Inspektor neu auf).

## Akzeptanzkriterien

- [x] `StarStats` liest nur `engine.state`; die Zahlen sind kleine `LiveText`-Ansichten, die allein `engine.live`
      lesen
- [x] Meldungen zeigt eine eigene Ansicht `MessageToast`; `ContentView` liest `engine.message` nicht mehr
- [x] Gemessen mit `Self._printChanges` (Diagnose, wieder entfernt), Szene 14 mit `-adapt`, 25 s: `StarStats`
      95 → 3 Neuaufbauten, `ContentView` und `InspectorView` nur noch beim Start (3), nicht mehr bei Meldungen
- [ ] Klicktest durch den Nutzer: (i) an Sternentstehung, Supernovae, c, θ bleibt offen, auch wenn eine Meldung
      erscheint

## Notizen

- Regel (Kommentare an `StarStats`, `LiveText`, `MessageToast`, `Engine.refresh`): Eine Ansicht mit (i), Menü oder
  Picker darf nur langsam veränderliche Werte lesen; alles, was sich im Lauf ändert, in eigene kleine Ansichten.

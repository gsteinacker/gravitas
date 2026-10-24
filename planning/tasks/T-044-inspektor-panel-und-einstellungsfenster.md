---
id: T-044
title: Inspektor-Panel und Einstellungsfenster
status: todo
milestone: M7
priority: 1
estimate: L
depends: [T-043]
---

## Ziel

Regler und Schalter statt Tastenkürzeln: Ein schwebender Inspektor zeigt und ändert die wichtigsten Parameter,
ein Einstellungsfenster speichert dauerhafte Vorlieben.

## Akzeptanzkriterien

- [ ] Inspektor-Panel (`NSPanel`) mit Reglern für c, Tempo, Masse, θ, Schaltern für Effekte, Auswahl für Werkzeug und
      Gravitation; Live-Werte steps/s, FPS, Substeps, Energiedrift
- [ ] Inspektor und Spieltasten bleiben synchron (Änderung per Taste bewegt den Regler)
- [ ] Einstellungsfenster (⌘,): Leistungsprofil, Sternzahl, GPU, Lernstufe, Standard-Effekte
- [ ] `internal/prefs`: JSON in Application Support, Roundtrip-Test, defekte Datei → Standardwerte
- [ ] Einstellungen gelten nach Neustart der App

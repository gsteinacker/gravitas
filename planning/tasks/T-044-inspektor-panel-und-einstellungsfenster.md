---
id: T-044
title: Inspektor-Panel und Einstellungsfenster
status: done
milestone: M7
priority: 1
estimate: L
depends: [T-043]
---

## Ziel

Regler und Schalter statt Tastenkürzeln: Ein schwebender Inspektor zeigt und ändert die wichtigsten Parameter,
ein Einstellungsfenster speichert dauerhafte Vorlieben.

## Akzeptanzkriterien

- [x] Inspektor-Panel (`NSPanel`) mit Reglern für c, Tempo, Masse, θ, Schaltern für Effekte, Auswahl für Werkzeug und
      Gravitation; Live-Werte steps/s, FPS, Substeps, Energiedrift
- [x] Inspektor und Spieltasten bleiben synchron (Änderung per Taste bewegt den Regler)
- [x] Einstellungsfenster (⌘,): Leistungsprofil, Sternzahl, GPU, Lernstufe, Standard-Effekte
- [x] `internal/prefs`: JSON in Application Support, Roundtrip-Test, defekte Datei → Standardwerte
- [x] Einstellungen gelten nach Neustart der App

## Notizen

Inspektor und Einstellungen werden wie das Menü aus einer Go-Beschreibung gebaut (`internal/mac/form.go`), Cocoa
fragt die Werte viermal pro Sekunde ab. Prüfung ohne Bildschirm: `GALAXY_FORM_SNAPSHOT=<dir>` schreibt beide
Fenster als PNG und klickt testweise „Spuren“ (Rundweg Widget → Go → Widget). Profil wirkt erst mit T-046.
Im Screenshot-Modus werden die Einstellungen ignoriert (reproduzierbare Messungen).

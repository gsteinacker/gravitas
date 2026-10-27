---
id: T-079
title: Menüstruktur vereinfachen (Recherche macOS-Konventionen)
status: done
milestone: M10
priority: 1
estimate: M
depends: []
---

## Ziel

Die Menüleiste ist zu verschachtelt und folgt den macOS-Konventionen nur teilweise. Sie soll einfacher werden und
sich an die Human Interface Guidelines halten.

## Recherche (HIG „The menu bar“, „Edit menus“)

- Reihenfolge: App, Ablage, Bearbeiten, Format, Darstellung, eigene Menüs, Fenster, Hilfe.
- Ablage ist für Öffnen, Sichern, Exportieren. Die App hat Dokumente (`.galaxy`, Szenen-JSON), Ablage bleibt also.
- Bearbeiten: Kopieren muss gehen, weil Erklärtexte auswählbar sind. Eigene Befehle nur, wenn sie wirklich
  Bearbeiten sind (Rückgängig gibt es nicht). Befehle ohne Funktion ausblenden statt leer anzeigen.
- Darstellung (View) ist der Ort für Ein-/Ausblenden von Seitenleiste, Inspektor und Effekten.
- Eigene Menüs: wenige, kurze Ein-Wort-Titel, nach Hierarchie und vom Allgemeinen zum Speziellen geordnet.
- Menüpunkte immer zeigen und bei Bedarf deaktivieren, nicht ausblenden. Standardkürzel respektieren.

## Vorschlag (zur Abstimmung)

Galaxy · Ablage · Bearbeiten · Darstellung · Simulation · Fenster · Hilfe

- **Ablage:** Neu aus Szene ▸ (gruppiert, T-078), Öffnen …, Sichern …, Schnell sichern, Letzten Stand laden,
  Bild exportieren …
- **Bearbeiten:** nur die Systembefehle (Kopieren, Alles auswählen), ohne leeres Rückgängig/Ausschneiden
- **Darstellung:** Inspektor, Lernmodus, Statusanzeige | Effekte (Leuchten, Spuren, Linse, Doppler, Staub, Gas,
  Dunkle Materie) | Ansicht zentrieren | Vollbild
- **Simulation:** Pause, Einzelschritt, Schneller, Langsamer, Neu starten | Licht schneller/langsamer,
  Gravitation der Löcher ▸, Dunkle Materie wirkt | Werkzeug ▸, Mehr/Weniger Masse
  (statt der drei Menüs Simulation, Physik, Werkzeug)
- **Hilfe:** Intro, Tastenübersicht, Lernmodus-Themen

## Akzeptanzkriterien

- [x] Struktur mit dem Nutzer abgestimmt
- [x] Umgesetzt in `GalaxyApp.swift`, de/en, Tastenkürzel unverändert erreichbar und ohne Doppelbelegung
- [x] Bearbeiten-Menü ohne funktionslose Einträge
- [x] Tastenübersicht (`KeyOverview`) passt zur neuen Struktur
- [x] Screenshot bzw. Liste der Menüs geprüft

## Notizen

Erledigt 2026-10-27 in `GalaxyApp.swift`. Ein Menü „Simulation“ statt Simulation, Physik, Werkzeug (Werkzeug und
Gravitation als Untermenüs). Bearbeiten nur mit Kopieren und Alles auswählen: Rückgängig, Ausschneiden, Einsetzen
und Textbearbeitung sind ersetzt, Diktat und Emoji per `NSDisabled…MenuItem` aus, AutoFill und überzählige Trenner
räumt `AppDelegate.tidyEditMenu` weg (auch beim Öffnen der Menüleiste, weil AppKit es neu einfügen kann). Keine
Fenster-Tabs mehr. Hilfe hat zusätzlich „Themen im Lernmodus“. Szenen ab der zehnten haben keine Taste mehr statt
alle „0“ (der Go-Kern kennt nur 1–9). Geprüft über einen Abzug von `NSApp.mainMenu` (Menüs fängt der
Screenshot nicht ein).

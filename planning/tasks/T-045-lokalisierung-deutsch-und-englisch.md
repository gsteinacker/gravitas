---
id: T-045
title: Lokalisierung Deutsch und Englisch
status: done
milestone: M7
priority: 2
estimate: M
depends: [T-043]
---

## Ziel

Die App soll Deutsch- und Englischsprachigen gleichermaßen dienen. Menüs, Dialoge, Inspektor und HUD folgen der
Systemsprache.

## Akzeptanzkriterien

- [x] `internal/i18n` mit Textpaaren de/en (`i18n.T("…", "…")`), Sprache aus `NSLocale.preferredLanguages`, sonst `LANG`
- [x] Menüs, Dialoge, Inspektor, HUD, Toasts und Hilfe-Overlay laufen über i18n
- [x] Test: jeder Schlüssel hat beide Sprachen und nutzt nur Glyphen der Go-Schrift
- [x] `-lang de|en` erzwingt eine Sprache (für Screenshots)

## Notizen

Statt einer Schlüsseltabelle stehen die Texte als Paare im Code (`i18n.T("Ablage", "File")`), so bleiben beide
Sprachen beieinander und es gibt keine verwaisten Schlüssel. Das HUD nutzt jetzt die Go-Schrift statt des
ASCII-Debugfonts (Umlaute). Die Inhalte des Lernmodus und die Szenennamen bleiben vorerst einsprachig
(Lerntexte deutsch, Szenen englisch); `CFBundleLocalizations` folgt mit dem Bundle in T-048.

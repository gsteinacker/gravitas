---
id: T-045
title: Lokalisierung Deutsch und Englisch
status: todo
milestone: M7
priority: 2
estimate: M
depends: [T-043]
---

## Ziel

Die App soll Deutsch- und Englischsprachigen gleichermaßen dienen. Menüs, Dialoge, Inspektor und HUD folgen der
Systemsprache.

## Akzeptanzkriterien

- [ ] `internal/i18n` mit Tabelle de/en, Sprache aus `NSLocale.preferredLanguages`, sonst `LANG`
- [ ] Menüs, Dialoge, Inspektor, HUD, Toasts und Hilfe-Overlay laufen über i18n
- [ ] Test: jeder Schlüssel hat beide Sprachen und nutzt nur Glyphen der Go-Schrift
- [ ] `-lang de|en` erzwingt eine Sprache (für Screenshots)

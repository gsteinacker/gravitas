---
id: T-048
title: App-Bundle, Icon, ad-hoc-Signatur und DMG
status: todo
milestone: M7
priority: 1
estimate: M
depends: [T-043]
---

## Ziel

Die App soll sich „wie üblich“ installieren lassen: DMG öffnen, Galaxy in Programme ziehen, starten. Zunächst
ad-hoc signiert, die Pipeline ist für Developer ID und Notarisierung vorbereitet.

## Akzeptanzkriterien

- [ ] `make app` baut `bin/Galaxy.app` als Universal Binary (arm64 + x86_64, `lipo -info`), ad-hoc signiert
- [ ] Info.plist mit Bundle-ID, Version aus `git describe`, Retina, Kategorie, Lokalisierungen de/en, Dokumenttyp `.galaxy`
- [ ] App-Icon (`.icns`), aus dem eigenen Renderer erzeugt
- [ ] Doppelklick auf eine `.galaxy`-Datei öffnet den Stand
- [ ] `make dmg` erzeugt `bin/Galaxy-<version>.dmg` mit Verknüpfung zu Programme
- [ ] `make notarize` vorbereitet (`SIGN_ID`, `notarytool`), README-Abschnitt „Installieren“
- [ ] App startet aus `/Applications` ohne Terminal (Arbeitsverzeichnis `/`), Szenen und Saves funktionieren

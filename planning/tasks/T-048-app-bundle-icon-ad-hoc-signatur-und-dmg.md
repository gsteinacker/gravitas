---
id: T-048
title: App-Bundle, Icon, ad-hoc-Signatur und DMG
status: done
milestone: M7
priority: 1
estimate: M
depends: [T-043]
---

## Ziel

Die App soll sich „wie üblich“ installieren lassen: DMG öffnen, Galaxy in Programme ziehen, starten. Zunächst
ad-hoc signiert, die Pipeline ist für Developer ID und Notarisierung vorbereitet.

## Akzeptanzkriterien

- [x] `make app` baut `bin/Galaxy.app` als Universal Binary (arm64 + x86_64, `lipo -info`), ad-hoc signiert
- [x] Info.plist mit Bundle-ID, Version aus `git describe`, Retina, Kategorie, Lokalisierungen de/en, Dokumenttyp `.galaxy`
- [x] App-Icon (`.icns`), aus dem eigenen Renderer erzeugt
- [x] Doppelklick auf eine `.galaxy`-Datei öffnet den Stand
- [x] `make dmg` erzeugt `bin/Galaxy-<version>.dmg` mit Verknüpfung zu Programme
- [x] `make notarize` vorbereitet (`SIGN_ID`, `notarytool`), README-Abschnitt „Installieren“
- [x] App startet aus `/Applications` ohne Terminal (Arbeitsverzeichnis `/`), Szenen und Saves funktionieren

## Notizen

- Universal Binary (Cross-Build x86_64 per `CC="clang -arch x86_64"`); das Intel-Binary läuft unter Rosetta
  samt Metal.
- DMG 14 MB.
- `spctl` lehnt die ad-hoc-Signatur erwartungsgemäß ab: Installationshinweis im README.
- Geprüft:
  - Start per `open` (Arbeitsverzeichnis `/`);
  - Datei beim Start und bei laufender App öffnen (`open -a … x.galaxy`, Log „open state“);
  - UTI `de.steinacker.galaxy.state` per `mdls`.
- Icon: `make icon` (Screenshot der Spiralgalaxie, `tools/icon`), Ergebnis eingecheckt.

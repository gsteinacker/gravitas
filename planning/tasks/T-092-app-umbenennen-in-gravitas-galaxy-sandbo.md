---
id: T-092
title: App umbenennen in Gravitas – Galaxy Sandbox
status: done
milestone: M10
priority: 2
estimate: M
depends: []
---

## Ziel

Die App heißt für Nutzer „Gravitas – Galaxy Sandbox“, kurz „Gravitas“. „Galaxy“ ist zu allgemein und
beschreibt seit Gas, Kosmos und Sternentstehung nur einen Teil der Simulation. Der Zusatz „Galaxy Sandbox“ grenzt
den Namen von bestehenden Produkten namens „Gravitas“ ab (Recherche 2026-11-03: Steam-Spiel von Galaxy Shark
Studios, „Gravitas Dark Matter“ für Vision Pro, Physik-Spielzeug „Gravitas“ auf itch.io; keine eingetragene Marke
für Software gefunden). Vorhandene Einstellungen und Speicherstände bleiben nutzbar.

## Akzeptanzkriterien

- [x] Info.plist: `CFBundleName` = „Gravitas“, `CFBundleDisplayName` = „Gravitas“; Fenstertitel und App-Menü
      zeigen „Gravitas“, Über-Fenster und Vorspann nennen „Gravitas – Galaxy Sandbox“ (de/en, Screenshot geprüft)
- [x] `make app` erzeugt `bin/Gravitas.app` mit Programm `Contents/MacOS/Gravitas`, `make dmg` erzeugt
      `bin/Gravitas-<Version>.dmg` mit Volume „Gravitas <Version>“; `make notarize` nutzt dieselben Namen
- [x] Neue Bundle-ID `de.steinacker.gravitas`; Entscheidung und Folgen (macOS sieht eine neue App,
      Freigabe in Datenschutz & Sicherheit erneut nötig) in den Notizen festgehalten
- [x] Einstellungen und Stände liegen unter `~/Library/Application Support/Gravitas`; existiert nur der alte
      Ordner `Galaxy`, wird er beim ersten Start übernommen (Test in `internal/prefs` mit temporärem Verzeichnis)
- [x] Speicherstände: neue Endung `.gravitas` (Dokumenttyp „Gravitas State“); `.galaxy`-Dateien lassen sich
      weiter öffnen (per Doppelklick und `-load`), Test für das Laden einer `.galaxy`-Datei
- [x] README, CLAUDE.md und `docs/` nennen den neuen Namen und die neuen Pfade; `grep -rn "Galaxy.app\|Support/Galaxy"`
      findet außer der Migration nichts mehr
- [x] Interne Namen (Go-Modul `galaxy`, `cmd/libgalaxy`, `macos/GalaxySwift`, `Galaxy.metal`) bleiben bewusst
      unverändert; das steht in den Notizen
- [x] `make check` und `make lib-test` grün, App startet aus `bin/Gravitas.app`

## Notizen

- Namensrecherche: keine Rechtsberatung. Vor einer kommerziellen Veröffentlichung TMview/DPMAregister selbst
  prüfen; in Deutschland gilt Werktitelschutz für Software auch ohne Eintragung (§ 5 MarkenG).
- Bundle-ID-Wechsel ist nicht zwingend, verhindert aber, dass Gravitas mit alten Galaxy-Einträgen in
  Launch Services und Gatekeeper vermischt wird.
- Umgesetzt: Bundle-ID `de.steinacker.gravitas`, macOS behandelt Gravitas als neue App (einmalige Freigabe,
  alte `Galaxy.app` kann gelöscht werden). Der Dokumenttyp `de.steinacker.gravitas.state` deklariert beide
  Endungen (`gravitas`, `galaxy`), damit alte Stände per Doppelklick in Gravitas öffnen.
- Übernahme des Ordners (`prefs.Dir`) nur, wenn `Galaxy` `prefs.json` oder `saves` enthält, und nur, wenn
  `Gravitas` noch nicht existiert; sonst bleibt alles, wie es ist. End-to-end mit temporärem `HOME` geprüft.
- Vorspann per Screenshot (de/en) geprüft: Titel GRAVITAS, Unterzeile „Galaxy Sandbox – …“. Fenstertitel und
  Über-Fenster (`orderFrontStandardAboutPanel` mit „Gravitas – Galaxy Sandbox“) zeigt der Screenshot-Modus nicht;
  per Code geprüft.
- ADRs und `docs/perf-log.md` sind historisch und nennen weiter „Galaxy“.

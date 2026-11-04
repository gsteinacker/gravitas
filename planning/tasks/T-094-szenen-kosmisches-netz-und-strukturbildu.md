---
id: T-094
title: Szenen Kosmisches Netz und Strukturbildung entfernen
status: done
milestone: M11
priority: 2
estimate: S
depends: []
---

## Ziel

Die Szenen „Kosmisches Netz“ und „Strukturbildung“ passen nicht gut zur Simulation von Galaxien, Sternen und
Löchern (Wunsch des Nutzers, 2026-11-04). Sie verschwinden aus App, Menü und Lernmodus. Der mitbewegte Modus und
die Komponente `cosmicweb` bleiben im Kern, getestet über Szenendateien in `internal/scene/testdata`.

## Akzeptanzkriterien

- [x] `12-cosmic-web.json` und `13-structure.json` nach `internal/scene/testdata/` (ohne Lernthemen); Szenen 14–16
      heißen jetzt 12–14 (Galaxienentstehung, Satelliten, Balkenspirale)
- [x] Lernthemen `cosmic-web` und `expansion` entfernt
- [x] Tests nutzen die Dateien in `testdata` (`TestCosmicWebMass`, `TestStructureFormation*`,
      `TestCosmicWebDrawsDarkMatterPoints`); `make check` grün
- [x] README, `docs/scenes.md`, `docs/physics.md` angepasst; Verweis „Szene 16“ in der Spiralgalaxie korrigiert
- [x] T-075 und T-076 (Etappe 3 des kosmischen Netzes) auf `blocked` mit Verweis auf diesen Task

## Notizen

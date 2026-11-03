---
id: T-078
title: Alle Szenen erreichbar, nicht nur über die Zifferntasten
status: done
milestone: M10
priority: 1
estimate: M
depends: []
---

## Ziel

Es gibt 13 Szenen, die Tasten reichen nur für 1–9. Der Go-Kern kennt nur `1`–`9`, das Swift-Menü gibt allen
Szenen ab der zehnten die Taste `0` (Konflikt). Im Inspektor steht die Szene nur als Text. Alle Szenen sollen gut
erreichbar und auffindbar sein, auch wenn noch mehr dazukommen.

## Akzeptanzkriterien

- [x] Szenenwahl im Fenster: Auswahl im Inspektor oder Szenenübersicht (Sheet) mit Name, Kurzbeschreibung und
      Gruppe (z. B. Galaxien, Sterne und Löcher, Gas, Kosmos), Gruppen aus dem Szenen-JSON
- [x] Menü „Neu aus Szene“ nach denselben Gruppen gegliedert, keine doppelten Tastenkürzel
- [x] Tasten 1–9 bleiben für die ersten neun Szenen (oder für eine kuratierte Auswahl); Tastenübersicht angepasst
- [x] Test: jede eingebaute Szene ist über die Oberfläche erreichbar, keine Taste doppelt belegt
- [x] Screenshot der Szenenwahl (`.sheet.png` bzw. `.window.png`)

Erledigt 2026-11-03:
- Szenen-JSON hat `group` (galaxies, stars, gas, cosmos). `scene.Catalog()` liefert Name, Gruppe, ersten Satz von
  `intro.school` und Taste. An Swift geht das als `galaxy_text("sceneCatalog")`.
- Szenenübersicht (Sheet) über Ablage → „Szenenübersicht …“ (⇧⌘N) und über den Szenennamen im Inspektor (jetzt ein
  Link). Ein Klick lädt die Szene, die laufende ist hervorgehoben. Screenshot mit `-scenes` (`.sheet.png`).
- Menü „Neu aus Szene“ in Abschnitten nach Gruppen. Tasten 1–9 bleiben bei den ersten neun Szenen, keine doppelt.
  Die Tastenübersicht nennt ⇧⌘N.
- Test `TestCatalogListsEverySceneOnce`.
- Offen: Die Szenennamen sind nur englisch.

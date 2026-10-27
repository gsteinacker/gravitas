---
id: T-078
title: Alle Szenen erreichbar, nicht nur über die Zifferntasten
status: todo
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

- [ ] Szenenwahl im Fenster: Auswahl im Inspektor oder Szenenübersicht (Sheet) mit Name, Kurzbeschreibung und
      Gruppe (z. B. Galaxien, Sterne und Löcher, Gas, Kosmos), Gruppen aus dem Szenen-JSON
- [ ] Menü „Neu aus Szene“ nach denselben Gruppen gegliedert, keine doppelten Tastenkürzel
- [ ] Tasten 1–9 bleiben für die ersten neun Szenen (oder für eine kuratierte Auswahl); Tastenübersicht angepasst
- [ ] Test: jede eingebaute Szene ist über die Oberfläche erreichbar, keine Taste doppelt belegt
- [ ] Screenshot der Szenenwahl (`.sheet.png` bzw. `.window.png`)

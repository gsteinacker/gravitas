---
id: T-082
title: Rotation für Gaswolken und Sternhaufen
status: done
milestone: M9
priority: 2
estimate: M
depends: [T-063]
---

## Ziel

Gesetzte Gaswolken und Sternhaufen sollen sich drehen können, wie echte Molekülwolken und Kugelsternhaufen. Eine
rotierende Wolke kollabiert zur Scheibe. Daran anschließend: eine Szene, in der aus einer rotierenden Gaswolke in
einem Halo aus Dunkler Materie eine Scheibengalaxie entsteht. Außerdem soll heißes Gas (Supernova-Blasen) seinen
Staub verlieren.

## Akzeptanzkriterien

- [x] Regler „Rotation“ für Gaswolke (β = f²/3) und Sternhaufen (Anteil gleichsinniger Sterne, Lynden-Bell) mit (i)
- [x] Szenenbausteine: `rotation` für `gasclump` und `plummer`, neuer Baustein `halo`
- [x] Tests: β der Wolke, Haufen behält die Energie und dreht
- [x] Szene „Galaxy formation“: Kollaps zu einer rotierenden, abgeflachten Scheibe mit Sternentstehung, headless
      gemessen; Lernthema „galaxy-formation“
- [x] Staub verschwindet in heißem Gas (10⁵–10⁶ K), Test, Doku, Lernthema „dust“
- [x] Screenshot-Prüfung

## Notizen

Fragen des Nutzers vom 2026-10-29: „sollten gaswolken und sternenhaufen auch einen drehimpuls haben?“, „müsste aus
einer ausreichend großen sternwolke nicht eine galaxie entstehen können?“, „wie entstehen die staubwolken …
Staub dazunehmen“.

Erledigt 2026-10-29: `internal/scene/rotate.go` (SpinUpCloud, SpinUpCluster), Regler im Werkzeug, Baustein `halo`,
Szene 14 „Galaxy formation“, `engine.dustShare` (Staub verschwindet zwischen 10⁵ und 10⁶ K). Szene 14 bei 8000
Teilchen: Höhe/Radius der Baryonen 1,0 → 0,25–0,4 nach 400–550 Myr, Kantenansicht zeigt eine Scheibe, von oben
eine klumpige Gasscheibe mit Supernova-Blasen. Die Szene wird mit dem dichten Zentrum langsamer (150 → 40
Schritte/s bei 8000 Teilchen).

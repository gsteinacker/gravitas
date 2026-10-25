---
id: T-070
title: Spiralarme und junge blaue Sterne sichtbar machen
status: done
milestone: M10
priority: 2
estimate: L
depends: []
---

## Ziel

Die Spiralarme und die im Lernmodus beschriebene Sternentstehung mit ihrer Blaufärbung sind kaum zu erkennen. Ein
Grund: Eine echte Galaxie hat ~10¹¹ Sterne, die Simulation 10⁴–10⁶ Teilchen; jedes Teilchen steht für eine ganze
Sternpopulation (10⁵–10⁶ M☉). Gesucht ist ein optischer Zusatz, der nach mehr aussieht, ohne viel mehr zu rechnen.

## Akzeptanzkriterien

- [x] Analyse: Wie viele Sterne entstehen in Spirale/Whirlpool/Antennen tatsächlich (Rate, Alter, Ort)? Liegt das
      Problem bei der Physik (zu wenig Sternentstehung in den Armen) oder bei der Darstellung?
- [x] Teilchen als Populationen zeichnen: jedes Sternteilchen erzeugt auf der GPU mehrere schwache Unterpunkte
      (Instancing, deterministische Hash-Versätze innerhalb seiner Glättungslänge), so dass Arme körnig-dicht wirken;
      keine zusätzliche Physik, Kosten nur im Vertex-Shader
- [x] Leuchtkraft nach Alter: junge Populationen (< 10–100 Myr) leuchten pro Masse viel heller und blau (OB-Sterne),
      alte gelblich; Masse-Leuchtkraft realistisch abgeschätzt (L/M jung ≈ 10–100× alt)
- [x] Optional: diffuses Scheibenlicht aus der Dichte (wie der Nebel der Dunklen Materie) und rosa HII-Regionen um
      junge Sterne, damit Arme als helle Bänder mit Staubbahnen erscheinen
- [x] Screenshots Spirale/Whirlpool vorher/nachher: Arme und blaue Sternentstehungsgebiete erkennbar; FPS bei 100k ≥ 60
- [x] Lernmodus-Text prüfen: beschreibt er, was man jetzt sieht (Modellgrenze „Teilchen = Population“ nennen)

## Notizen

Rückmeldung des Nutzers vom 2026-10-25.

Ergebnis (2026-10-26):
- Physik: Die Schwelle `sfDensity` 0,5 lag 5× über der höchsten Gasdichte, deshalb entstand fast nichts. Mit 0,002
  (≈ 0,1 cm⁻³) bildet Szene 1 rund 0,25 M☉/Jahr. Das sind aber nur einige Dutzend Teilchen in 100 Myr, sichtbar wird
  es deshalb erst über die Darstellung.
- Darstellung (`internal/engine/population.go`, `Galaxy.metal`): 8 Sprites pro Körper. Ein Kern, ein Lichthof von
  2,5 ε (aus der Ferne ein Nebel) und 6 Einzelsterne (beim Zoomen aufgelöst). Die Farbe kommt aus dem Alter statt
  aus der Teilchenmasse. Junge Teilchen leuchten ×10. Sternbildendes Gas ist ein bläulicher Schleier (auf Wunsch des
  Nutzers keine Punkte) mit Klumpungsfaktor 5 und L/M = 300 über 10 Myr.
- Der Bloom verkleinert jetzt per Boxfilter statt Einzelabtastung. Das behebt das Flackern kleiner Sterne und die
  Kästchen im Bloom.
- FPS bei 100k im Screenshot-Modus: 58–62 vorher wie nachher (Bildratengrenze). Die zusätzlichen Sprites kosten
  messbar nichts.
- Neue Screenshot-Flags `-zoom` und `-target`.


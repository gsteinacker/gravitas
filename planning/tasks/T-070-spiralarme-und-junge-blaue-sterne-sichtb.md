---
id: T-070
title: Spiralarme und junge blaue Sterne sichtbar machen
status: todo
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

- [ ] Analyse: Wie viele Sterne entstehen in Spirale/Whirlpool/Antennen tatsächlich (Rate, Alter, Ort)? Liegt das
      Problem bei der Physik (zu wenig Sternentstehung in den Armen) oder bei der Darstellung?
- [ ] Teilchen als Populationen zeichnen: jedes Sternteilchen erzeugt auf der GPU mehrere schwache Unterpunkte
      (Instancing, deterministische Hash-Versätze innerhalb seiner Glättungslänge), so dass Arme körnig-dicht wirken;
      keine zusätzliche Physik, Kosten nur im Vertex-Shader
- [ ] Leuchtkraft nach Alter: junge Populationen (< 10–100 Myr) leuchten pro Masse viel heller und blau (OB-Sterne),
      alte gelblich; Masse-Leuchtkraft realistisch abgeschätzt (L/M jung ≈ 10–100× alt)
- [ ] Optional: diffuses Scheibenlicht aus der Dichte (wie der Nebel der Dunklen Materie) und rosa HII-Regionen um
      junge Sterne, damit Arme als helle Bänder mit Staubbahnen erscheinen
- [ ] Screenshots Spirale/Whirlpool vorher/nachher: Arme und blaue Sternentstehungsgebiete erkennbar; FPS bei 100k ≥ 60
- [ ] Lernmodus-Text prüfen: beschreibt er, was man jetzt sieht (Modellgrenze „Teilchen = Population“ nennen)

## Notizen

Rückmeldung des Nutzers vom 2026-10-25.

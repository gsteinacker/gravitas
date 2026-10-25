---
id: T-067
title: Schwarze Löcher aus Kollaps entstehen lassen
status: todo
milestone: M9
priority: 2
estimate: L
depends: [T-063]
---

## Ziel

Wenn sich genug Masse auf engem Raum sammelt, soll ein Schwarzes Loch entstehen, z. B. im Zentrum einer
kollabierenden Gas-/Staubwolke (T-063) oder in einem sehr dichten Sternhaufen. Das macht Strukturbildung erlebbar
und verbindet Gaswolken, Sternentstehung und Schwarze Löcher.

## Akzeptanzkriterien

- [ ] Analyse/ADR: physikalische Wege und was davon im Modell vertretbar ist
      - Hoop-Kriterium: Masse M innerhalb eines Radius r < r_s = 2GM/c² (mit reduziertem c deutlich früher erreicht –
        das macht den Effekt sichtbar, ist aber eine bewusste Übertreibung)
      - Kollaps massereicher Sterne (Sternentstehung aus Gas: Teilchen über einer Massengrenze → Loch nach ihrer
        Lebensdauer, Subgrid)
      - Direkter Kollaps von Gaswolken (frühes Universum, 10⁴–10⁶ M☉) und Runaway-Verschmelzungen in dichten Haufen
- [ ] Erkennung im Integrator/SPH: Dichtespitzen effizient finden (Baum/Nachbarlisten vorhanden), Kriterium
      prüfen, Teilchen zu einem Loch zusammenfassen mit Erhaltung von Masse, Impuls (und bilanzierter Energie wie
      bei Akkretion, ExternalEnergy)
- [ ] Glättung/Auflösung beachten: kein künstliches Loch, nur weil Teilchen numerisch verklumpen (Kriterium auf
      aufgelöster Skala, Mindestteilchenzahl)
- [ ] Sichtbar machen: kurze Hervorhebung/Toast „Ein Schwarzes Loch ist entstanden“, Lernmodus-Hinweis
- [ ] Szene/Startoption zum Ausprobieren (kalte, massereiche Gaswolke bei reduziertem c)
- [ ] Invarianten-Tests (Masse, Impuls, Energie-Bilanz) und Test, dass stabile Szenen keine Löcher erzeugen
- [ ] Lernmodus: Thema „Entstehung Schwarzer Löcher“ in drei Stufen mit Modellgrenzen

## Notizen

Wunsch des Nutzers vom 2026-10-25. Bezug: T-063 (Gaswolken), T-065 (Akkretionsscheiben), T-059 (Kerr).

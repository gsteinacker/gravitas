---
id: T-091
title: Schwarzes Loch wandert aus der Galaxienmitte
status: todo
milestone: M9
priority: 2
estimate: M
depends: []
---

## Ziel

Das zentrale Schwarze Loch der Galaxien soll im Zentrum bleiben, wie ein echtes Loch, das durch dynamische Reibung
in die Mitte sinkt. Heute ist es kaum schwerer als die DM-Teilchen und wird von ihnen herumgestoßen (Brownsche
Bewegung): ein Rechenartefakt.

## Akzeptanzkriterien

- [ ] ADR: Verfahren wählen. Kandidaten sind
      - das Loch auf das Potentialminimum der umgebenden Sterne setzen („repositioning“, wie in EAGLE und
        Illustris),
      - eine zusätzliche dynamische Reibung nach Chandrasekhar unterhalb der Auflösung (Tremmel et al. 2015),
      - schwerere Löcher bzw. feinere Dunkle Materie.
      Wichtig ist der Erhalt von Impuls und Energie (Buchung in `ExternalEnergy`).
- [ ] Gaszustrom (ADR-0016) und Messungen (Balken, Infall-Quelle) beziehen sich danach auf das richtige Zentrum
- [ ] Test: Loch in einer Galaxie mit 20 000 Körpern bleibt über 2 Gyr innerhalb von ~2 Glättungslängen um das
      Dichtezentrum der Sterne
- [ ] Löcher, die frei fliegen sollen (Startrampe, Szene 4, S-Sterne), bleiben unbeeinflusst

## Notizen

Gefunden bei T-090 am 2026-11-03, 20 000 Körper, Loch 2·10⁷ M☉, DM-Teilchen ~9·10⁶ M☉:
- Szene 1: Das Loch steht nach 1,3 Gyr 0,9 Einheiten (2,2 kpc) neben der Mitte.
- Scheibendominierte Variante: 2,1 Einheiten (5 kpc) nach 2,6 Gyr.

Die Balkenmessung zentriert seit T-090 deshalb auf die Sternendichte (schrumpfende Kugeln) statt auf das Loch.

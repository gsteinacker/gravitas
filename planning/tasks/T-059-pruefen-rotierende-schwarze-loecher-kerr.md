---
id: T-059
title: Prüfen: rotierende Schwarze Löcher (Kerr) simulieren?
status: todo
milestone: M9
priority: 2
estimate: M
depends: []
---

## Ziel

Echte Schwarze Löcher rotieren fast immer (Spin a/M bis nahe 1). Heute sind die Löcher nicht rotierend: Paczyński-Wiita
-Potential (ISCO bei 6 GM/c²), Horizont r_s, Linse und Rotverschiebung nach Schwarzschild. Zu prüfen ist, ob und wie
Rotation in das Modell passt und was sie sichtbar machen würde, bevor etwas gebaut wird.

## Akzeptanzkriterien

- [ ] Kurze Analyse (`docs/analysis/`): sichtbare Effekte eines Kerr-Lochs in dieser Simulation (ISCO zwischen
      1 und 9 GM/c² je nach Spin und Umlaufsinn, kleinerer Horizont, Frame-Dragging/Lense-Thirring-Präzession,
      asymmetrischer Schatten, Spin-Änderung bei Akkretion und Verschmelzung)
- [ ] Mögliche Modelle mit Kosten und Grenzen: pseudo-Newtonsche Kerr-Potentiale (z. B. Artemova-Björnsson-Novikov,
      Mukhopadhyay), gravitomagnetische Kraft, nur Darstellung (Schatten/Linse) vs. Dynamik
- [ ] Verträglichkeit mit reduziertem c, Substeps nahe Löchern und Energie-Invarianten (docs/physics.md)
- [ ] Empfehlung mit Aufwand; bei „ja“ Folge-Tasks und ADR-Entwurf
- [ ] Lernmodus: welche Themen/Texte neu nötig wären

## Notizen

Wunsch des Nutzers vom 2026-10-25.

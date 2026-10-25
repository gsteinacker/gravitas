---
id: T-059
title: Prüfen: rotierende Schwarze Löcher (Kerr) simulieren?
status: done
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

- [x] Kurze Analyse (`docs/analysis/`): sichtbare Effekte eines Kerr-Lochs in dieser Simulation (ISCO zwischen
      1 und 9 GM/c² je nach Spin und Umlaufsinn, kleinerer Horizont, Frame-Dragging/Lense-Thirring-Präzession,
      asymmetrischer Schatten, Spin-Änderung bei Akkretion und Verschmelzung)
- [x] Mögliche Modelle mit Kosten und Grenzen: pseudo-Newtonsche Kerr-Potentiale (z. B. Artemova-Björnsson-Novikov,
      Mukhopadhyay), gravitomagnetische Kraft, nur Darstellung (Schatten/Linse) vs. Dynamik
- [x] Verträglichkeit mit reduziertem c, Substeps nahe Löchern und Energie-Invarianten (docs/physics.md)
- [x] Empfehlung mit Aufwand; bei „ja“ Folge-Tasks und ADR-Entwurf
- [x] Lernmodus: welche Themen/Texte neu nötig wären

## Notizen

Wunsch des Nutzers vom 2026-10-25.

Erledigt 2026-10-26: `docs/analysis/03-kerr.md`. Empfehlung: ja, in zwei Stufen. Stufe 1 ist Spin als Zustand und
Darstellung (T-071), Stufe 2 die Lense-Thirring-Kraft mit ADR (T-072). Pseudo-Kerr-Potentiale (ABN, Mukhopadhyay)
unterscheiden pro- und retrograd nicht, Kerr-Geodäten passen nicht zum N-Körper-Kern.


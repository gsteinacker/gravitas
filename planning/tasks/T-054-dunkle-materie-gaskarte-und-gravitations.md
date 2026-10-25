---
id: T-054
title: Dunkle Materie, Gaskarte und Gravitationsrotverschiebung in Metal
status: done
milestone: M8
priority: 2
estimate: M
depends: [T-052]
---

## Ziel

Die übrigen Effekte der Ebitengine-App: Dunkle Materie als violetter Schleier, Gas als Falschfarbenkarte,
Gravitationsrotverschiebung nahe Löchern im Doppler-Effekt.

## Akzeptanzkriterien

- [x] Haze-Sprites für Dunkle Materie und Gas (Farben wie `render`), Deckkraft aus dem Go-Kern
- [x] Doppler mit √(1 − r_s/r) aller Löcher (bis 16)
- [x] Schalter in Menü und Inspektor; Screenshots Lokale Gruppe (DM) und Pferdekopf (Gaskarte)

## Notizen

Haze-Deckkraft und Lochliste (Weltkoordinaten, r_s) aus `engine.Prepare`. Nebenbefund: ohne Leuchten las der
Schlusspass die nie gezeichnete Bloom-Textur (0 · NaN = NaN, Bild fiel bis auf Grün aus); jetzt nur bei Glow.
Screenshots: Lokale Gruppe mit DM-Schleier, Pferdekopf-Gaskarte wie in der Go-Version, Doppel-Loch mit Doppler.
`-fx` als Startargument (Teil von T-056).

---
id: T-030
title: Sterne als GPU-Punkte rendern statt CPU-Quads
status: todo
milestone: M2
priority: 3
estimate: M
depends: [T-023]
---

## Ziel

Bei 1M erzeugt die CPU pro Frame 4M Vertices, und Ebitengine verarbeitet sie (mipmap, DrawTriangles). Stattdessen Sternpositionen als Daten an einen Shader geben (z. B. Positionen in eine Textur, Quads im Vertex-Pfad) oder Metal direkt nutzen.

## Akzeptanzkriterien

- [ ] ≥ 30 FPS bei 1M mit allen Effekten
- [ ] Optik unverändert (Screenshot-Vergleich bei 100k)

## Notizen

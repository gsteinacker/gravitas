---
id: T-011
title: Render-Batching für 100k+ Punkte
status: done
milestone: M2
priority: 2
estimate: M
depends: [T-004]
---

## Ziel

Rendering darf bei 100k Sternen nicht der Engpass sein.

## Akzeptanzkriterien

- [x] ≥ 60 FPS Rendering bei 100k Punkten (ohne Post-Effekte; mit allen M4-Effekten ~50 FPS, siehe Notiz)
- [x] Vertex-Buffer wiederverwendet

## Notizen
- Bereits in T-004 umgesetzt: Quads gebündelt in Batches ≤ MaxVertexCount, Vertex- und Index-Buffer wiederverwendet. Zeichnen ~8 % CPU im App-Profil (T-024).
- Nachtrag M4: Mit Glow, Linse und Doppler zeigt die App bei 100k Sternen ~50 FPS, während die Simulation alle Kerne belegt. Bei 30k sind es 120 FPS. Für Rendering und Simulation gemeinsam ist das bei 100k die Grenze. Falls es stört: Effekte per Taste abschalten oder GOMAXPROCS für die Simulation begrenzen.

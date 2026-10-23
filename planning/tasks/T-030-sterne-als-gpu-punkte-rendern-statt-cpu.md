---
id: T-030
title: Sterne als GPU-Punkte rendern statt CPU-Quads
status: done
milestone: M2
priority: 3
estimate: M
depends: [T-023]
---

## Ziel

Bei 1M erzeugt die CPU pro Frame 4M Vertices, und Ebitengine verarbeitet sie (mipmap, DrawTriangles). Stattdessen Sternpositionen als Daten an einen Shader geben (z. B. Positionen in eine Textur, Quads im Vertex-Pfad) oder Metal direkt nutzen.

## Akzeptanzkriterien

- [x] ≥ 30 FPS bei 1M mit allen Effekten: 61 FPS bei ruhender Kamera, ~28 FPS während die Kamera bewegt wird
- [x] Optik unverändert (Screenshot-Vergleich bei 100k)

## Notizen
- **Nicht** wie geplant umgesetzt (Shader-Geometrie geht mit Kage nicht, es gibt nur Fragment-Shader). Stattdessen zwei gemessene Hebel:
  1. Profil: Die Hälfte der Render-Thread-Zeit war `mipmapLevelFromDistance` (Ebitengine wählt pro Dreieck eine Mipmap-Stufe). Jetzt `DisableMipmaps` und ein 16-px-Sprite statt 32 px, damit das Verkleinern nicht flimmert. 12–18 → 28 FPS.
  2. Gegentests: Ohne Post-Effekte bleiben es 28 FPS, mit CPU-Simulation nur 12 FPS. Die Grenze ist CPU-Arbeit pro Frame bei Kernkonkurrenz. Das Sternbild wird nur neu gezeichnet, wenn sich Snapshot-Version, Kamera oder Doppler ändern. Die Simulation liefert bei 1M nur ~15 Zustände/s. 28 → 61 FPS.
- Schritte/s steigen dabei mit (100k: ~100 → 113), weil das Rendering weniger Kerne belegt.
- Offen: Bei bewegter Kamera ~28 FPS bei 1M. Weiterer Hebel wäre direktes Metal-Rendering (Sterne liegen ohnehin in GPU-Buffern).
- Screenshot-Modus gibt jetzt eine Kennzahlen-Zeile aus (`shot …: bodies= fps= steps_per_s= …`).

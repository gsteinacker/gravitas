---
id: T-016
title: Glow, additive Blending, Farbe nach Masse
status: done
milestone: M4
priority: 2
estimate: M
depends: [T-011]
---

## Ziel

Es soll gut aussehen.

## Akzeptanzkriterien

- [x] Additives Blending (seit T-004)
- [x] Farbe/Größe nach Masse (seit T-004)
- [x] Bloom-Shader (Kage): 1/4 herunterskaliert, separabler 9-Tap-Gauß, additiv zurück

## Notizen
- `render/post.go`: Pipeline Sterne → (Spuren) → über opaken Sternhimmel → (Linse) → Bildschirm → (+ Bloom) → Löcher. Taste `g`, Start mit `-fx`.
- Gefundener Fehler beim Bau: Der Sternhimmel war transparent, das Linsenbild akkumulierte dadurch über Frames (ungewollte Spuren bis zur Sättigung). Jetzt ist er opak, und Shader-Ausgaben nutzen `BlendCopy`.

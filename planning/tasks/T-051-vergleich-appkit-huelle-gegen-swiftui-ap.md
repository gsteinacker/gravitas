---
id: T-051
title: Vergleich AppKit-Hülle gegen SwiftUI-App
status: done
milestone: M7
priority: 3
estimate: S
depends: [T-050, T-048]
---

## Ziel

Beide Varianten sollen sich nebeneinander installieren und objektiv vergleichen lassen, damit entschieden werden kann,
welche auf `main` weiterlebt.

## Akzeptanzkriterien

- [x] Beide Apps geben dieselbe Messzeile aus (FPS, steps/s, Profil)
- [x] `docs/analysis/02-mac-ui-vergleich.md`: Leistung, Startzeit, Bundle-Größe, Aufwand, Bediengefühl, Empfehlung

## Notizen

Ergebnis in `docs/analysis/02-mac-ui-vergleich.md`: gleiche Bildrate, B 5–10 % mehr Schritte/s; B deutlich
nativer (Seitenleiste, Inspektor, Bedienungshilfen), A vollständiger (Staub, Karten) und fertig paketiert.
Empfehlung: A jetzt ausliefern, mittelfristig B nach Port von Staub/Karten. Nebenbefund: relative Pfade im
gebündelten Binary (GLFW wechselt nach Contents/Resources), behoben auf `main`.

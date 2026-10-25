---
id: T-053
title: Staub in der SwiftUI-App
status: todo
milestone: M8
priority: 1
estimate: L
depends: [T-052]
---

## Ziel

Staub fehlt in der SwiftUI-App: dunkle Staubbänder, Rötung und die Silhouette des Pferdekopfnebels. Port von
`render/dust.go`: Staubkarte im Go-Kern, Abschwächung pro Stern und Emission, Staubbild für den Hintergrund.

## Akzeptanzkriterien

- [ ] Staubkarte (Pyramide, Tiefenmomente, automatische Opazität) im Go-Kern, Abschwächung pro Körper in einen
      Aufrufer-Puffer, ohne Allokation pro Bild
- [ ] Shader: Sterne und Hα abgeschwächt, Hintergrund über Staubbild mit exp(−τ k) pro Farbkanal
- [ ] Pferdekopf zeigt die dunkle Silhouette vor IC 434 (Screenshot), Spiralgalaxie Staubbänder
- [ ] Schalter „Staub“ in Menü und Inspektor

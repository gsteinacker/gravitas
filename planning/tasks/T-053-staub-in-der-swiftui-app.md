---
id: T-053
title: Staub in der SwiftUI-App
status: done
milestone: M8
priority: 1
estimate: L
depends: [T-052]
---

## Ziel

Staub fehlt in der SwiftUI-App: dunkle Staubbänder, Rötung und die Silhouette des Pferdekopfnebels. Port von
`render/dust.go`: Staubkarte im Go-Kern, Abschwächung pro Stern und Emission, Staubbild für den Hintergrund.

## Akzeptanzkriterien

- [x] Staubkarte (Pyramide, Tiefenmomente, automatische Opazität) im Go-Kern, Abschwächung pro Körper in einen
      Aufrufer-Puffer, ohne Allokation pro Bild
- [x] Shader: Sterne und Hα abgeschwächt, Hintergrund über Staubbild mit exp(−τ k) pro Farbkanal
- [x] Pferdekopf zeigt die dunkle Silhouette vor IC 434 (Screenshot), Spiralgalaxie Staubbänder
- [x] Schalter „Staub“ in Menü und Inspektor

## Notizen

Port von `render/dust.go` nach `internal/engine/dust.go`. Abschwächung pro Körper in einen Ringpuffer der App (3 Stück,
pro Bild gefüllt), Staubbild auf der GPU aus den Gas-Sprites. Auch die Belichtung sieht jetzt das abgeschwächte
Licht (wie `render`). Test `TestDustAttenuatesStarsBehindGas`; Screenshots Pferdekopf (Silhouette) und
Spiralgalaxie (Staubbänder).

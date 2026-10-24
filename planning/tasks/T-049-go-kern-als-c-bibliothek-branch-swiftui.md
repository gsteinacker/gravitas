---
id: T-049
title: Go-Kern als C-Bibliothek (Branch swiftui-app)
status: todo
milestone: M7
priority: 2
estimate: M
depends: [T-042]
---

## Ziel

Für die SwiftUI-Variante muss der Go-Kern ohne Ebitengine als Bibliothek nutzbar sein: Szene, Runner, Aktionen und
Snapshot über eine C-Schnittstelle.

## Akzeptanzkriterien

- [ ] Branch `swiftui-app`
- [ ] `internal/engine` ohne Ebitengine: Controller, Snapshot in Aufrufer-Puffer (float32), keine Allokation pro Frame
- [ ] `cmd/libgalaxy` mit `-buildmode=c-archive`, Header generiert, C-Test ruft new/action/snapshot auf
- [ ] Grafikfreie Render-Mathematik (Kamera, Doppler) in eigenem Paket, von beiden Varianten genutzt

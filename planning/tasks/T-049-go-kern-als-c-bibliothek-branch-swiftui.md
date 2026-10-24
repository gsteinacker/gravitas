---
id: T-049
title: Go-Kern als C-Bibliothek (Branch swiftui-app)
status: done
milestone: M7
priority: 2
estimate: M
depends: [T-042]
---

## Ziel

Für die SwiftUI-Variante muss der Go-Kern ohne Ebitengine als Bibliothek nutzbar sein: Szene, Runner, Aktionen und
Snapshot über eine C-Schnittstelle.

## Akzeptanzkriterien

- [x] Branch `swiftui-app`
- [x] `internal/engine` ohne Ebitengine: Controller, Snapshot in Aufrufer-Puffer (float32), keine Allokation pro Frame
- [x] `cmd/libgalaxy` mit `-buildmode=c-archive`, Header generiert, C-Test ruft new/action/snapshot auf
- [x] Grafikfreie Render-Mathematik: Kamera in `internal/view` (Go); Farben/Doppler werden in den Metal-Shader portiert (MSL), siehe T-050

## Notizen

C-API: `galaxy_init/tick/frame/state/action/set/text/open/save` und Kamera/Klick. Frame: 12 float32 pro Körper.
Smoke-Test in C: `make lib-test`. Die Engine kalibriert wie die App (`perf`) und nutzt denselben Regler; die
Korrektur über gemessene Schrittzeiten und „Leistung neu messen“ fehlen in dieser Variante.

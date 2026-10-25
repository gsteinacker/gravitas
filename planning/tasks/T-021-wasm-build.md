---
id: T-021
title: WebAssembly-Build
status: blocked
milestone: M5
priority: 3
estimate: S
depends: [T-006]
---

## Ziel

Im Browser teilbar.

## Akzeptanzkriterien

- [ ] `make wasm` erzeugt lauffähige Seite

## Notizen

Blockiert seit 2026-10-26 durch ADR-0011: Die Oberfläche ist SwiftUI + Metal, der Go-Kern läuft als C-Bibliothek.
Ein Browser-Build bräuchte einen eigenen WebGPU-Renderer und eine eigene Oberfläche, also einen zweiten Stack.
Nur auf ausdrücklichen Wunsch wieder aufnehmen (dann mit eigenem ADR).


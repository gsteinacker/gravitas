---
id: T-058
title: SwiftUI wird Hauptapp, Ebitengine- und AppKit-Code löschen
status: done
milestone: M8
priority: 1
estimate: M
depends: [T-053, T-054, T-055, T-057]
---

## Ziel

Entscheidung des Nutzers (2026-10-25): Nur die SwiftUI-Oberfläche wird weiterverfolgt. Sobald sie vollständig ist,
werden Ebitengine-App und AppKit-Hülle gelöscht.

## Akzeptanzkriterien

- [x] ADR-0011: SwiftUI als einzige Oberfläche (ersetzt Teile von ADR-0001, ADR-0009)
- [x] Branch `swiftui-app` in `main`, Bundle „Galaxy“ mit ID `de.steinacker.galaxy`
- [x] Gelöscht: `cmd/galaxy` (Ebitengine), `internal/ui`, `internal/render`, `internal/mac`; Ebitengine aus go.mod
- [x] CLAUDE.md, README, architecture.md auf den neuen Stand; `make check` grün

## Notizen

Gelöscht: `cmd/galaxy`, `internal/ui`, `internal/render`, `internal/mac`. `go.mod`: nur noch `x/image` (Icon-Werkzeug).
Makefile: `build`/`run`/`shot`/`app`/`dmg`/`notarize`/`icon` beziehen sich auf die SwiftUI-App. ADR-0011, Status von
ADR-0001/0008/0009/0010 angepasst, CLAUDE.md/README/architecture.md/scenes.md aktualisiert.

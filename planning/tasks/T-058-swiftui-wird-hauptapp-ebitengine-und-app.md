---
id: T-058
title: SwiftUI wird Hauptapp, Ebitengine- und AppKit-Code löschen
status: todo
milestone: M8
priority: 1
estimate: M
depends: [T-053, T-054, T-055, T-057]
---

## Ziel

Entscheidung des Nutzers (2026-10-25): Nur die SwiftUI-Oberfläche wird weiterverfolgt. Sobald sie vollständig ist,
werden Ebitengine-App und AppKit-Hülle gelöscht.

## Akzeptanzkriterien

- [ ] ADR-0011: SwiftUI als einzige Oberfläche (ersetzt Teile von ADR-0001, ADR-0009)
- [ ] Branch `swiftui-app` in `main`, Bundle „Galaxy“ mit ID `de.steinacker.galaxy`
- [ ] Gelöscht: `cmd/galaxy` (Ebitengine), `internal/ui`, `internal/render`, `internal/mac`; Ebitengine aus go.mod
- [ ] CLAUDE.md, README, architecture.md auf den neuen Stand; `make check` grün

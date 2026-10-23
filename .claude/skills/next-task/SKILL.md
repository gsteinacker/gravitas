---
name: next-task
description: Pick the next ready backlog task (or continue the one in progress), implement it end-to-end against its acceptance criteria, and mark it done. Use when asked to "continue", "work on the next task", or a T-NNN id.
---

# Nächsten Task bearbeiten

1. `go run ./tools/plan next` ausführen. Ist ein Task `doing`, diesen fortsetzen. Sonst den angezeigten
   „Next“-Task nehmen (oder die vom Nutzer genannte ID mit `plan show`).
2. Task-Datei vollständig lesen, dazu die verlinkten Docs (`docs/physics.md`, `docs/architecture.md`, ADRs).
3. `go run ./tools/plan set <ID> doing`.
4. Implementieren in kleinen Schritten. Zuerst Tests für die Akzeptanzkriterien, dann Code.
   Physik: passende Invariante aus `docs/physics.md` testen. Performance: `/bench` vorher/nachher.
5. `make check` muss grün sein, bei Parallelität zusätzlich `make race`.
6. Checkboxen in der Task-Datei abhaken, Erkenntnisse unter „Notizen“ festhalten.
   Neue Folgearbeit nicht still erledigen, sondern als Task anlegen (`/new-task`).
7. `go run ./tools/plan set <ID> done`, dann commit `T-NNN: <summary>`, sofern der Nutzer Commits freigegeben hat.
8. Kurz berichten: was erledigt ist, Messwerte, offene Punkte, nächster bereiter Task.

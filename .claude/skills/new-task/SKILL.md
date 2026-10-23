---
name: new-task
description: Create a new backlog task in planning/tasks with goal and acceptance criteria. Use when new work, a follow-up, or a bug is discovered.
---

# Task anlegen

1. Meilenstein aus `planning/ROADMAP.md` wählen, Abhängigkeiten über `go run ./tools/plan list` bestimmen.
2. `go run ./tools/plan new -m <MS> -p <1-3> -e <S|M|L> [-d T-001,T-002] <Titel>`. Die Ausgabe ist der Dateipfad.
3. In der Datei „Ziel“ (1–3 Sätze, warum) und **prüfbare** Akzeptanzkriterien als Checkboxen ausfüllen.
   Messbar formulieren („≥ 30 Schritte/s bei N=100k“, nicht „schnell“).
4. `go run ./tools/plan check`.

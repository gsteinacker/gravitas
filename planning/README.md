# Planung

Dateibasiertes Backlog: **eine Markdown-Datei pro Task** in `tasks/`, Name `T-NNN-slug.md`.

```markdown
---
id: T-012
title: Geschwindigkeitslimit über Eigengeschwindigkeit
status: todo          # todo | doing | blocked | done
milestone: M3         # siehe ROADMAP.md
priority: 1           # 1 hoch … 3 niedrig
estimate: S           # S (< 2h) | M (halber Tag) | L (> 1 Tag)
depends: [T-003]
---
## Ziel / ## Akzeptanzkriterien (Checkboxen) / ## Notizen
```

## Werkzeug (`go run ./tools/plan …`)

| Befehl | Zweck |
|--------|-------|
| `board` | Fortschritt je Meilenstein (`make plan`) |
| `next` | laufende Tasks und den nächsten bereiten Task (`make next`) |
| `list [-s doing] [-m M2]` | Tabelle |
| `show T-012` | Task anzeigen |
| `set [-force] T-012 doing` | Status ändern, prüft Abhängigkeiten |
| `new -m M3 -p 2 -e S -d T-012 Titel …` | neuen Task anlegen (nächste freie ID) |
| `check` | Validierung (IDs, Felder, Abhängigkeiten, Zyklen), Teil von `make check` |

„Nächster Task“ = Status `todo`, alle Abhängigkeiten `done`, sortiert nach Meilenstein, Priorität, ID.

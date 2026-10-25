# ADR-0008: Go-Schriften für Lerntexte

- Status: accepted (Oberfläche ersetzt durch ADR-0011)
- Datum: 2026-10-24
- Tasks: T-039

## Kontext
Der Lernmodus zeigt längere deutsche Texte mit Umlauten, griechischen Buchstaben und Hoch-/Tiefstellungen. Die
Debug-Schrift von Ebitengine kann nur Latin-1 und ist für Fließtext schlecht lesbar.

## Entscheidung
`golang.org/x/image/font/gofont` (Go Regular/Bold) über `ebiten/v2/text/v2`. Das Modul war bereits indirekt über
Ebitengine eingebunden und wird jetzt direkt genutzt. `TestLearningTextsUseOnlyAvailableGlyphs` prüft jedes Zeichen
aller Lerntexte gegen die Schrift. Fehlende Zeichen (z. B. ∝, ∇, ☉) werden in den Texten umschrieben.

## Alternativen
- Eigene Schrift (z. B. Noto) einbetten: größere Abdeckung, aber eine weitere Lizenz und mehrere MB im Binary.
- Formeln als Bilder: aufwendig und schlecht pflegbar.

## Konsequenzen
- HUD bleibt vorerst in der Debug-Schrift (Englisch). Lerntexte sind Deutsch.

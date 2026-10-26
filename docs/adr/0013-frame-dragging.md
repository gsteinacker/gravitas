# ADR-0013: Gravitomagnetische Kraft rotierender Löcher (Frame-Dragging)

- Status: accepted
- Datum: 2026-10-26
- Tasks: T-072 (Analyse: T-059, `docs/analysis/03-kerr.md`)

## Kontext
Seit T-071 tragen Löcher einen Spin. Er verändert aber nur die Darstellung (Horizont, ISCO-Ringe), die Bahnen
bleiben Paczyński-Wiita. Der auffälligste dynamische Kerr-Effekt fehlt damit: Der Raum wird mitgezogen,
Bahnebenen präzedieren, und pro- und retrograde Bahnen verhalten sich verschieden.

## Entscheidung
Zusätzlich zu Paczyński-Wiita wirkt auf jeden Körper im Feld eines rotierenden Lochs die Lense-Thirring-
Beschleunigung in Ordnung 1/c²:

    a = v × B,   B = (2G / (c² r³)) (J − 3 (J·r̂) r̂)

- **Integration:** als Boris-Rotation der Geschwindigkeit in den Kicks des RESPA-Schemas („kick, rotate …
  rotate, kick“). Das hält |u| exakt und ist zeitumkehrbar. Ferne Körper bekommen die Rotation in den äußeren
  Kicks, nahe in den Unterschritten.
- **Rückwirkung:** Das Loch erhält den entgegengesetzten Impuls und, über seinen Spin, den entgegengesetzten
  Drehimpuls. Impuls und Gesamtdrehimpuls (Bahnen plus Spins) bleiben erhalten.
- Ohne Spin oder ohne c entfällt alles (ein Test pro Kick über die Löcher).

## Alternativen
- **Pseudo-Kerr-Potentiale** (Artemova-Björnsson-Novikov, Mukhopadhyay): Sie sind kugelsymmetrisch und
  unterscheiden pro- und retrograd nicht.
- **Kerr-Geodäten:** Nur für Testteilchen exakt und nicht mit N-Körper- und Mehrlochdynamik vereinbar.

## Konsequenzen
- Tests: Knotenpräzession einer geneigten Kreisbahn gegen Ω_LT = 2GJ/(c²a³) (gemessen 0,527 rad statt 0,540 rad
  über 300 Zeiteinheiten). Ohne Spin keine Präzession. Erhaltung von Impuls und Gesamtdrehimpuls; |u| bleibt
  bei der Rotation gleich.
- Modellgrenze: nur erste Ordnung. Die ISCO verschiebt sich in die richtige Richtung, erreicht aber nicht
  1 bzw. 9 r_g. Keine Retardierung des Gravitomagnetismus, keine Ergosphäre.
- Kosten: +0,7 % pro Schritt (Szene 4, 50k, beide Löcher mit a* = 0,9), ohne Spin praktisch null
  (`docs/perf-log.md`).

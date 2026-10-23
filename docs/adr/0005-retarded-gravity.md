# ADR-0005: Retardierte Gravitation Schwarzer Löcher als optionaler Modus

- Status: accepted (experimentell, Standard bleibt instantan)
- Datum: 2026-10-23
- Tasks: T-020 (Spike)

## Kontext
Gravitation breitet sich mit c aus. ADR-0003 hat dafür einen Spike vorgesehen, beschränkt auf Schwarze Löcher
als Quellen, weil N×N Lichtkegel nicht bezahlbar sind. Bekannt ist: Reine Retardierung (Kraft zeigt auf die
retardierte Position) erzeugt Aberration, also ein Drehmoment auf Bahnen. Laplace folgerte daraus 1805, die
Gravitation müsse fast instantan sein. In der ART heben geschwindigkeitsabhängige Terme die Aberration
weitgehend auf: Das Feld einer gleichförmig bewegten Quelle zeigt auf ihre aus der retardierten Lage
extrapolierte aktuelle Position (Carlip 2000).

## Entscheidung
Drei Modi in `Params.Gravity`, Standard `instant`, umschaltbar mit `p` und `-gravity`:
- `instant`: wie bisher.
- `retarded`: Die Quelle wirkt von ihrer retardierten Position (Lichtkegel |x − x_h(t_r)| = c(t − t_r),
  Fixpunkt-Iteration bis 1e-7 aus einem Ringpuffer mit 8192 Schritten pro Loch).
- `retarded+v`: retardierte Position plus v(t_r)·Δt, so hebt sich die Aberration in erster Ordnung auf.
Nur Kräfte *von* Löchern werden retardiert. Löcher, die während des Laufs gesetzt werden, wirken erst, wenn
ihre Gravitationsfront angekommen ist (Kugelwelle mit c). Löcher einer Szene gelten als schon immer
vorhanden (lineare Vorgeschichte).

## Messungen
Doppel-Loch, v ≈ 0,1c, ωΔt ≈ 0,2, Abstandsänderung:

| | nach 1 Umlauf | nach 20 Umläufen |
|---|---|---|
| instant | +0,5 % (Konstante aus Anfangsbedingung) | +0,5 % |
| retarded | **+57 %** | +673 % (zerfallen) |
| retarded+v | +4,3 % | +46 % (~2 %/Umlauf) |

Lichtkegel und Extrapolation gegen exakte gleichförmige Bewegung: Abweichung < 10⁻⁶. Die Front eines neuen
Lochs kommt nach d/c an (Test). Kosten bei 100k (Kollision, 2 Löcher): 86,6 → 66,4 Schritte/s (−23 %).

## Bewertung
- Naive Retardierung ist für Spielzwecke unbrauchbar. Gebundene Systeme zerfallen innerhalb eines Umlaufs.
  Das ist ein schöner Lehr-Effekt („warum Gravitation nicht einfach verzögert sein kann“).
- `retarded+v` unterdrückt die Scheindrift um etwa (ωΔt)² ≈ 1/25, entfernt sie aber nicht, denn die übrigen
  1PN-Terme fehlen. Bei typischen Lochgeschwindigkeiten in Galaxien (≲ 0,05c) ist sie klein, bei engen,
  schnellen Doppellöchern sichtbar.
- Der eigentliche Gewinn ist die **sichtbare Ausbreitung**: Ein neues Loch „schaltet“ seine Umgebung von innen
  nach außen ein.
- Energie und Impuls sind in den retardierten Modi nicht erhalten (das Feld würde sie tragen). Das HUD
  weist darauf hin.

## Alternativen
- Volle 1PN-Bewegungsgleichungen (Einstein-Infeld-Hoffmann) für Loch-Paare: physikalisch richtig, deutlich
  mehr Aufwand, verlangt eigene Integration. Möglicher Folge-Task, wenn Doppellöcher realistisch werden sollen.
- Retardierung für alle Paare: O(N²) Lichtkegel, nicht bezahlbar.

## Konsequenzen
- `State.ID` (stabile Identität über Compact und Verschmelzung) ist neu und auch für künftige Features nutzbar
  (z. B. Spuren einzelner Körper, Speichern und Laden).

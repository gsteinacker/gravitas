---
id: T-071
title: Kerr Stufe 1: Spin, Horizont, ISCO-Ringe, Spin-Entwicklung
status: todo
milestone: M9
priority: 2
estimate: M
depends: [T-059]
---

## Ziel

Schwarze Löcher bekommen einen Spin (Modell A aus `docs/analysis/03-kerr.md`). Sichtbar werden ein kleinerer
Horizont, die ISCO-Ringe für pro- und retrograde Bahnen und ein D-förmiger Photonenring. Der Spin ändert sich durch
Akkretion und Verschmelzung. Die Dynamik bleibt dabei unverändert.

## Akzeptanzkriterien

- [ ] Spinvektor J pro Loch im State, in Speicherständen und im Szenen-JSON (`spin` als a* plus Achse)
- [ ] Akkretion: J += L des verschluckten Körpers relativ zum Loch; Verschmelzung J₁ + J₂ + L_Bahn; a* ≤ 0,998
      (Test der Drehimpulsbilanz)
- [ ] Darstellung: Horizont r₊ = r_g(1 + √(1 − a*²)), ISCO-Ringe nach Bardeen et al. 1972, D-förmiger Photonenring
      je nach Blickwinkel; Screenshot bei c = 10 mit a* = 0 und 0,9
- [ ] Werkzeug „Schwarzes Loch“ mit Spin-Regler; Inspektor zeigt a*
- [ ] Lernmodus-Thema „Rotierende Schwarze Löcher (Kerr)“, Thema „isco“ ergänzt, (i)-Erklärung des Reglers

## Notizen

Aus der Analyse T-059.

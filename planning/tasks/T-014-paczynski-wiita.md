---
id: T-014
title: Paczyński-Wiita-Potential für BH-Quellen
status: done
milestone: M3
priority: 2
estimate: M
depends: [T-013]
---

## Ziel

Realistischere Bahnen nahe dem Horizont (ISCO bei 3 r_s).

## Akzeptanzkriterien

- [x] Test: Kreisbahn außerhalb der ISCO stabil (6 r_s), innerhalb stürzt sie (2 r_s), Newtonsche Kontrolle bleibt gebunden

## Notizen
- Kriterium präzisiert: „marginal stabil bei genau 3 r_s“ gilt für PW mit Newtonscher Kinematik. Mit unserer relativistischen Kinematik (u = γv) verschiebt sich die ISCO etwas, daher wird mit Abstand zur Grenze getestet.
- Symmetrisches Paarpotential (Impulserhaltung). Zuerst hatte ich PW nur für die Kraft auf Sterne vorgesehen, das hätte actio = reactio verletzt.
- Dabei gefundener Fehler aus T-013: Ohne Neuberechnung der Beschleunigungen nach einer Absorption wird der Impuls verletzt. Behoben, mit Regressionstest (schlägt ohne Fix fehl).
- Kraft = numerischer Gradient des Potentials, auch im gekappten Bereich (`TestPairForceIsGradientOfPotential`). Energie-Drift der PW-Bahn ~1e-6, beschränkt.
- ISCO-Ring (3 r_s) wird gezeichnet. Kosten +2,5 %, relativ gemessen, weil die App parallel lief.

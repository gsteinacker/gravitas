# ADR-0016: Gaszufluss aus dem Halo und Kalibrierung der Sternentstehung

- Status: accepted
- Datum: 2026-10-29
- Tasks: T-081

## Kontext
Nach T-080 verbrauchen die Galaxienszenen ihr Gas, und die Sternentstehung sinkt. Sie war außerdem etwa zehnmal
niedriger als in der Milchstraße. Echte Scheibengalaxien bilden seit 10 Gyr Sterne, obwohl ihr Gas nur für
2–3 Gyr reicht. Sie bekommen frisches Gas, etwa 1 M☉/Jahr für die Milchstraße (Fraternali & Binney 2008,
Putman et al. 2012). Der Nutzer möchte, dass dauerhaft Sterne entstehen.

## Entscheidung
Jede Galaxie einer Szene bekommt eine Zuflussquelle mit vorgegebener Rate (`infallMsunPerYear`). Sie erzeugt neue
Gasteilchen über und unter der Scheibe, die auf sie herabregnen. Die Effizienz der Sternentstehung steigt auf
ε_ff = 0,066.

- Rate ≈ M_gas / 3 Gyr je Galaxie (Badewannenmodell, Zufluss ≈ SFR im Gleichgewicht, Bouché et al. 2010).
- Neue Teilchen sind so schwer wie die vorhandenen Gasteilchen der Galaxie. Sie starten in 1,5 kpc Höhe, wo
  extraplanares H I liegt, bei einem Radius aus dem exponentiellen Gasprofil (bis `gasMax`). Sie rotieren mit
  0,8 v_c, wie das nachhinkende Halogas (Fraternali & Binney 2006), und haben 10⁴ K.
- Die Ebene ist die Richtung des Drehimpulses von Sternen und Gas um das Loch der Galaxie. So funktioniert es auch
  bei geneigten, kollidierenden und verschmolzenen Galaxien. Wird das Loch verschluckt, folgt die Quelle dem
  schwersten Loch.
- Die Energie jedes neuen Teilchens (kinetisch, thermisch, potentiell im Paarmodell) wird in `ExternalEnergy`
  verbucht, wie bei der Akkretion. `Epoch` bleibt unverändert, die Drift-Anzeige misst weiter.
- Reproduzierbar: Zufall aus Quelle, Schritt und Index; angesammelte Restmasse je Quelle wird gespeichert.
- ε_ff = 0,066: An der Schwelle 0,1 cm⁻³ (t_ff ≈ 140 Myr) entspricht das t* = 2,1 Gyr von Springel & Hernquist
  (2003). Gemessen sind es einige Gyr Gasverbrauchszeit, wie bei Kennicutt–Schmidt.

## Alternativen
- **Heißer Gashalo, der abkühlt** (Korona aus SPH-Teilchen): physikalisch näher, aber mit der festen Kühlzeit
  (ohne Λ(T, ρ)) würde der ganze Halo in einer Freifallzeit auf einmal kollabieren. Außerdem kostet er viele
  Teilchen.
- **Teilchen recyceln** (Gasvorrat konstant halten, verbrauchtes Gas wieder einsetzen): Damit bliebe die
  Teilchenzahl fest, aber Sterne verschwänden oder Masse entstünde ohne Herkunft. Neue Teilchen sind ehrlicher, und
  der Zuwachs ist klein (+0,7 % Teilchen pro Gyr in Szene 1).
- **Kosmologischer Zufluss aus Filamenten**: Er gehört zum kosmischen Netz (T-073 ff.) und braucht ein Volumen von
  Mpc-Größe.

## Konsequenzen
- Tests: `TestInfallAddsGasAtRateInThePlaneAndBooksEnergy` (Rate, Lage, Drehsinn, Energiebilanz je Ereignis),
  `TestInfallFollowsTheHeaviestHoleAfterAMerger`.
- Szene 1 über 1 Gyr bei 20 000 Teilchen: Zufluss 0,60 M☉/Jahr, Sternentstehung etwa konstant bei 0,55 M☉/Jahr
  (vorher 0,14 und fallend). Gas 1024 → 1164 Teilchen.
- Kosten: je neuem Teilchen drei Durchläufe O(N) und eine zusätzliche Kraftauswertung (die Beschleunigungen werden
  neu berechnet). Bei 50 000 Teilchen kommt etwa alle 100 Schritte eines dazu, also rund 1 % mehr Aufwand.
  Die Schrittrate sank über 1 Gyr von 170 auf 130 Schritte/s. Das liegt vor allem am dichteren, gewachsenen Gas
  (mehr Nachbarn, kleinere h).
- Vom Nutzer gesetzte Galaxien (Werkzeug) haben keinen Zufluss.

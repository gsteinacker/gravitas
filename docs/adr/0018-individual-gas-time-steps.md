# ADR-0018: Eigene Zeitschritte für Gas (Blockzeitschritte)

- Status: accepted
- Datum: 2026-10-29
- Tasks: T-085

## Kontext
Bisher zerlegte die Courant-Bedingung des dichtesten Gasteilchens den ganzen Schritt in gleiche Teile (bis 64).
In jedem Teil wurden die Gravitation aller Körper und die Hydrodynamik des gesamten Gases neu berechnet. In
klumpigen Szenen bestimmte so eine Handvoll dichter oder heißer Teilchen das Tempo von allem. Bei kleinen
Teilchenzahlen kam hinzu, dass jede Gravitationsrechnung auf der GPU eine feste Latenz hat (etwa 6 ms), während
der die CPU wartet. Der Nutzer sah die Kerne nur zur Hälfte ausgelastet.

## Entscheidung
Gravitation und Hydrodynamik werden getrennt integriert (`internal/sim/blockstep.go`):

- **Gravitation im globalen Schritt.** Wie Sterne bekommt das Gas die Gravitationskicks am Anfang und Ende des
  Szenenschritts Δt (Impuls-/RESPA-Aufteilung wie bei den Teilschritten nahe Löchern, T-025, `split.go`). Die
  Gravitation wird einmal pro Schritt berechnet, parallel zur letzten Gasauswertung.
- **Blockzeitschritte für die Hydrodynamik.** Jedes Gasteilchen hat eine Stufe r und den Schritt Δt/2^r, die
  kleinste Stufe, deren Schritt die Courant-Bedingung erfüllt (r ≤ 10, also bis Δt/1024). Es durchläuft
  einen eigenen Kick-Drift-Kick-Leapfrog mit Druck- und Viskositätskraft und du/dt. Ein Schritt darf nur auf
  einem Vielfachen seiner Länge beginnen, so bleiben die Stufen geschachtelt, und am Ende von Δt sind alle
  Teilchen synchron.
- **Nur aktive Teilchen werden ausgewertet.** Von Ereignis zu Ereignis driftet das ganze Gas. Dichte, Kraft und
  neuer Schritt werden nur für die Teilchen berechnet, deren Schritt endet. Inaktive Nachbarn gehen mit
  vorhergesagter Dichte und Glättungslänge ein (ρ ← ρ e^(−∇·v Δt), h ← h e^(∇·v Δt/3)), mit ihrer aktuellen
  Energie und Geschwindigkeit (wie GADGET-2, Springel 2005).
- **Begrenzer und Aufwecken (Saitoh & Makino 2009).** Der neue Schritt eines Teilchens ist höchstens viermal so
  lang wie der kürzeste Schritt seiner Nachbarn. Bekommt ein Teilchen einen kurzen Schritt, werden Nachbarn mit
  mehr als dem Vierfachen geweckt: Ihr laufender Schritt endet beim nächsten passenden Takt, der öffnende Kick
  wird auf die verkürzte Länge korrigiert. Sonst liefe eine Stoßwelle unbemerkt in Gas mit langem Schritt.
  Vor jedem globalen Schritt wird der Schritt jedes Teilchens zusätzlich auf die Schallzeit seiner aktuellen
  Energie begrenzt, damit durch Supernovae geheiztes Gas sofort kurze Schritte bekommt.
- Kühlung, Sternentstehung, Supernovae und Zufluss bleiben am Ende des globalen Schritts, wenn alles synchron
  ist. Gespeichert werden zusätzlich die Hydro-Beschleunigungen und Schrittlängen. Damit läuft eine geladene
  Simulation bitidentisch weiter.

## Alternativen
- **Weiter global zerlegen, aber ohne Gravitation in den Teilschritten:** Das spart die GPU-Latenz, rechnet aber
  weiter in jedem Teilschritt das gesamte Gas.
- **Nur Begrenzer, ohne Aufwecken:** einfacher, aber der Begrenzer greift erst, wenn das Teilchen ohnehin
  aktiv wird (Durier & Dalla Vecchia 2012 zeigen, dass das bei starker Heizung nicht reicht).
- **Strengerer Courant-Faktor** (0,15 statt 0,3), um die Energieerhaltung des alten Verfahrens zu erreichen
  (siehe unten): Er kostet in Szenen, deren Gas ohnehin knapp an der Courant-Grenze läuft (Pferdekopfnebel),
  mehr als das alte Verfahren.
- **Eigene Gravitationsschritte für alle Körper:** Das ist der größere Umbau (Baum je Stufe, GPU-Latenz pro
  Stufe). Sterne kommen mit dem Szenenschritt aus.

## Konsequenzen
- Der Impuls ist nicht mehr paarweise exakt erhalten, wenn Nachbarn auf verschiedenen Stufen sind (die Kräfte
  wirken zu verschiedenen Zeiten). Test `TestIndividualGasStepsConserveEnergyAndMomentum` (heißer Kern mit
  20-facher Energie in einer Gaskugel, 300 Schritte): Impuls 10⁻⁶ relativ, Energie 1,9 %. Das alte Verfahren kam
  hier auf 1,5 %, weil der heiße Kern das gesamte Gas und die Gravitation in Teilschritte zwang. Bei 100-facher
  Energie sind es 3,6 % gegenüber 2,0 %. Ein Begrenzer-Verhältnis von 2 statt 4 brachte kaum etwas (3,3 %), erst
  gleiche Schritte für alle (1,1 %), und die kosten das Vierfache. In den Szenen ohne Kühlung (Kollision 10 000,
  Galaxienentstehung 8000) liegt die Drift alt wie neu bei 10⁻⁴.
- Der Schrittzähler `Steps` zählt jetzt nur globale Schritte. Bisher zählte er die Teilschritte mit. Was alle n
  Schritte geschieht (Ionisation `ionEvery`), lief in Gasszenen mit Teilschritten öfter. Der Pferdekopfnebel
  rechnet deshalb jetzt mit `ionEvery` 2 statt 4 (vorher etwa zwei Teilschritte pro Schritt).
- Gas nahe einem Loch bewegt sich mit den Teilschritten der Löcher (T-025). Es bekommt seine Hydro-Kicks, aber
  seine Lage während der Gasereignisse ist die vom Ende des Schritts. Das betrifft nur wenige Teilchen kurz vor der
  Akkretion.
- Die Gravitation von dichtem Gas wird nicht mehr feiner aufgelöst als der Szenenschritt, wie bei Sternen.
  Unaufgelöst dichtes Gas wird ohnehin zu Sternen (ADR-0017).
- HUD und Screenshotzeile zeigen unter „gas“ die Zahl der Gasauswertungen pro Schritt. `HydroLoad` misst deren
  Kosten in Auswertungen des gesamten Gases.
- Messungen: `docs/perf-log.md` (2026-10-29, T-085).

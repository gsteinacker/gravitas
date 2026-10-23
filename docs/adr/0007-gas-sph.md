# ADR-0007: Gas mit SPH, Kühlung und Sternentstehung

- Status: accepted
- Datum: 2026-10-24
- Tasks: T-036

## Kontext
Reine Sternscheiben heizen auf und verlieren ihre Spiralarme (T-033). In echten Galaxien halten Gas, Kühlung und
Sternentstehung Scheiben kalt und machen die Arme durch junge Sterne sichtbar. Ohne Gas fehlen auch Starbursts
bei Kollisionen, Staub und ionisierte Nebel (Pferdekopfnebel).

## Entscheidung
- Körperart `Gas`: Gravitation wie ein Stern (Baum, GPU, Akkretion), dazu Hydrodynamik nach SPH
  (`internal/sph`, Schnittstelle `sim.Hydro`).
- Kubik-Spline-Kernel, adaptive Glättungslänge (~40 Nachbarn, 3 gedämpfte Iterationen), symmetrische Variable-h-
  Formulierung (Impuls und Energie paarweise exakt erhalten), Monaghan-Viskosität (α = 1, β = 2), Energiegleichung für u.
- Nachbarsuche: Morton-sortierter Bounding-Volume-Baum mit hmax je Knoten (Paare mit unterschiedlichem h finden
  sich gegenseitig).
- Kühlung: implizite Relaxation auf `uFloor` mit `coolTime` (stabil für jedes dt). Sternentstehung: Gas mit
  ρ > `sfDensity`, ∇·v < 0 und u < 3 uFloor wird mit Wahrscheinlichkeit 1 − exp(−ε dt / t_ff) zum Stern
  (deterministischer Zufall aus ID und Schritt). Neue Sterne tragen `Born` und werden jung (blau) gezeichnet.
- Szenen geben Temperaturen in K an (`gasTemperature`, `gasFloorK`) und die Geschwindigkeitseinheit
  (`velocityUnit`); daraus u = 3kT/(2μ m_p).

## Alternativen
- Gitter-Hydrodynamik (AMR): genauer bei Stößen, aber nicht lagrangesch und schwer mit der Teilchen-Gravitation
  zu koppeln.
- „Sticky particles“ (inelastische Stöße): billig, aber ohne Druck und Temperatur, also ohne Ionisation.

## Konsequenzen
- Kosten: Spirale mit 150k Körpern (11k Gas) 90 → ~35 Schritte/s. Hydro läuft nur auf der CPU.
- Keine Gradienten-h-Korrekturen: Energieerhaltung adiabatisch ~1 % in 300 Schritten (Test).
- Courant-Bedingung wird nicht automatisch erzwungen. Szenen müssen dt passend wählen (Galaxien-Szenen:
  dt_C ≈ 0,02–0,04 gegenüber dt = 0,004).
- Lokale Gruppe bleibt ohne Gas: Bei ihrer Auflösung (~2.500 Gasteilchen pro Galaxie) wäre es bedeutungslos und
  verletzte die Courant-Bedingung.

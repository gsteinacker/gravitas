# ADR-0017: Unaufgelöstes Gas wird sofort zu Sternen

- Status: accepted
- Datum: 2026-10-29
- Tasks: T-083

## Kontext
In der Szene „Galaxy formation“ (T-082) sammelt sich Gas im Zentrum bis zu Dichten, bei denen die
Glättungslänge an ihrer Untergrenze ε/4 klebt. Dieses Gas ist von der Simulation nicht mehr aufgelöst. Es bleibt
dabei warm, weil die Kompression es schneller heizt, als die feste Kühlzeit es abkühlt. Deshalb bildet es nach
dem Kriterium u < 3 u_floor (ADR-0007) keine Sterne. Gleichzeitig bestimmt es über die Courant-Bedingung den
Zeitschritt des gesamten Gases: Die Szene fiel von 150 auf 40 Schritte/s.

## Entscheidung
Zusammenströmendes, neutrales Gas, dessen Dichte über der Auflösungsgrenze liegt, wird im selben Schritt zu einem
Stern, unabhängig von seiner Temperatur.

- Grenze: ρ > 3 N m / (4π (ε/2)³). Das ist die Dichte, bei der die SPH-Glättungslänge h = ½ (3 N m / 4π ρ)^⅓
  unter ε/4 fiele, also unter die Auflösung der Gravitation.
- Weitere Bedingungen wie bisher: ρ > sfDensity, div v < 0, Ionisationsgrad < 0,5.
- Das entspricht dem Prinzip von Sink-Teilchen (Bate et al. 1995): Was unterhalb der Auflösung kollabiert, wird
  nicht weiter verfolgt. Physikalisch würde es dort ohnehin weiter kollabieren und Sterne bilden.

## Alternativen
- **Glättungslänge ohne Untergrenze:** Die Dichte stiege ohne physikalische Bedeutung weiter, der Zeitschritt
  schrumpfte unbegrenzt.
- **Dichteabhängige Kühlung Λ(T, ρ):** Dichtes Gas kühlte schneller, das Kriterium griffe wieder. Das ist richtig,
  aber ein größerer Umbau. Er ersetzt diese Regel nicht, denn auch kaltes, unaufgelöstes Gas ist nicht verfolgbar.
- **Eigene Zeitschritte für Gas (T-085):** Sie mildern die Kosten, lösen aber nicht das Auflösungsproblem.

## Konsequenzen
- Test: `TestUnresolvedGasFormsStarsAtOnce` (heißes Gas knapp über der Grenze wird zu Sternen, knapp darunter
  nicht).
- In den Galaxienszenen 1, 2, 6, 7 erreicht das Gas die Grenze selten. Die Kalibrierung der Sternentstehung
  (ADR-0016) bleibt erhalten, siehe Messung in T-083.
- Der dichte Zustand der Szene „Galaxy formation“ rechnet gut doppelt so schnell (zusammen mit T-086).

# ADR-0014: Kosmologischer Modus (mitbewegte Koordinaten, Particle-Mesh)

- Status: accepted
- Datum: 2026-10-26
- Tasks: T-074 (Analyse: T-062, `docs/analysis/04-universe.md`)

## Kontext
Das kosmische Netz (T-073) ist eine statische Zel'dovich-Momentaufnahme. Um zu zeigen, wie es aus winzigen
Schwankungen wächst, braucht es den expandierenden Raum. Das passt nicht zum bisherigen Modell, weil dieses
newtonsch mit offenen Rändern rechnet.

## Entscheidung
Szenen mit `params.cosmology` rechnen in einem eigenen Modus: mitbewegte Koordinaten in einem periodischen Würfel.

- **Variablen:** u = a² dx/dt. Bewegungsgleichungen dx/dt = u/a² und du/dt = −∇φ/a mit ∇²φ = 4πG(ρ − ρ̄),
  flaches ΛCDM für H(a).
- **Integrator:** KDK in Schritten Δ ln a = `dt` mit den exakten Faktoren ∫dt/a (Kick) und ∫dt/a² (Drift),
  Simpson in ln a (Quinn et al. 1997). Danach Rückfaltung in den Würfel.
- **Kräfte:** Particle-Mesh mit CIC-Zuweisung, FFT-Poisson (−4πG/k², ohne Mittelwert, CIC-Fenster entfaltet)
  und spektralem Gradienten, CIC-Interpolation zurück. Das Gitter ist eine Zweierpotenz mit g³ ≥ N (16–128).
  Der Löser liegt in `internal/cosmo` und wird über `sim.ComovingSolver` eingehängt, damit `sim` nicht von
  `cosmo` abhängt.
- **Anfangsbedingungen:** Zel'dovich mit u = a² D f H Ψ. Box, Ω_m, H₀ und a_start setzt die Komponente
  `cosmicweb`.
- **Im Modus abgeschaltet:** Löcher, Gas, c, Energiedrift-Anzeige (die Newton-Energie gilt nicht). Die Anzeige
  zeigt z statt t.

## Alternativen
- **TreePM:** höhere Auflösung, aber deutlich mehr Aufwand (periodischer Baum, Kraftaufteilung). Etappe 3, erst
  wenn PM überzeugt.
- **Ewald-Summation mit dem Baum:** genau, aber teuer.
- **Expansion nur als Skalierung der Darstellung:** keine Dynamik, also nichts gewonnen.

## Konsequenzen
- Tests:
  - Wachstum einer ebenen Welle gegen D(a): 12,15 statt 12,42, also −2 % durch die Gitterauflösung.
  - Layzer-Irvine d[a(K+W)] = −K da bis in den nichtlinearen Bereich: 0,4 %.
  - Impulserhaltung.
  - Szene 13 baut ihren Würfel und wächst.
- Auflösung etwa zwei Gitterzellen: Halos unter ~10¹³ M☉ (bei 64³ in 100 Mpc/h) bleiben unaufgelöst.
- Kosten: 12 ms pro Schritt bei 50k (64³), 29 ms bei 262k (`docs/perf-log.md`).

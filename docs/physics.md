# Physikmodell

Ziel: **plausibel und spielbar statt exakt.** Jede Vereinfachung ist hier dokumentiert. Änderungen am Modell
brauchen einen ADR.

## Einheiten

Simulationseinheiten mit `G = 1`. Längen, Massen und Zeiten sind dimensionslos.
- `c` (Lichtgeschwindigkeit) ist ein **einstellbarer Parameter**. Default ist `c = 10`: Die innere Scheibe
  läuft mit 0,3–0,5 c, r_s eines Lochs mit M = 2 ist 0,04, die Galaxie als Ganzes bleibt Newtonsch.
  Im UI mit `<` `>`, beim Start mit `-c`. `c = 0` (`-c -1`) schaltet alle relativistischen Effekte ab.
  „Reduziertes c“ ist der zentrale Trick: echte Werte würden die Effekte unsichtbar machen.

## Gravitation (Newton, geglättet)

    a_i = Σ_j  G m_j (r_j − r_i) / (|r_j − r_i|² + ε²)^(3/2)

- **Softening ε** verhindert Singularitäten bei engen Begegnungen.
- **Barnes-Hut** (Octree, Öffnungswinkel θ = 0,9, **Quadrupol-Momente**) für N ≫ 10³. Median-Kraftfehler
  0,05 % (der Monopol bei θ = 0,6 hatte 0,23 %). Die direkte Summe bleibt als Referenzlöser für Tests.
- Schwarze Löcher (wenige, schwere Körper) werden **immer direkt** summiert und nicht über den Baum genähert.

## Integration

- **Leapfrog KDK** (kick-drift-kick), symplektisch, feste globale Schrittweite `dt` für Stern-Stern-Kräfte.
- **Mehrfach-Zeitschritte nahe Schwarzer Löcher** (`sim/split.go`, RESPA): Die Löser liefern nur Stern-Stern-
  Kräfte. Alle Kräfte mit einem Loch summiert der Integrator direkt. Sterne mit τ = √(r/a) < 20·dt an einem
  Loch sowie alle Löcher bekommen diese Kräfte in n Unterschritten (Zweierpotenz, n ≥ 20·dt/τ_min, ≤ 256),
  alle anderen Sterne an den äußeren Kicks. Beide Partner eines Paars werden immer gleichzeitig gekickt, der
  Impuls bleibt also exakt erhalten.
  Gemessen: Drift der Testgalaxie (c = 10) von 0,30 auf 5·10⁻⁴ in 500 Schritten, +1,7 % Rechenzeit.
- Diagnose: Gesamtenergie, Impuls, Drehimpuls. Ein Energie-Drift-Test ist Pflicht für jede Solver-Änderung.

## Lichtgeschwindigkeit: was wir berücksichtigen

| Effekt | Modell | Kosten | Meilenstein |
|--------|--------|--------|-------------|
| Geschwindigkeitslimit | Integration der **Eigengeschwindigkeit** `u = γv`: `du/dt = a`, `v = u / √(1 + u²/c²)`. Garantiert `|v| < c` | ~0 | M3 |
| Ereignishorizont | Schwarzschild-Radius `r_s = 2GM/c²`. Was hineinfällt, wird absorbiert (Masse und Impuls bleiben erhalten) | ~0 | M3 |
| Innerste stabile Bahn (ISCO) | **Paczyński-Wiita-Potential** `Φ = −GM/(r − r_s)` für Quellen, die Schwarze Löcher sind. Liefert ISCO bei `3 r_s` und „Plunge“-Verhalten | gering (nur BH-Quellen) | M3 |
| Endliche Ausbreitung der Gravitation | Optional (`p`, `-gravity`): retardierte Position aus Ringpuffer, ohne oder mit v·Δt-Extrapolation, **nur für BH-Quellen**. Siehe ADR-0005 | −23 % bei 100k | M5 ✓ |
| Visuell: Doppler-/Rotverschiebung | Farbverschiebung nach radialer Geschwindigkeit zur Kamera und nach Potentialtiefe | nur Rendering | M4 |
| Visuell: Gravitationslinse | Bildschirm-Shader, der den Hintergrund um BHs verzerrt | nur Rendering | M4 |

## Umsetzung (M3)

- **Gespeicherte Geschwindigkeit ist bei c > 0 die Eigengeschwindigkeit u = γv** (`sim/relativity.go`).
  Kick: `u += a·dt`, Drift: `x += dt · u/γ`, γ = √(1 + u²/c²). Das ist ein symplektischer Leapfrog für
  H = Σ m c² (γ − 1) + U. Kinetische Energie m u²/(γ + 1), Impuls m·u.
- Anfangsbedingungen und UI liefern Newtonsche Geschwindigkeiten. Szenen übernehmen sie direkt als u
  (bei v ≪ c gleichwertig), platzierte Objekte werden mit `ProperVelocity` umgerechnet (Kappung bei 0,99 c).
- **Akkretion** nach jedem Schritt (`sim/accrete.go`): Sterne innerhalb r_s werden verschluckt, Löcher
  verschmelzen, wenn sich ihre Horizonte berühren (d < r_s1 + r_s2). Erhalten bleiben Ruhemasse und Impuls,
  der Ort ist der Schwerpunkt. Vereinfachung: Die kinetische Energie des Sterns wird nicht zu Masse.
- **Energiebuchhaltung:** Jede Absorption ändert die Gesamtenergie legitim. Die Änderung wird exakt berechnet
  (O(N): Paarterm plus verschobene Masse für alle anderen Körper) und in `Simulation.ExternalEnergy`
  aufsummiert. Die Drift im HUD ist (E − E0 − ΔE_extern)/|E0|, also nur der Integrationsfehler.
- Bei einem Wechsel von c beginnt die Drift-Messung neu, weil die kinetische Energie von c abhängt.
- Nach jeder Absorption werden die Beschleunigungen neu berechnet. Die alten enthielten noch den Zug des
  verschluckten Körpers ohne Gegenkraft, was den Impuls verletzt (gemessen: Δp = 1 bei massiver Akkretion).
- **Paczyński-Wiita als Paarpotential** (`sim/pair.go`): Stern-Loch-Paare verwenden U = −m M/(r_ε − r_s) mit dem
  r_s des Lochs und r_ε = √(r² + ε²). Stern-Stern und Loch-Loch bleiben Newtonsch. Dadurch ist die Kraft
  symmetrisch und der Impuls erhalten. Löcher bekommen ihre Kraft deshalb direkt von allen Sternen, nicht über
  den Baum. Unterhalb 1,5 r_s wird die Kraft auf ihren Wert an der Photonensphäre gekappt (Potential linear
  fortgesetzt), damit sie bei r → r_s nicht divergiert. Die ISCO bei 3 r_s bleibt unberührt.
- Gemessen: Kreisbahn bei 6 r_s und 0,34 c ist stabil, gestört bei 2 r_s stürzt sie hinein (die Newtonsche
  Kontrolle bleibt gebunden). Mehraufwand ~2,5 % bei 100k.

## Dunkle Materie (ADR-0006)

Halos aus unsichtbaren Teilchen (`DarkMatter`), gravitativ wie Sterne. Mit `x` lässt sich ihre Gravitation
abschalten (sie fliegen dann als Testteilchen weiter), mit `v` werden sie sichtbar. Galaxien in den Szenen
5–7 haben Halos mit ~85–95 % der Masse.

## Bulge (T-040)

Hernquist-Kugel alter Sterne (Massen 0,1–1 statt bis 10, daher röter) mit isotroper Jeans-Dispersion im
Gesamtpotential aus Bulge, Scheibe (sphärisch genähert), Halo und Loch; die Scheibe rotiert in diesem Potential.
Milchstraße 15 %, M31 30 % der Sternmasse; M33 und die Magellanschen Wolken haben keinen. Kein Balken, keine
Rotation des Bulges. Test: Virialverhältnis ≈ 1 und stabiler Halbmassenradius.

## Staub (T-035, T-040)

Staub folgt dem neutralen Gas. τ_V aus der Säulendichte in den Szeneneinheiten (N_H/A_V = 1,9·10²¹ cm⁻²,
Milchstraßen-Verhältnis); Szenen ohne `velocityUnit` normieren automatisch. Jeder Stern und jedes Hα-Sprite wird
mit dem Staub vor ihm geschwächt, τ · Φ((d − μ)/σ) aus Mittelwert und Streuung der Staubtiefe je Bildpunkt
(`internal/render/dust.go`); der Himmelshintergrund liegt hinter allem Staub.

## Entstehung Schwarzer Löcher (ADR-0012)

Hoop-Kriterium auf der kleinsten aufgelösten Kugel: Steckt in R = 2ε die Masse M ≥ R c²/2 in mindestens 20
Teilchen, werden diese zu einem Loch (`sim/collapse.go`, alle 10 Schritte). Dabei bleiben Masse, Impuls und
Drehimpuls erhalten (Drehimpuls als Spin), die Energie wird bilanziert. Mit dem echten c geschieht das in keiner
Szene. Die Szene „Direct collapse“ zeigt es mit verkleinertem c.

Rotierende Löcher (T-071, `sim/kerr.go`): Jedes Loch trägt seinen Drehimpuls J. Akkretion und Verschmelzung
addieren Spins und Bahndrehimpuls, begrenzt auf a* = 0,998. Gezeichnet werden r₊, die ISCO und der Photonenring
nach Kerr (`docs/analysis/03-kerr.md`). Dazu kommt Frame-Dragging (T-072, ADR-0013, `sim/framedrag.go`), die
Lense-Thirring-Kraft a = v × B in Ordnung 1/c² als Boris-Rotation in den Kicks. Die Rückwirkung geht in Impuls
und Spin des Lochs. Invarianten: Impuls und Gesamtdrehimpuls inklusive Spins (`TestFrameDraggingConserves`),
Knotenpräzession gegen Ω_LT (`TestFrameDraggingPrecessesTheNode`).

Akkretionsscheiben (T-065) sind reine Darstellung. Jedes Loch merkt sich seine Akkretionsrate (gleitend über 50
Schritte, `sim/accrete.go`), die Helligkeit 1 − exp(−Ṁ/M/10⁻³) treibt. Gezeichnet wird eine dünne
Novikov-Thorne-Scheibe von der ISCO bis 30 r_g in der Ebene senkrecht zum Spin. Dazu kommen Doppler und
Rotverschiebung g⁴, die Rückseite über eine Ablenkung 4 r_g/b und der Schatten b < √27 r_g. Die Dynamik der
Scheibe selbst löst das SPH-Gas nicht auf (Glättung ≫ r_g). Entschieden in T-065: Darstellung statt Dynamik,
weil eine α-Scheibe auf r_g-Skalen Auflösungen und Zeitschritte weit unter denen der Szenen bräuchte.

## Gas (ADR-0007)

SPH-Gas mit Druck, Viskosität, Kühlung auf eine Untergrenze und Sternentstehung in dichtem, kaltem,
zusammenströmendem Gas. Junge Sterne werden blau gezeichnet (`o` zeigt das Gas selbst). Szenen 1, 2, 6, 7 haben
15 % Gas in der Scheibe bei 10⁴ K.

Schwelle in den Galaxienszenen: `sfDensity` = 0,002 Simulationseinheiten (≈ 0,1 cm⁻³, Springel & Hernquist 2003).
Die geglättete Gasdichte der Scheibe liegt bei 0,005 (Median) bis 0,1 (99 %). Die frühere Schwelle 0,5
(≈ 20 cm⁻³) erreichte sie nie, deshalb entstanden fast keine Sterne. Gemessen (T-070, 60k Teilchen) sind es jetzt
rund 0,25 M☉/Jahr in Szene 1, in der Größenordnung der Milchstraße (1–2 M☉/Jahr).

Darstellung als Sternpopulationen (T-070, `internal/engine/population.go`, Lernthema „populations“): Jedes
Sternteilchen ist eine Population von 10⁵–10⁶ M☉. Es wird als Lichthof der Größe ε gezeichnet, dazu kommen
Einzelsterne, die beim Hineinzoomen aufgelöst werden. Sternbildendes Gas leuchtet mit dem erwarteten Licht seiner
jungen Sterne als bläulicher Schleier. Das ist reine Darstellung, die Dynamik bleibt unverändert.

Zeitschritt: Gas verlangt Δt ≤ 0,3 h / v_sig (Courant, v_sig = c_i + c_j − 3 min(0, v_ij·r̂), Monaghan 1997).
Ist der Szenen-Schritt länger, wird er in gleiche Teile zerlegt (höchstens 64, HUD „gas N“). Ohne das
heizt sich Gas in Schocks bei Galaxienverschmelzungen explizit unbegrenzt auf und die Rechnung explodiert.
h ist nach unten auf ε/4 begrenzt: Unterhalb der Gravitationsauflösung ist Gas ohnehin nicht aufgelöst.

## Was wir bewusst ignorieren

- Allgemeine Relativität jenseits der obigen Näherungen (keine Metrik, kein Frame-Dragging).
- Gravitationswellen-Abstrahlung (eventuell später als „Inspiral-Hack“ für BH-Paare).
- Sternentwicklung und Supernova-Rückkopplung. Sterne sind Massepunkte.
- Kollisionen zwischen normalen Sternen (optional: Verschmelzen unterhalb eines Abstands).

## Invarianten für Tests

1. Zweikörperproblem: Kepler-Bahn bleibt über 1000 Umläufe geschlossen (Energie-Drift < 1e-4 relativ).
2. Barnes-Hut mit θ → 0 entspricht der direkten Summe (relativer Fehler < 1e-10).
3. Impulserhaltung bei Absorption durch ein Schwarzes Loch.
4. Kein Körper erreicht jemals `|v| ≥ c`.

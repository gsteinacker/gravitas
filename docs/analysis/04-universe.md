# Analyse: vom Galaxie- zum Universum-Maßstab (kosmisches Netz)

Stand 2026-10-26, Task T-062. Frage: Kann Galaxy die großräumige Struktur zeigen, also Haufen, Filamente, Voids und
Attraktoren, bis hin zur Strukturbildung aus dem frühen Universum? Was ist physikalisch und technisch sinnvoll?

## Was sich gegenüber dem heutigen Modell ändert

### 1. Expandierender Raum
Kosmologische N-Körper-Rechnungen laufen in **mitbewegten Koordinaten** x = r/a(t) mit dem Skalenfaktor a. Mit dem
kanonischen Impuls p = a² ẋ gilt:

- ẋ = p / a², ṗ = −∇φ (in diesen Variablen)
- ∇²φ = 4πG a² (ρ − ρ̄) / a³: Nur die **Abweichung** von der mittleren Dichte zieht an.
- Friedmann (flaches ΛCDM): H(a)² = H₀² (Ω_m a⁻³ + Ω_Λ), mit Planck 2018 etwa H₀ = 67,7 km/s/Mpc, Ω_m = 0,31,
  Ω_Λ = 0,69, σ₈ = 0,81, n_s = 0,965.

Der Integrator braucht dafür Drift- und Kick-Faktoren ∫dt/a² und ∫dt/a (Quinn et al. 1997), am besten in Schritten
gleich in ln a. Das ist ein neues Physikmodell und braucht einen ADR. Die heutige Energie-Invariante gilt dann
nicht mehr, an ihre Stelle tritt die Layzer-Irvine-Gleichung.

### 2. Periodische Randbedingungen und langreichweitige Kraft
Ein Ausschnitt des Universums ist ein periodischer Würfel. Der Barnes-Hut-Baum kennt nur offene Ränder.
- **PM (Particle-Mesh):** Dichte auf ein Gitter (CIC), Poisson per FFT, Kräfte zurück interpoliert. Das ist
  periodisch von selbst, schnell und für Haufen und Filamente ausreichend; die Auflösung ist auf etwa 2 Gitterzellen
  begrenzt.
- **TreePM (GADGET-2):** PM für große Abstände, der vorhandene Baum für kurze, mit einem erfc-Split. Damit lassen
  sich Halos auflösen. Der Baum muss dafür periodische Bilder berücksichtigen (nächstes Bild).
- **Ewald-Summation** mit reinem Baum: genau, aber teuer. Nicht empfohlen.
- Die Go-Standardbibliothek hat keine FFT. Eine 3D-Radix-2-FFT ist etwa 200 Zeilen eigener Code (ohne neue
  Abhängigkeit).

### 3. Anfangsbedingungen
- **Leistungsspektrum P(k)** aus der Transferfunktion nach Eisenstein & Hu (1998), als Fit-Formel ohne externe
  Codes, normiert auf σ₈.
- **Zel'dovich-Näherung oder 2LPT** bei z ≈ 50: Gauß'sches Zufallsfeld im k-Raum, Verschiebungen
  Ψ = −i k/k² δ(k), Geschwindigkeiten aus f(Ω) H a Ψ.
- **Reale Nachbarschaft** (Laniakea, Großer Attraktor) über „constrained ICs“ aus Cosmicflows-4 (Wiener-Filter und
  Constrained Realizations). Das wäre ein eigenes Forschungsprojekt mit externen Daten. Statistisch typische
  Volumen reichen aus, um das kosmische Netz zu zeigen. **Empfehlung:** zunächst typische Volumen, Laniakea nur als
  Erklärtext und Bild im Lernmodus.

### 4. Dunkle Materie als Hauptkomponente und Halo-Finder
Fast alle Teilchen sind Dunkle Materie, deren Darstellung als Nebel schon existiert. Damit man Haufen und Filamente
erkennt, braucht es:
- **Dichte pro Teilchen** (k nächste Nachbarn, die SPH-Nachbarsuche ist vorhanden) für eine Färbung nach log ρ
- **Friends-of-Friends** (Verknüpfungslänge 0,2 × mittlerer Abstand) für Halo-Massen und Beschriftung der Haufen

### 5. Galaxienbildung
Gas mit Kühlung, Sternentstehung und Rückkopplung bräuchte im Würfel von 100 Mpc einige Milliarden Teilchen
(IllustrisTNG: 2 × 2500³). Mit 10⁵–10⁶ Teilchen ist das **nicht machbar**. Galaxien könnte man nur als
Subgrid-Etikett zeigen („in diesem Halo säße eine Galaxie der Masse …“, Abundance Matching).

## Skalen und Grenzen

Teilchenmasse m_p = Ω_m ρ_krit L³ / N mit ρ_krit = 2,775 · 10¹¹ h² M☉/Mpc³:

| Würfel L | N | m_p | Milchstraßen-Halo (10¹² M☉) | Haufen (10¹⁴–10¹⁵ M☉) |
|---|---|---|---|---|
| 50 Mpc/h | 10⁶ (100³) | 1,1 · 10¹⁰ h⁻¹ M☉ | ~90 Teilchen, gerade erkennbar | 10⁴–10⁵ |
| 100 Mpc/h | 10⁶ | 8,6 · 10¹⁰ h⁻¹ M☉ | ~12 Teilchen | 10³–10⁴ |
| 100 Mpc/h | 10⁵ (46³) | 8,6 · 10¹¹ h⁻¹ M☉ | 1 Teilchen | 10²–10³ |
| 500 Mpc/h | 10⁶ | 1,1 · 10¹³ h⁻¹ M☉ | – | einige 10–100 |

**Folge:** Filamente, Voids und Haufen sind ab etwa 64³ Teilchen in 100 Mpc/h gut sichtbar, Einzelgalaxien nicht.
Laniakea (~160 Mpc) passt nur in einen Würfel von 300–500 Mpc, und dann nur grob.

**Zeit:** Von z = 50 bis heute sind es 13,7 Gyr, in ln a etwa 4. Mit 200–500 globalen Schritten (PM) sind das bei
10⁵ Teilchen Sekunden bis Minuten. Interaktiv („Struktur wächst vor den Augen“) geht das mit 64³–100³ Teilchen.

## Darstellung
- Anzeige von Rotverschiebung z und kosmischer Zeit statt Simulationszeit; Skalenbalken in Mpc
- Färbung nach Dichte (log ρ, Voids dunkel, Knoten hell), Kamera mit periodischer Wiederholung des Würfels
- Übergang von Skala zu Skala nur illustrativ: einen Halo auswählen und die Galaxienszene laden, ohne
  Konsistenz in der Physik

## Verträglichkeit mit dem Bestehenden
- **Einheiten:** G = 1 bleibt, Länge Mpc/h und Masse 10¹⁰ M☉/h als Szeneneinheiten (`velocityUnit`, `yearsPerUnit`
  bleiben brauchbar).
- **Reduziertes c:** Ohne Sinn auf dieser Skala, die Rechnung ist newtonsch (Würfel ≪ Hubble-Radius). c wird in
  diesem Modus abgeschaltet.
- **Schwarze Löcher, Retardierung, Linse:** in diesem Modus aus.
- **Lernmodus, neue Themen:** Expansion des Raums, Dunkle Energie, kosmisches Netz, Rotverschiebung,
  Strukturwachstum (lineares Wachstum D(a), Zel'dovich-Pfannkuchen).

## Leistung
- **PM 128³:** CIC-Zuweisung O(N), 3D-FFT etwa 2 · 10⁸ Flops, rechnet man vorwärts und rückwärts zusammen. In
  Go parallel sind das grob 20–40 ms pro Kraftberechnung bei 10⁶ Teilchen. Auf Metal ginge es schneller, das ist
  aber für den ersten Schritt nicht nötig.
- **TreePM:** Der Baumanteil wird kurzreichweitig und dadurch billiger als der heutige volle Baum.

## Name
„Universe“ ist als App-Name vielfach belegt, und die App heißt „Galaxy“. **Empfehlung:** Den Namen behalten und den
Modus „Kosmisches Netz“ bzw. „Cosmic Web“ als Szenengruppe nennen.

## Empfehlung und Etappen
1. **Etappe 1 (M): Zel'dovich-Szene.** Ein Szenen-Komponententyp `cosmicweb` erzeugt aus P(k) per Zel'dovich das
   Netz bei wählbarem z. Statisch oder mit dem bestehenden Integrator als Nicht-Expansion-Demo; dazu
   Dichtefärbung. Das zeigt Filamente und Voids sofort, ganz ohne neues Physikmodell. Bei z = 0 schmiert die
   Näherung allerdings, ehrlich als Modellgrenze benannt.
2. **Etappe 2 (L): Expansion und PM**, periodisch, mitbewegt, mit ADR. Struktur wächst von z = 50 bis 0.
   Invariante: Layzer-Irvine; Test: lineares Wachstum D(a) einer einzelnen Mode.
3. **Etappe 3 (L): TreePM und Friends-of-Friends**, Halos aufgelöst, Haufen beschriftet, Halo-Massenfunktion gegen
   Tinker et al. 2008 geprüft.
4. **Etappe 4 (optional): Galaxien als Subgrid-Etiketten** (Abundance Matching), keine Hydrodynamik.

Etappe 1 und 2 sind lohnend und gut mit dem Lernmodus verbunden. Etappe 3 erst, wenn 2 überzeugt. Gas und
Galaxienbildung im kosmischen Würfel ausdrücklich nicht.

## Quellen
- Springel 2005, MNRAS 364, 1105 (GADGET-2, TreePM); Quinn et al. 1997 (kosmologischer Leapfrog)
- Eisenstein & Hu 1998, ApJ 496, 605 (Transferfunktion); Zel'dovich 1970; Crocce et al. 2006 (2LPT)
- Davis et al. 1985 (Friends-of-Friends); Tinker et al. 2008 (Halo-Massenfunktion)
- Planck Collaboration 2020, A&A 641, A6; Tully et al. 2014, Nature 513, 71 (Laniakea)

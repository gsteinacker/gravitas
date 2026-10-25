# Analyse: rotierende Schwarze Löcher (Kerr)

Stand 2026-10-26, Task T-059. Einheiten wie in `docs/physics.md`: G = 1, c ist ein Parameter. Hier steht
r_g = GM/c² für den Gravitationsradius (r_s = 2 r_g) und a* = Jc/(GM²) ∈ [0, 1) für den dimensionslosen Spin.

## Heutiges Modell

- **Horizont:** r_s = 2 r_g. Akkretion verschluckt, was hineinfällt (`sim/accrete.go`).
- **ISCO:** über das Paczyński-Wiita-Potential Φ = −M/(r − r_s) für Löcher als Quellen: ISCO bei 6 r_g und
  „Plunge“ darunter.
- **Darstellung:** Schwarzschild. Horizont und Photonenring bei 1,5 r_s, ISCO-Ring bei 3 r_s (`holeFragment`).
  Dazu Punktlinse mit θ_E² ∝ 4GM/c² und gravitative Rotverschiebung √(1 − r_s/r).
- **Löcher** tragen Masse, Ort und Geschwindigkeit, aber keinen Drehimpuls.

## Was ein Kerr-Loch sichtbar ändern würde

| Effekt | Kerr | Schwarzschild heute | In der App sichtbar? |
|---|---|---|---|
| Horizont | r₊ = r_g (1 + √(1 − a*²)): von 2 r_g bis 1 r_g | 2 r_g | Ja, bei reduziertem c: das Loch schrumpft mit dem Spin bis auf die Hälfte |
| ISCO (Bardeen et al. 1972) | prograd 6 → 1 r_g, retrograd 6 → 9 r_g (a* = 0,9: 2,32 bzw. 8,72 r_g) | 6 r_g für alle | Ja: Sterne mit der Drehung kreisen viel enger, gegen die Drehung stürzen sie früher |
| Strahlungseffizienz 1 − E_ISCO | 5,7 % → 42 % (Thorne-Grenze a* = 0,998: 32 %) | 5,7 % | Nur mit Akkretionsscheibe (T-065): hellere Scheiben bei hohem Spin |
| Photonenring und Schatten | D-förmig verformt, um bis zu ~2 r_g zur Gegenseite verschoben; Größe ändert sich nur um wenige Prozent (≈ 10,4 r_g Durchmesser bei a* = 0) | Kreis | Ja, aber subtil; deutlich nur seitlich und bei a* > 0,7 |
| Frame-Dragging (Lense-Thirring) | Bahnebenen präzedieren mit Ω_LT = 2GJ/(c²r³) = 2a* r_g² c/r³ | fehlt | Ja, bei kleinem c und Bahnen innerhalb ~20 r_g; die S-Sterne-Szene mit realem c zeigt es nicht (Effekt ~10⁻⁴ der Schwarzschild-Präzession) |
| Spin-Entwicklung | Akkretion dreht auf bis a* ≈ 0,998; Verschmelzung gleicher, nicht rotierender Löcher ergibt a* ≈ 0,69 | – | Ja, als Anzeige und über die geänderte ISCO |
| Rückstoß bei Verschmelzung | bis ~5000 km/s bei passenden Spins („superkicks“) | heute nur Impulserhaltung, kein GW-Rückstoß | Nur mit einem eigenen Rückstoßmodell (Ausblick) |

## Mögliche Modelle

### A. Nur Darstellung (Spin als Zustand, Horizont, Ringe, Schatten)
- Pro Loch ein Spinvektor **J** (3 float64 im State, mit gespeichert).
- Die Anzeige verwendet r₊, den ISCO-Ring für pro- und retrograd (zwei Ringe in der Bahnebene der Scheibe) und
  einen verformten Photonenring. Für die Form genügt die analytische Schattenkontur (Bardeen 1973), im Shader über
  eine Tabelle α(β) je a* und Blickwinkel.
- Spin-Entwicklung bei Akkretion: J += L des verschluckten Körpers relativ zum Loch, a* auf 0,998 begrenzt. Bei
  Verschmelzung J₁ + J₂ + L_Bahn, auch auf 0,998 begrenzt; die Abstrahlung über Gravitationswellen fehlt, sie liefe
  bei gleichen Massen auf a* ≈ 0,69 hinaus.
- **Kosten:** vernachlässigbar. **Grenze:** Die Bahnen merken nichts vom Spin, Darstellung und Dynamik passen dann
  nicht zusammen.

### B. Pseudo-newtonsche Kerr-Potentiale
- **Artemova, Björnsson & Novikov (1996):** F = −M / (r^(2−β) (r − r₊)^β) mit β = r_ISCO/r₊ − 1. Horizont und
  prograde ISCO stimmen exakt.
- **Mukhopadhyay (2002):** Potential für die Äquatorebene, ebenfalls an Horizont und ISCO angepasst.
- **Grenze:** Beide gelten nur in der Äquatorebene. Sie sind zudem kugelsymmetrisch in r, unterscheiden also nicht
  zwischen pro- und retrograd. Genau dieser Unterschied ist aber der auffälligste Kerr-Effekt.
- **Kosten:** wie heute PW (nur Paare mit Löchern).

### C. Gravitomagnetische Kraft (Lense-Thirring) zusätzlich zu PW
- a_LT = (2G / (c² r³)) [3 (**J**·r̂)(r̂ × **v**) + **v** × **J**] für jeden Körper im Feld eines Lochs
  (Gravitoelektromagnetismus, Ordnung 1/c²).
- Die Kraft steht senkrecht auf **v** und leistet keine Arbeit. Mit einem Boris-artigen Rotationsschritt im
  Leapfrog (wie bei Ladungen im Magnetfeld) bleibt die Energie-Invariante erhalten. Das Drehmoment auf das Loch
  ergibt die Spinpräzession; damit lässt sich die Drehimpulsbilanz testen.
- Unterscheidet pro- und retrograd und erzeugt Knotenpräzession. Die ISCO verschiebt sich in der richtigen
  Richtung, aber nur in erster Ordnung: ±1 r_g statt 1 bzw. 9 r_g bei a* → 1. Für exakte Werte bräuchte man eine
  Kopplung mit B, etwa PW mit spin- und richtungsabhängigem r_in aus L·J.
- **Kosten:** eine Kreuzproduktsumme pro Körper und Loch, wenige Prozent der Schrittzeit.

### D. Geodäten im Kerr-Feld (Boyer-Lindquist oder Kerr-Schild)
- Exakt für Testteilchen, aber nicht mit der newtonschen N-Körper-Dynamik und Mehrlochsystemen vereinbar.
  Außerdem ist es teuer (Zeitschritte ∝ r^(3/2) nahe dem Horizont). **Nicht empfohlen.**

## Verträglichkeit
- **Reduziertes c:** Alle Kerr-Längen skalieren mit r_g = GM/c². Bei c = 10 werden sie wie heute die
  Schwarzschild-Größen gut sichtbar. Mit realem c bleiben sie unsichtbar klein, das ist korrekt.
- **Substeps nahe Löchern:** Prograde Bahnen bis 1 r_g brauchen engere Substeps. Das bestehende Kriterium
  (Beschleunigung, Abstand) greift weiter, kostet aber bei schnell rotierenden Löchern mehr. Die Grenze dafür ist
  a* ≤ 0,998.
- **Energie-Invarianten (`docs/physics.md`):**
  - A ändert die Energie nicht.
  - C leistet keine Arbeit (Testteilchen-Näherung). Weil die Kraft nicht symmetrisch ist, ist der Impuls nur
    zusammen mit dem Spin des Lochs bilanziert. Das muss ein Test „J_Loch + Σ L = konst.“ abdecken.
- **Retardierte Gravitation:** Der Spin wirkt in C instantan. Retardierung des Gravitomagnetismus ist eine weitere
  Ordnung und wird nicht empfohlen.

## Empfehlung
**Ja, in zwei Stufen**, weil Rotation der Normalfall ist und die Unterschiede bei reduziertem c groß und lehrreich
sind:
1. **Stufe 1 (M, ~2 Tage):** Modell A. Spin als Zustand (Szenen-JSON `spin` je Loch, Werkzeug „Spin“ beim
   Platzieren), Horizont r₊, pro- und retrograder ISCO-Ring, D-förmiger Photonenring, Spin-Entwicklung durch
   Akkretion und Verschmelzung, Anzeige im Inspektor. Passt zur Akkretionsscheibe (T-065), deren Innenkante an
   der ISCO liegt.
2. **Stufe 2 (M–L, ~3 Tage):** Modell C. Lense-Thirring-Kraft mit Boris-Schritt, Spinpräzession des Lochs, Tests:
   Knotenpräzession gegen Ω_LT (Testteilchen), Energie bei reiner LT-Kraft, Drehimpulsbilanz. Braucht einen ADR,
   weil es das Physikmodell ändert (Entwurf unten).

Modell B lohnt sich nicht, weil es den Richtungsunterschied nicht abbildet. Modell D passt nicht zum N-Körper-Kern.

### ADR-Entwurf (für Stufe 2)
- **Titel:** Gravitomagnetische Kraft rotierender Löcher.
- **Entscheidung:** Zusätzlich zu Paczyński-Wiita gibt es eine Lense-Thirring-Beschleunigung in Ordnung 1/c² auf
  Körper im Feld eines rotierenden Lochs. Integriert wird sie mit einem Boris-Rotationsschritt, die Rückwirkung
  wirkt nur auf den Spin.
- **Alternativen:** ABN-/Mukhopadhyay-Potentiale (richtungsblind), Kerr-Geodäten (inkompatibel mit N-Körper).
- **Konsequenzen:** ISCO-Verschiebung nur in erster Ordnung (dokumentierte Modellgrenze); neue Invariante
  „Gesamtdrehimpuls inklusive Spin“.

## Lernmodus
- **Neues Thema „Rotierende Schwarze Löcher (Kerr)“:** Spin, kleinerer Horizont, ISCO 1–9 r_g, Frame-Dragging,
  Thorne-Grenze, Spin durch Akkretion und Verschmelzung. Modellgrenzen: Ordnung 1/c², keine Ergosphären-Dynamik,
  keine Gravitationswellen.
- **Thema „isco“** um die Spinabhängigkeit ergänzen.
- **Themen „lensing“ und „black-holes“:** Hinweis auf den D-förmigen Schatten (EHT-Bilder von M87* und Sgr A*).

## Quellen
- Bardeen, Press & Teukolsky 1972, ApJ 178, 347 (Kerr-ISCO, Effizienz)
- Bardeen 1973, in *Black Holes* (Les Houches): Schatten des Kerr-Lochs
- Thorne 1974, ApJ 191, 507 (Spin-Grenze 0,998)
- Artemova, Björnsson & Novikov 1996, ApJ 461, 565; Mukhopadhyay 2002, ApJ 581, 427
- Rezzolla et al. 2008, ApJ 674, L29 (Spin nach Verschmelzung); Campanelli et al. 2007, PRL 98, 231102 (superkicks)

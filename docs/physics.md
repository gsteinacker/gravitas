# Physikmodell

Ziel: **plausibel und spielbar statt exakt.** Jede Vereinfachung ist hier dokumentiert. Änderungen am Modell
brauchen einen ADR.

## Einheiten

Simulationseinheiten mit `G = 1`. Längen, Massen und Zeiten sind dimensionslos.
- `c` (Lichtgeschwindigkeit) ist ein **einstellbarer Parameter** (Default so gewählt, dass relativistische
  Effekte in der Nähe Schwarzer Löcher sichtbar werden, typischerweise `c ≈ 10–50 · v_typisch`).
  „Reduziertes c“ ist der zentrale Trick: echte Werte würden die Effekte unsichtbar machen.

## Gravitation (Newton, geglättet)

    a_i = Σ_j  G m_j (r_j − r_i) / (|r_j − r_i|² + ε²)^(3/2)

- **Softening ε** verhindert Singularitäten bei engen Begegnungen.
- **Barnes-Hut** (Octree, Öffnungswinkel θ) für N ≫ 10³. Die direkte Summe bleibt als Referenzlöser für Tests.
- Schwarze Löcher (wenige, schwere Körper) werden **immer direkt** summiert und nicht über den Baum genähert.

## Integration

- **Leapfrog KDK** (kick-drift-kick), symplektisch, feste globale Schrittweite `dt`.
- Diagnose: Gesamtenergie, Impuls, Drehimpuls. Ein Energie-Drift-Test ist Pflicht für jede Solver-Änderung.
- Später optional: hierarchische Zeitschritte für Körper nahe Schwarzer Löcher.

## Lichtgeschwindigkeit: was wir berücksichtigen

| Effekt | Modell | Kosten | Meilenstein |
|--------|--------|--------|-------------|
| Geschwindigkeitslimit | Integration der **Eigengeschwindigkeit** `u = γv`: `du/dt = a`, `v = u / √(1 + u²/c²)`. Garantiert `|v| < c` | ~0 | M3 |
| Ereignishorizont | Schwarzschild-Radius `r_s = 2GM/c²`. Was hineinfällt, wird absorbiert (Masse und Impuls bleiben erhalten) | ~0 | M3 |
| Innerste stabile Bahn (ISCO) | **Paczyński-Wiita-Potential** `Φ = −GM/(r − r_s)` für Quellen, die Schwarze Löcher sind. Liefert ISCO bei `3 r_s` und „Plunge“-Verhalten | gering (nur BH-Quellen) | M3 |
| Endliche Ausbreitung der Gravitation | Retardierte Position aus Ringpuffer, **nur für BH-Quellen** | mittel | M5 (Spike) |
| Visuell: Doppler-/Rotverschiebung | Farbverschiebung nach radialer Geschwindigkeit zur Kamera und nach Potentialtiefe | nur Rendering | M4 |
| Visuell: Gravitationslinse | Bildschirm-Shader, der den Hintergrund um BHs verzerrt | nur Rendering | M4 |

## Was wir bewusst ignorieren

- Allgemeine Relativität jenseits der obigen Näherungen (keine Metrik, kein Frame-Dragging).
- Gravitationswellen-Abstrahlung (eventuell später als „Inspiral-Hack“ für BH-Paare).
- Sternentwicklung, Gas, Hydrodynamik. Sterne sind Massepunkte.
- Kollisionen zwischen normalen Sternen (optional: Verschmelzen unterhalb eines Abstands).

## Invarianten für Tests

1. Zweikörperproblem: Kepler-Bahn bleibt über 1000 Umläufe geschlossen (Energie-Drift < 1e-4 relativ).
2. Barnes-Hut mit θ → 0 entspricht der direkten Summe (relativer Fehler < 1e-10).
3. Impulserhaltung bei Absorption durch ein Schwarzes Loch.
4. Kein Körper erreicht jemals `|v| ≥ c`.

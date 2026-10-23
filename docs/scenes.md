# Szenen

Szenen sind **Rezepte als JSON**. Die eingebauten liegen in `internal/scene/scenes/` (eingebettet, Tasten 1–9 in
Dateinamen-Reihenfolge). Eigene startet man mit `go run ./cmd/galaxy -scene meine.json -n 100000`.
Spielstände (F5/F9, `-load`) sind dagegen **exakte Abzüge** eines laufenden Zustands, siehe unten.

## Aufbau

```json
{
  "name": "Name im HUD",
  "info": "Einheiten und Annahmen, eine Zeile im HUD",
  "yearsPerUnit": 13200000,
  "params": {"dt": 0.004, "softening": 0.06, "c": 10},
  "removeBulkMotion": true,
  "camera": {"target": [0, 0, 0], "distance": 12, "pitch": 50, "yaw": 0},
  "effects": "glow,doppler,trails",
  "components": [ ... ]
}
```

- Einheiten: G = 1. Für eine physikalische Deutung wählt man Länge L und Masse M, dann gilt
  v = √(GM/L) und t = L/v. `yearsPerUnit` sorgt dafür, dass das HUD die Zeit in yr/kyr/Myr/Gyr anzeigt.
  Beispiele: 2,5 kpc und 2·10¹⁰ M☉ ergeben 185 km/s und 13,2 Myr. 100 kpc und 10¹² M☉ ergeben 207 km/s und 0,47 Gyr.
- `c` in denselben Einheiten: echt wäre 3·10⁵ km/s geteilt durch die Geschwindigkeitseinheit.
- Winkel in Grad. `weight` verteilt die Teilchenzahl `-n`, die letzte gewichtete Komponente bekommt den Rest.
- Gas: `velocityUnit` (km/s je Geschwindigkeitseinheit) und `gasFloorK` auf Szenenebene, in `params.gas`
  `coolTime`, `sfDensity`, `sfEfficiency`, `youngAge` (Simulationseinheiten). Siehe ADR-0007.
- `note` ist freier Text für Quellen und Annahmen und wird nicht ausgewertet.

## Komponenten

| `type` | Felder | Beschreibung |
|--------|--------|--------------|
| `galaxy` | `mass`, `diskFraction`, `diskScale`, `diskMin`, `diskMax`, `thickness`, `dispersion` (oder `toomreQ` und `coldFraction`), `haloProfile` (`plummer`/`hernquist`), `haloScale`, `haloCut` (plummer), `haloMax` (hernquist), `darkFraction`, `gasFraction`, `gasTemperature` (K), `holeMass`, `inclination`, `orientation`, `position`, `velocity`, `seed` | Exponentielle Scheibe auf Kreisbahnen (aus der eingeschlossenen Masse von Loch, Scheibe und Halo), optional Plummer-Halo aus Dunkler Materie (`darkFraction` der Teilchen) |
| `plummer` | `mass`, `scale`, `position`, `velocity`, `seed` | Sternhaufen im Virialgleichgewicht |
| `hole` | `mass`, `position`, `velocity` | einzelnes Schwarzes Loch |
| `orbits` | `mass` (zentrales Loch), `aScale`, `orbits[]` (`a`, `e`, `i`, `node`, `peri`, `anomaly`), `starMass`, `background`, `backgroundA`, `backgroundMass`, `seed` | Kepler-Bahnen um ein Loch, z. B. aus Bahnelementen der Literatur |

## Eingebaute Szenen

| Taste | Szene | Einheiten | Quelle / Annahmen |
|---|---|---|---|
| 1–4 | Spirale, Kollision, Sternhaufen, Doppel-Loch | skalenfrei, gedeutet als 2,5 kpc bzw. 1 pc | Spielszenen, c reduziert |
| 5 | Lokale Gruppe | 100 kpc, 10¹² M☉, **echtes c** | galaktische Koordinaten und Geschwindigkeiten aus der Literatur, siehe `note` in der Datei |
| 6 | Antennen-Galaxien | 2,5 kpc | Toomre & Toomre (1972): gleich schwere Scheiben, parabolisch, prograd |
| 7 | Whirlpool (M51) | 2,5 kpc | angelehnt an M51/NGC 5195, nicht angepasst |
| 8 | S-Sterne um Sgr A* | 1000 AE, 4,3·10⁶ M☉, **echtes c** | Bahnelemente gerundet nach Gillessen et al. 2017, Phasen illustrativ |

## Spielstände

`F5` schreibt `saves/galaxy-<Zeit>.galaxy` (gob + gzip, Format-Version 3; 1 und 2 bleiben lesbar): alle Körper in float64 samt Gasgrößen
(u, h, Ionisation, Leuchtkraft, Geburtszeit), die letzte Kraftauswertung, Parameter, Zeit,
Akkretions-Buchhaltung, Zahl entstandener Sterne, Kamera und die Szenen-Metadaten. `F9` lädt den neuesten,
`-load datei.galaxy` startet damit; `go run ./tools/evolve -preset 9 -t 0.5 -o x.galaxy` rechnet eine Szene headless vor. Ein geladener Stand rechnet bitidentisch weiter (Test). Nicht gespeichert
wird die Bahngeschichte der retardierten Gravitation: Löcher gelten nach dem Laden als schon immer vorhanden.

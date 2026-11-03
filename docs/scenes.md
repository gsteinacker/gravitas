# Szenen

Szenen sind **Rezepte als JSON**. Die eingebauten liegen in `internal/scene/scenes/` (eingebettet, Tasten 1–9 in
Dateinamen-Reihenfolge). Eigene startet man über *Ablage → Öffnen …* oder mit `make run ARGS="-scene $PWD/meine.json -n 100000"`.
Spielstände (F5/F9, `-load`) sind dagegen **exakte Abzüge** eines laufenden Zustands, siehe unten.

## Aufbau

```json
{
  "name": "Name im HUD",
  "group": "galaxies",
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
- `velocityUnit` (km/s je Geschwindigkeitseinheit) zusammen mit `yearsPerUnit` legt die Längeneinheit fest
  (L = v · t, Test `TestUnitsMatchInfo`). Damit zeigt das Start-Werkzeug Abstände und Tempo physikalisch an.
- Gas: `velocityUnit` und `gasFloorK` auf Szenenebene, in `params.gas`
  `coolTime`, `sfDensity`, `sfEfficiency`, `youngAge` (Simulationseinheiten). Siehe ADR-0007.
- Supernovae (ADR-0015): `supernovaErgPerMsun` (z. B. 1e49) und optional `supernovaDelayMyr` ([3, 40]) auf Szenenebene
- `group` ordnet die Szene in der Szenenübersicht (⇧⌘N) und im Menü „Neu aus Szene“ ein: `galaxies`, `stars`,
  `gas` oder `cosmos` (T-078). Die Kurzbeschreibung dort ist der erste Satz von `intro.school`.
- `note` ist freier Text für Quellen und Annahmen und wird nicht ausgewertet.
- `paused: true` startet die Szene angehalten (statische Szenen wie das kosmische Netz).

## Komponenten

| `type` | Felder | Beschreibung |
|--------|--------|--------------|
| `galaxy` | `mass`, `diskFraction`, `diskScale`, `diskMin`, `diskMax`, `thickness`, `dispersion` (oder `toomreQ` und `coldFraction`), `haloProfile` (`plummer`/`hernquist`), `haloScale`, `haloCut` (plummer), `haloMax` (hernquist), `darkFraction`, `gasFraction`, `gasTemperature` (K), `bulgeFraction`, `bulgeScale`, `holeMass`, `infallMsunPerYear` (Gaszufluss, braucht Loch und Einheiten, ADR-0016), `inclination`, `orientation`, `position`, `velocity`, `seed` | Exponentielle Scheibe auf Kreisbahnen (aus der eingeschlossenen Masse von Loch, Bulge, Scheibe und Halo), optional Halo aus Dunkler Materie (`darkFraction` der Teilchen) und Hernquist-Bulge alter Sterne (`bulgeFraction` der Sternmasse, Skala `bulgeScale`, isotrope Jeans-Geschwindigkeiten) |
| `plummer` | `mass`, `scale`, `position`, `velocity`, `seed`, `rotation` (0–1), `spinAxis` | Sternhaufen im Virialgleichgewicht; `rotation` lässt diesen Anteil der Sterne gleichsinnig umlaufen (Lynden-Bell, Gleichgewicht bleibt) |
| `halo` | `mass`, `scale`, `haloCut` (Standard 0,99), `axes` (Achsenverhältnisse, triaxial), `inclination`, `orientation`, `position`, `velocity`, `seed` | Halo aus Dunkler Materie (Plummer), z. B. für eine Gaswolke, die darin kollabiert; mit `axes` gestreckt (Orte und Geschwindigkeiten, Produkt auf 1 normiert), dann um x und z gedreht (T-084) |
| `gasbox`, `gasclump` | `mass`, `size` (gasbox) bzw. `scale` (gasclump), `gasTemperature` (K), `turbulence` (km/s), `perturb`, `position`, `velocity`, `seed`, `rotation`, `spinAxis` | Gas als Kasten oder Plummer-Wolke; `rotation` gibt zusätzlich `rotation` × Kreisbahngeschwindigkeit (aller zuvor gebauten Körper) um `spinAxis`, β = rotation²/3 |
| `hole` | `mass`, `position`, `velocity`, `spin` (a*, 0–0,998), `spinAxis` (Standard +z) | einzelnes Schwarzes Loch, mit `spin` rotierend (Kerr, nur mit c) |
| `orbits` | `mass` (zentrales Loch), `aScale`, `orbits[]` (`a`, `e`, `i`, `node`, `peri`, `anomaly`), `starMass`, `background`, `backgroundA`, `backgroundMass`, `seed` | Kepler-Bahnen um ein Loch, z. B. aus Bahnelementen der Literatur |
| `cosmicweb` | `box` (Mpc/h), `redshift`, `position`, `seed` | kosmisches Netz: Zel'dovich-Realisierung von ΛCDM (Planck 2018) als Dunkle Materie, Masse Ω_m ρ_krit L³ (T-073) |

## Eingebaute Szenen

| Taste | Szene | Einheiten | Quelle / Annahmen |
|---|---|---|---|
| 1–4 | Spirale, Kollision, Sternhaufen, Doppel-Loch | skalenfrei, gedeutet als 2,5 kpc bzw. 1 pc | Spielszenen, c reduziert |
| 5 | Lokale Gruppe | 100 kpc, 10¹² M☉, **echtes c** | galaktische Koordinaten und Geschwindigkeiten aus der Literatur, siehe `note` in der Datei |
| 6 | Antennen-Galaxien | 2,5 kpc | Toomre & Toomre (1972): gleich schwere Scheiben, parabolisch, prograd |
| 7 | Whirlpool (M51) | 2,5 kpc | angelehnt an M51/NGC 5195, nicht angepasst |
| 8 | S-Sterne um Sgr A* | 1000 AE, 4,3·10⁶ M☉, **echtes c** | Bahnelemente gerundet nach Gillessen et al. 2017, Phasen illustrativ |
| 9 | Pferdekopfnebel | 1 pc, 100 M☉ | Photoverdampfung einer Wolkenkante, siehe `note` |
| (14) | Galaxienentstehung | 2,5 kpc, 2·10¹⁰ M☉ | rotierende heiße Gaswolke (Baryonenanteil 0,15) in einem DM-Halo kollabiert zur Scheibe (T-082), siehe `note` |
| (15) | Milchstraße mit Satelliten | 2,5 kpc, 2·10¹⁰ M☉ | LMC, SMC und Sagittarius-Zwerg an Orten und auf Bahnebenen aus der Literatur, feinere Massenauflösung der Satelliten (T-088, ADR-0019), siehe `note` |
| (16) | Balkenspirale | 2,5 kpc, 2·10¹⁰ M☉ | scheibendominierte Variante von Szene 1, bildet einen Balken (T-090) |
| Menü | 10 Direkter Kollaps | 1 pc, 10⁴ M☉, c = 15 | 10⁵ M☉ Gas bei 8000 K kollabiert zum Loch (T-067, ADR-0012) |
| Menü | 11 Akkretionsscheibe | 1 pc, 10⁵ M☉, c = 3 | Gaswolke füttert ein Loch, Scheibe als Darstellung (T-065) |
| Menü | 12 Kosmisches Netz | 1 Mpc, Würfel 100 Mpc/h | Zel'dovich, ΛCDM Planck 2018, statisch (T-073) |

## Spielstände

`F5` schreibt `~/Library/Application Support/Galaxy/saves/galaxy-<Zeit>.galaxy` (`os.UserConfigDir`) (gob + gzip, Format-Version 4; 1 bis 3 bleiben lesbar): alle Körper in float64 samt Gasgrößen
(u, h, Ionisation, Leuchtkraft, Geburtszeit) und dem Spin der Löcher, die letzte Kraftauswertung, Parameter, Zeit,
Akkretions-Buchhaltung, Zahl entstandener Sterne, Kamera und die Szenen-Metadaten. `F9` lädt den neuesten,
`-load datei.galaxy` startet damit; `go run ./tools/evolve -preset 9 -t 0.5 -o x.galaxy` rechnet eine Szene headless vor. Ein geladener Stand rechnet bitidentisch weiter (Test). Nicht gespeichert
wird die Bahngeschichte der retardierten Gravitation: Löcher gelten nach dem Laden als schon immer vorhanden.

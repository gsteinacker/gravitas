---
id: T-064
title: Objekte in 3D platzieren und mit realistischer Geschwindigkeit starten
status: done
milestone: M10
priority: 1
estimate: L
depends: [T-058]
---

## Ziel

Heute landet ein platziertes Objekt in der Brennebene der Kamera, die Tiefe ist dadurch zufällig, und die
Startgeschwindigkeit ergibt sich aus der Zieh-Länge ohne nachvollziehbare Einheit. Gebraucht wird ein Werkzeug,
mit dem man Ort, Richtung und Geschwindigkeit in 3D gezielt festlegt, mit realistischen Werten in physikalischen
Einheiten (km/s, kpc, pc).

## Akzeptanzkriterien

- [x] Platzierungsebene sichtbar und wählbar (z. B. Galaxienebene der Szene, Ebene durch ein gewähltes Objekt,
      Kameraebene), mit Gitter/Schatten bzw. Lot zur Ebene, damit die Tiefe erkennbar ist
- [x] Höhe über der Ebene einstellbar (Modifier-Taste + Ziehen oder Feld im Inspektor)
- [x] Richtung als 3D-Pfeil, drehbar unabhängig von der Kamera; Vorschau der Bahn (einige Orbits/Zeitschritte
      vorausgerechnet im Testfeld, ohne die Simulation zu stören)
- [x] Geschwindigkeit in physikalischen Einheiten (km/s) mit Bezug: z. B. Kreisbahn-, Flucht- und
      Relativgeschwindigkeit am Ort als Referenzmarken; Eingabe per Regler/Zahl; realistische Bereiche je Szene
      (Galaxienbegegnungen 100–1000 km/s, Sterne um Sgr A* bis einige 1000 km/s, Haufen 1–10 km/s)
- [x] Ziel wählen: auf ein Objekt/Galaxienzentrum zielen (z. B. Stoßparameter angeben)
- [x] Gilt für Schwarze Löcher, Sternhaufen und Gaswolken (T-063)
- [x] Physik-Invarianten unverändert (Impuls/Energie der Bestandsszene), Test der Umrechnung Einheiten → Simulation
- [x] Lernmodus: Hinweise zu Kreisbahn- und Fluchtgeschwindigkeit

## Notizen

Wunsch des Nutzers vom 2026-10-25. Einheiten aus `kmPerS`/`yearsPerUnit` der Szene (docs/scenes.md); Szenen ohne
Einheiten zeigen Simulationseinheiten mit Hinweis.

## Notizen

- Umsetzung: `internal/engine/launch.go` (Ebene, Basis prograd/radial, eingefrorenes Feld, Referenzen, Vorschau, Gizmo),
  Swift: Klick plant, ↩ startet, ⎋ verwirft, Doppelklick sofort, ⌥-Ziehen = Richtung/Tempo und Start; Abschnitt
  „Start“ im Inspektor; Gizmo als Vektor-Layer über dem Metal-Bild.
- Höhe per Inspektor-Regler (nicht per Modifier-Ziehen). Stoßparameter wird angezeigt und über Richtung/„Zentrum“
  eingestellt, nicht als eigenes Eingabefeld.
- Einheiten: `velocityUnit` für alle Szenen ergänzt (3, 4, 5, 8), L = v·t, Test gegen die Info-Zeilen.
- Messwerte: Spiralgalaxie 5,7 kpc → v_K 189 km/s, v_F 383 km/s; S-Sterne 4480 AE → 922 / 1304 km/s
  (Kepler: 890 km/s).
- Neues Lernthema `orbital-speeds`, Hinweis beim ersten Start-Planen. `-launch x,y[,preset]` für Screenshots.

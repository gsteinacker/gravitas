---
id: T-062
title: Prüfen: aus Galaxy ein Universe machen (kosmologische Strukturen)
status: todo
milestone: M11
priority: 2
estimate: L
depends: []
---

## Ziel

Kann Galaxy über einzelne Galaxien hinaus die großräumige Struktur des Universums zeigen: Galaxienhaufen,
Filamente, Voids, Attraktoren (z. B. Großer Attraktor, Laniakea), bis hin zum frühen Universum mit Strukturbildung
und Galaxienentstehung? Vor dem Bauen wird geprüft, was physikalisch und technisch sinnvoll machbar ist.

## Akzeptanzkriterien

- [ ] Analyse (`docs/analysis/`) der nötigen Physik und Abweichungen vom heutigen Modell:
      - expandierender Raum (Friedmann-Gleichungen, mitbewegte Koordinaten, Skalenfaktor a(t), ΛCDM-Parameter)
      - periodische Randbedingungen und langreichweitige Kräfte (TreePM/Particle-Mesh mit FFT, Ewald)
      - Anfangsbedingungen aus dem Leistungsspektrum (Zel'dovich-Näherung/2LPT bei z ≈ 50–100)
      - Dunkle Materie als dominierende Komponente; Halo-Finder (Friends-of-Friends) zum Sichtbarmachen
      - Galaxienbildung: Gas, Kühlung, Sternentstehung, Rückkopplung – was mit SPH (T-036) im Maßstab geht und was
        nur als Subgrid-Modell
      - reale Strukturen in der Nähe (constrained ICs, z. B. aus Cosmicflows) vs. statistisch typische Volumen
- [ ] Skalen und Grenzen: Teilchenzahl/Masse pro Teilchen bei 10⁵–10⁶ Teilchen in 50–500 Mpc Box, was dann
      auflösbar ist (Haufen ja, Einzelgalaxien kaum); Zeitbereich und Schrittweiten
- [ ] Darstellung: Zeit/Rotverschiebung anzeigen, Kamera über Skalen, Filamente/Voids sichtbar machen (Dichte-
      Färbung), Übergang zwischen Skalen (Universe-Szene ↔ Galaxien-Szene)
- [ ] Verträglichkeit mit dem Bestehenden: Einheiten (G = 1), reduziertes c (sinnvoll?), Schwarze Löcher, Lernmodus
      (neue Themen: Expansion, Dunkle Energie, kosmisches Netz, Rotverschiebung)
- [ ] Leistung: Abschätzung PM-FFT auf CPU/Metal neben dem Baum
- [ ] Empfehlung mit Etappen (z. B. 1. statisches kosmisches Netz aus Simulation „Szene“; 2. Expansion + PM;
      3. Strukturbildung ab z ≈ 50; 4. Galaxienbildung grob) und Aufwand; ADR-Entwurf, Folge-Tasks

## Notizen

Wunsch des Nutzers vom 2026-10-25. Name/Marke („Universe“) mitprüfen.

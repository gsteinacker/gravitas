---
id: T-081
title: Gas strömt nach (Infall aus dem Halo) und Sternentstehungsrate kalibrieren
status: done
milestone: M9
priority: 2
estimate: L
depends: [T-080]
---

## Ziel

Damit in den Galaxienszenen dauerhaft Sterne entstehen, soll frisches Gas nachströmen, wie in echten Galaxien
durch Akkretion aus dem Halo und aus dem kosmischen Netz (~1 M☉/Jahr für die Milchstraße). Dazu soll die
Sternentstehungsrate in die Größenordnung realer Galaxien kommen (heute in Szene 1 etwa 0,14 M☉/Jahr, Milchstraße
1,65 M☉/Jahr).

## Akzeptanzkriterien

- [x] Analyse/ADR: Infall-Modell (z. B. kühlendes Halogas als Teilchenreservoir, oder neue Gasteilchen auf einer
      Schale mit Drehimpuls des Halos, die einregnen), Rate aus Literatur (Fraternali & Binney 2008, Putman et al.
      2012), Bilanz von Masse, Impuls und Energie (ExternalEnergy, Epoch)
- [x] Umsetzung für Szenen mit einer und mit mehreren Galaxien (Zentrum und Drehimpuls je Galaxie)
- [x] Teilchenbudget: neue Gasteilchen ersetzen verbrauchte, Gesamtzahl bleibt im Budget des Leistungsprofils
- [x] Kalibrierung der Sternentstehung (sfDensity, Effizienz, Auflösung) gegen Kennicutt–Schmidt; Messung mit
      `tools/evolve` über ≥ 1 Gyr: Rate bleibt etwa konstant statt zu fallen
- [x] Gasvorrat und Infall-Rate im Inspektor (Abschnitt „Sterne und Gas“) mit (i)
- [x] Invarianten-Tests, Lernthema (Gaskreislauf: Infall, Sternentstehung, Supernovae, Ausfluss) in drei Stufen
- [x] Screenshot-Prüfung

## Notizen

Wunsch des Nutzers vom 2026-10-29: „wir sollten das nachströmende Gas zu den szenen hinzufügen“.

Erledigt 2026-10-29: ADR-0016, `internal/sim/infall.go`, Szenenfeld `infallMsunPerYear` (1, 2, 6, 7; ≈ M_gas/3 Gyr),
ε_ff = 0,066. Szene 1 bei 20k über 1 Gyr: Zufluss 0,60 M☉/Jahr, Sternentstehung etwa konstant bei 0,55 M☉/Jahr
(vorher 0,14 und fallend). Teilchenbudget: neue Teilchen kommen hinzu statt verbrauchte zu ersetzen, +0,7 %
Teilchen pro Gyr; Recycling verworfen (siehe ADR). Inspektor zeigt „Gaszufluss“ mit (i), Lernthema „gas-cycle“.
Nebenbei: Zeiteinheiten auf Deutsch („Mio. J.“, „Mrd. J.“) statt Myr/Gyr; Lernthema „supernovae“ nennt
getriggerte Sternentstehung in Schalen (Nutzerfrage).

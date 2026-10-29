---
id: T-081
title: Gas strömt nach (Infall aus dem Halo) und Sternentstehungsrate kalibrieren
status: todo
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

- [ ] Analyse/ADR: Infall-Modell (z. B. kühlendes Halogas als Teilchenreservoir, oder neue Gasteilchen auf einer
      Schale mit Drehimpuls des Halos, die einregnen), Rate aus Literatur (Fraternali & Binney 2008, Putman et al.
      2012), Bilanz von Masse, Impuls und Energie (ExternalEnergy, Epoch)
- [ ] Umsetzung für Szenen mit einer und mit mehreren Galaxien (Zentrum und Drehimpuls je Galaxie)
- [ ] Teilchenbudget: neue Gasteilchen ersetzen verbrauchte, Gesamtzahl bleibt im Budget des Leistungsprofils
- [ ] Kalibrierung der Sternentstehung (sfDensity, Effizienz, Auflösung) gegen Kennicutt–Schmidt; Messung mit
      `tools/evolve` über ≥ 1 Gyr: Rate bleibt etwa konstant statt zu fallen
- [ ] Gasvorrat und Infall-Rate im Inspektor (Abschnitt „Sterne und Gas“) mit (i)
- [ ] Invarianten-Tests, Lernthema (Gaskreislauf: Infall, Sternentstehung, Supernovae, Ausfluss) in drei Stufen
- [ ] Screenshot-Prüfung

## Notizen

Wunsch des Nutzers vom 2026-10-29: „wir sollten das nachströmende Gas zu den szenen hinzufügen“.

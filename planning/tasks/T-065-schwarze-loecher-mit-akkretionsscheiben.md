---
id: T-065
title: Schwarze Löcher mit Akkretionsscheiben
status: doing
milestone: M9
priority: 2
estimate: L
depends: []
---

## Ziel

Schwarze Löcher sollen Akkretionsscheiben bekommen: heißes Gas, das auf Kreisbahnen bis zur innersten stabilen
Bahn (ISCO) spiralt, dabei hell leuchtet und das Loch für den Betrachter sichtbar macht – mit den bekannten Effekten
aus Bildern wie dem des Event Horizon Telescope: helle, durch Doppler-Beaming einseitig aufgehellte Seite,
Gravitationsrotverschiebung nach innen, durch die Linse über und unter das Loch gebogene Rückseite der Scheibe.

## Akzeptanzkriterien

- [x] Analyse und Entscheidung (Notiz oder ADR): reine Darstellung vs. Dynamik
      - Darstellung: dünne Scheibe (Novikov-Thorne/Shakura-Sunyaev, T(r) ∝ r^(−3/4) mit Innenkante am ISCO), Ausrichtung
        aus dem Drehimpuls des einfallenden Materials oder der Umgebung, Helligkeit aus der Akkretionsrate
      - Dynamik: Gas (SPH, T-036) mit Drehimpulstransport (α-Viskosität), Akkretion über den Horizont, Rückwirkung
        (Strahlung/Eddington-Grenze, Jets ausdrücklich nur als Ausblick)
- [x] Akkretionsrate aus der Simulation (Masse, die das Loch pro Zeit schluckt) treibt Helligkeit und Größe
- [ ] Metal-Darstellung: Temperaturfarbe, Doppler-Beaming (∝ D³…⁴) und Rotverschiebung in der Scheibe, Linse biegt die
      Rückseite sichtbar über das Loch (Abgleich mit Schwarzschild-Raytracing-Bildern, mindestens qualitativ)
- [ ] Wirkt mit reduziertem c (größere Horizonte → große, gut sichtbare Scheiben) und realem c (punktförmig hell)
- [x] Szene oder Startoption, die eine Scheibe gut zeigt (z. B. Loch mit Gaswolke/zerrissenem Stern, T-063)
- [x] Lernmodus: Thema „Akkretionsscheiben“ in drei Stufen mit Modellgrenzen; Bezug zu ISCO und Kerr (T-059)
- [x] Physik-Invarianten (Energie/Impuls) unverändert, wenn nur Darstellung; sonst Test der Massen- und
      Drehimpulsbilanz

## Notizen

Wunsch des Nutzers vom 2026-10-25. Hängt inhaltlich mit T-059 (Kerr: ISCO 1–9 GM/c², Frame-Dragging) und T-063
(Gaswolken als Materialquelle) zusammen.

Stand 2026-10-26:
- Entscheidung: Darstellung (Begründung in `docs/physics.md`). Eine α-Scheibe auf r_g-Skala bräuchte Auflösungen
  und Zeitschritte weit unter denen der Szenen.
- Akkretionsrate je Loch, gleitend über 50 Schritte (`sim/accrete.go`, `TestAccretionRateTracksSteadyFeeding`).
  Helligkeit 1 − exp(−Ṁ/M/10⁻³) (`engine.diskBrightness`). Die Ebene folgt dem Spin, der den Drehimpuls des
  Verschluckten aufnimmt (T-071).
- Shader `holeFragment`: dünne Novikov-Thorne-Scheibe ISCO…30 r_g, g⁴-Beaming und Rotverschiebung,
  Temperaturfarbe, Rückseite über eine Ablenkung 4 r_g/b, Schatten b < √27 r_g, Photonenring am Schattenrand.
- Szene 11 „Accretion disk“: Loch 10⁴ M☉ bei c = 3, Gaswolke auf exzentrischer Bahn. Headless: Ṁ/M ≈ 0,2–0,4
  pro Zeiteinheit, die Scheibe leuchtet also durchgehend.
- Lernthema „accretion-disks“.
- Offen: Screenshot-Abgleich mit Raytracing-Bildern und Prüfung bei realem c. Der Bildschirm war gesperrt.


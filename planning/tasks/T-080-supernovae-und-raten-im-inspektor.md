---
id: T-080
title: Supernovae und Sternentstehungsraten im Inspektor
status: done
milestone: M9
priority: 2
estimate: M
depends: [T-070]
---

## Ziel

Junge Sternpopulationen sollen nach einigen Millionen Jahren als Supernovae explodieren und das Gas aufheizen.
Sternentstehungsrate, Supernova-Rate und Gasvorrat sollen rechts im Inspektor ablesbar sein.

## Akzeptanzkriterien

- [x] ADR zur Rückkopplung (ADR-0015)
- [x] Jede aus Gas entstandene Population explodiert genau einmal nach 3–40 Myr, Energie 10⁴⁹ erg/M☉ als Wärme
      an die nächsten Gasteilchen, verbucht in ExternalEnergy; reproduzierbar, gespeichert
- [x] Invarianten-Tests (einmalige Explosion, Energiebilanz, keine für alte Sterne)
- [x] Raten gleitend über 100 Myr in M☉/Jahr und Supernovae pro Jahrhundert, Gasanteil; Tests
- [x] Inspektor-Abschnitt „Sterne und Gas“ mit (i)-Erklärungen in drei Stufen (de/en)
- [x] Sichtbarer, kurzer Blitz bei der Explosion (Markierung), Meldung bei der ersten Supernova
- [x] Lernthema „supernovae“, Szenen 1, 2, 6, 7 schalten Supernovae ein
- [x] Screenshot-Prüfung von Inspektor und Blitz

## Notizen

Wunsch des Nutzers vom 2026-10-29 („Sollten nicht ständig neue Sterne entstehen? oder auch explodieren?“).

Erledigt 2026-10-29: `internal/sim/supernova.go`, `internal/app/rates.go`, Shader-Blitz in `spriteVertex`.
Szene 1 bei 50 000 Teilchen nach 240 Myr: 0,14 M☉/Jahr, 0,14 Supernovae pro Jahrhundert, Gasanteil 8,7 %.
Milchstraße: 1,65 M☉/Jahr. Die Rate ist also etwa zehnmal zu niedrig. Kalibrierung und Gaszufuhr: T-081.

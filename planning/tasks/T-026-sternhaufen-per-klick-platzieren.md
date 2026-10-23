---
id: T-026
title: Sternhaufen per Klick platzieren
status: done
milestone: M1
priority: 1
estimate: S
depends: [T-006]
---

## Ziel

Neben Schwarzen Löchern auch kleine Sternhaufen in die Szene setzen oder hineinschießen (Nutzerwunsch).

## Akzeptanzkriterien

- [x] Werkzeugwahl: `b` Schwarzes Loch, `k` Sternhaufen, `Tab` wechselt, Anzeige im HUD
- [x] Klick platziert, Alt-Ziehen mit Geschwindigkeit, wie beim BH
- [x] Masse über `m`/`n` (gemeinsam mit BH)
- [x] Haufen ist gebunden: Plummer-Kugel, Virialtest in `scene`

## Notizen
- Größe: max(300, N/40) Sterne, also 2500 bei 100k. Radius 0,2 · (M/0,5)^⅓.
- Der Haufen wird auf der UI-Goroutine gebaut, auf der Sim-Goroutine nur angehängt. Test über den echten Runner in `ui`.
- Haufensterne sind schwerer als Galaxiesterne (M/N_Haufen statt 1/N) und leuchten dadurch heller. Das ist gewollt, sie bleiben so erkennbar.

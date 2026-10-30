# ADR-0019: Feinere Massenauflösung für Satellitengalaxien

- Status: accepted
- Datum: 2026-10-30
- Tasks: T-088

## Kontext
Die Szene „Milky Way satellites“ zeigt die Milchstraße mit LMC, SMC und dem Sagittarius-Zwerg. Mit den
Teilchenmassen der Hauptgalaxie (Sterne ~10⁶ M☉, Dunkle Materie ~10⁷ M☉ bei 50 000 Körpern) hätte Sagittarius
(Modellmasse 2·10⁹ M☉, Sterne 8·10⁷ M☉) nur etwa 100 Teilchen. Zweikörper-Relaxation
(t_relax ≈ N/(8 ln N) t_cross) löst so einen Zwerg in wenigen Kreuzungszeiten künstlich auf. Gemessen, isoliert
und ohne Wirt: Bei 200 Teilchen wächst der stellare Halbmassenradius in 1 Gyr von 0,65 auf 0,9 Einheiten.

## Entscheidung
Satelliten bekommen einen größeren Anteil am Teilchenbudget, als ihrer Masse entspricht. Das Szenenformat trennt
bereits Teilchenzahl (`weight`) und Masse (`mass`), deshalb braucht es keinen neuen Code.

Innerhalb eines Satelliten wird das Massenverhältnis von DM- zu Sternteilchen klein gehalten (Sgr `darkFraction` 0,8,
Verhältnis ≈ 5; LMC 0,75). Schwere DM-Teilchen heizen leichte Sterne auf (Energie-Äquipartition): Mit
`darkFraction` 0,6 (Verhältnis 16) wuchs der Radius von Sagittarius bei 1500 Teilchen in 530 Myr um 20 %, mit 0,8
bei 3000 Teilchen um 0,4 %.

Aufteilung bei 50 000 Körpern: Milchstraße 76 %, LMC 12 %, Sagittarius 8 %, SMC 4 %. Sagittarius hat dann
Sternteilchen von 10⁵ M☉, zehnmal leichter als die Scheibe der Milchstraße.

## Alternativen
- A: Gleiche Teilchenmassen für alle Komponenten: Die Satelliten wären zu grob aufgelöst und lösten sich künstlich
  auf, oder die Milchstraße bekäme nur noch wenige tausend Teilchen.
- B: Externes analytisches Potential für die Milchstraße: billig und rauschfrei. Dann gibt es aber keine
  dynamische Reibung, keine Reaktion der Scheibe und keinen Rückstoß der Milchstraße durch die LMC. Das ist das
  Interessante an der Szene.
- C: Nur Sternteilchen für die Satelliten (DM als statisches Potential, das mitfliegt): gleiches Problem wie B, und
  das Abstreifen der DM wäre falsch.

## Konsequenzen
- Positiv: Die Satelliten bleiben isoliert über viele Umläufe stabil (Test
  `TestFinelyResolvedDwarfKeepsItsSize`). Gezeitenströme, dynamische Reibung und die Störung der Scheibe entstehen
  von selbst.
- Negativ: Die schweren DM-Teilchen der Milchstraße (~10⁷ M☉) heizen Satelliten bei jedem Durchflug etwas auf. Im
  Halo ist ihre Dichte klein. Mischen sich gestreifte Satellitensterne mit Sternen der Scheibe, heizen sich auch
  diese gegenseitig leicht auf (Massenverhältnis 10).
- Bei kleinem Budget (< 20 000 Körpern) hat Sagittarius weniger als 1600 Teilchen, und die Relaxation wird wieder
  merklich. Die App wählt das Budget nach der Leistung, typischerweise 50 000 oder mehr.
- Kugelsternhaufen und lichtschwache Zwerge bleiben unter der Auflösung (Glättung 150 pc).

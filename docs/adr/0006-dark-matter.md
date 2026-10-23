# ADR-0006: Dunkle Materie als unsichtbare, schaltbare Teilchenart

- Status: accepted
- Datum: 2026-10-23
- Tasks: T-031, T-032

## Kontext
Für bekannte Strukturen (Lokale Gruppe, Antennen, M51) reichen nackte Scheiben nicht. Echte Galaxien bestehen
zu ~90 % aus Dunkler Materie. Ohne Halos fehlen flache Rotationskurven und dynamische Reibung, und
Milchstraße und Andromeda würden sich nur streifen statt verschmelzen. Bisher hat ein übergroßes zentrales
Loch die Scheiben stabilisiert.

## Entscheidung
- Neue Körperart `DarkMatter`: gravitativ wie ein Stern (Baum, Akkretion, Paczyński-Wiita mit Löchern), wird
  aber nicht als Stern gezeichnet.
- Halos als Plummer-Kugeln aus gleich schweren Teilchen, abgeschnitten bei einem Massenanteil (`haloCut`).
  Die Kreisgeschwindigkeiten der Scheibe berücksichtigen die eingeschlossene Halo-Masse analytisch.
- **Schaltbar zur Laufzeit** (`x`, `Params.DarkMatterOff`): Abgeschaltete Dunkle Materie hat keine
  Quellmasse mehr (`Params.SourceMass`), bewegt sich aber als Testteilchen weiter. So sieht man, dass Scheiben
  ohne sie zu schnell rotieren und auseinanderfliegen. Nach dem Wiedereinschalten ist der Halo noch da.
- **Sichtbar machbar** (`v`, Effekt `dark`): violetter Dunst mit eigener, von der Teilchenzahl unabhängiger
  Belichtung.

## Alternativen
- Statisches analytisches Halo-Potential: billiger, aber keine dynamische Reibung, und Halos könnten nicht
  verschmelzen. Gerade das macht die Lokale Gruppe interessant.
- NFW-Profile: noch realistischer, aber ohne geschlossene Masse bis unendlich und ohne einfache
  Verteilungsfunktion.

## Nachtrag (2026-10-23): Hernquist-Halos
Plummer-Halos sind zu zentral konzentriert. Bei der Milchstraße lagen 80 % der Masse innerhalb von 50 kpc
(real ~45 %). Dadurch war die LMC-Bahn zu eng (Apozentrum 67 kpc), und die Wolken verschmolzen nach 2 Gyr.
Szenen können jetzt `"haloProfile": "hernquist"` wählen (abgeschnitten bei `haloMax`, Geschwindigkeiten aus der
isotropen Jeans-Dispersion nach Hernquist 1990, unter der Fluchtgeschwindigkeit gekappt). Die Milchstraße hat damit
eine flache Rotationskurve von ~220 km/s zwischen 8 und 50 kpc. Lokale Gruppe danach: LMC-Apozentrum 130 kpc,
Verschmelzung nach ~4,5 Gyr, erste MW-M31-Begegnung nach 4,8 Gyr bei 65 kpc. Die Werte liegen im Bereich
veröffentlichter Modelle.

## Konsequenzen
- Doppelte Teilchenzahl für gleich viele sichtbare Sterne (`darkFraction` ~0,4–0,5).
- DM-Teilchen sind ~10–15× schwerer als Sterne, was Scheiben durch Zweikörper-Streuung leicht aufheizt.
  Softening dämpft das.
- Energie-Diagnose bei abgeschalteter DM: nur die gravitierenden Körper zählen. Akkretion von DM-Testteilchen
  wird dabei nicht exakt verbucht (Randfall).

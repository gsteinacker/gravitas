# ADR-0015: Supernova-Rückkopplung als einmalige thermische Einspeisung

- Status: accepted
- Datum: 2026-10-29
- Tasks: T-080

## Kontext
Sterne entstehen aus dichtem, kaltem Gas (ADR-0007), aber nichts wirkt zurück. In echten Galaxien explodieren die
massereichen Sterne einer jungen Population nach 3–40 Myr als Kernkollaps-Supernovae, etwa eine pro 100 M☉
gebildeter Sterne zu je 10⁵¹ erg. Sie heizen das Gas und bremsen die Sternentstehung. Ein Sternteilchen ist eine
Population von 10⁵–10⁶ M☉ (T-070). Der Nutzer möchte Sterne entstehen und explodieren sehen und die Raten
ablesen.

## Entscheidung
Jedes aus Gas entstandene Sternteilchen explodiert genau einmal und gibt e_SN · m als Wärme an seine 32 nächsten
Gasteilchen ab.

- Verzögerung gleichverteilt in [3, 40] Myr, deterministisch aus der ID (reproduzierbar, auch nach dem Laden,
  ohne zusätzlichen Zustand). Eine Population explodiert in dem Schritt, dessen Intervall [t, t + Δt) ihren
  Zeitpunkt enthält. So gibt es genau ein Ereignis.
- e_SN = 10⁴⁹ erg/M☉ (Szenenfeld `supernovaErgPerMsun`, umgerechnet in `params.gas.snEnergy`). Alle Gasteilchen
  bekommen dasselbe Δu = E / Σm. Die Energie wird in `ExternalEnergy` verbucht, die Energiebilanz bleibt prüfbar.
- Nachbarsuche linear über das Gas: Explosionen sind selten (einige pro tausend Schritte), der Hot Loop bleibt
  unberührt.
- Darstellung: Die Population blitzt etwa 3 Myr lang als heller Punkt auf. Das ist eine Markierung und im
  Lernthema als solche benannt.
- Raten (Sternentstehung in M☉/Jahr, Supernovae pro Jahrhundert, Gasanteil) misst `app.rateMeter` gleitend über
  100 Myr und zeigt der Inspektor.

## Alternativen
- **Kinetische Rückkopplung** (Teilchen nach außen stoßen, Dalla Vecchia & Schaye 2008): Sie wirkt bei grober
  Auflösung stärker. Dafür braucht man eine Impulsbilanz und Parameter für die Windgeschwindigkeit. Wir sehen sie
  vor, falls die thermische Variante zu schwach bleibt.
- **Verzögerte Kühlung** (Stinson et al. 2006): Sie lindert Overcooling, verlangt aber pro Teilchen einen
  Zeitgeber und eine Kühlfunktion, die das Modell noch nicht hat.
- **Viele kleine Ereignisse pro Population** über 3–40 Myr: Sie wären genauer. Bei der Auflösung macht das aber
  kaum einen Unterschied, und der Blitz wäre nicht mehr als Ereignis erkennbar.
- **Effektives Mehrphasenmodell** (Springel & Hernquist 2003): Es ist der Standard in Kosmologie-Codes, aber zu
  aufwendig und für ein Spiel zu wenig anschaulich.

## Konsequenzen
- Tests: `TestSupernovaeExplodeOnceAndBookEnergy` (genau eine Explosion je junger Population, keine für alte
  Sterne, Energie verbucht), `TestSupernovaDelayInRangeAndReproducible`, `TestNoSupernovaeWithoutEnergy`.
- Die Sternentstehung sinkt leicht (Szene 1, 20k, 200 Myr: 20 → 14 neue Sterne). Sie ist ohnehin etwa zehnmal
  niedriger als in der Milchstraße, und ohne Gaszufuhr sinkt sie weiter (T-081).
- Heißes Gas verkürzt den Courant-Schritt etwas. Bisher waren keine zusätzlichen Gas-Teilschritte nötig.

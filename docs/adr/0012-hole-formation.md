# ADR-0012: Schwarze Löcher aus Kollaps (Hoop-Kriterium auf aufgelöster Skala)

- Status: accepted
- Datum: 2026-10-26
- Tasks: T-067

## Kontext
Bisher entstehen Schwarze Löcher nur aus Szenen und durch den Nutzer. Gaswolken (T-063) und dichte Haufen können
aber so stark kollabieren, dass physikalisch ein Loch entstünde. Die Simulation löst nichts unterhalb der Glättung ε
auf. Ein Kriterium auf kleineren Skalen würde Löcher aus numerischem Verklumpen erzeugen.

## Entscheidung
Wir verwenden das Hoop-Kriterium auf der kleinsten aufgelösten Kugel R = 2ε: Liegt darin die Masse
M ≥ R c²/2 (also r_s(M) ≥ R) in mindestens 20 Teilchen, verschmelzen diese Teilchen zu einem Loch.

- Die Suche läuft alle 10 Schritte. Sie wird nur ausgeführt, wenn die Gesamtmasse aller Nicht-Löcher die
  kritische Masse übersteigt, bei echtem c also praktisch nie. Ein Gitter der Weite R wird sortiert, geprüft
  werden Kugeln um die Schwerpunkte der schwersten Zellen.
- Verschmelzen wie bei der Akkretion: Ein neues Loch mit Masse 0 im Schwerpunkt verschluckt die Teilchen.
  Masse und Impuls bleiben erhalten, die Energieänderung wird exakt in ExternalEnergy verbucht, und der
  Drehimpuls um den Schwerpunkt wird zum Spin (T-071, begrenzt auf a* = 0,998).
- Die Szene „Direct collapse“ zeigt den Effekt mit c = 15 statt 45 700 (r_s um 10⁷ überhöht, vgl. ADR-0003).

## Alternativen
- **Kriterium auf Teilchenebene** (Dichte eines SPH-Teilchens über einer Schwelle): Es misst nicht auf
  aufgelöster Skala und erzeugt Löcher aus Rauschen.
- **Subgrid-Sternkollaps** (massereiche Sterne werden nach ihrer Lebensdauer zu Löchern): Sternteilchen sind
  Populationen von 10⁵ M☉ (T-070), nicht einzelne Sterne. Sinnvoll nur in Szenen mit Einzelsternen, als
  Ausblick.
- **Sink-Teilchen** wie in Sternentstehungscodes (Bate et al. 1995): Dafür wären Akkretionsradius und
  Bindungstests nötig. Es ist mehr Aufwand, und die Rolle übernimmt die vorhandene Akkretion am Horizont.

## Konsequenzen
- Neue Invarianten-Tests: Masse, Impuls, Drehimpuls und Energiebilanz bei der Entstehung
  (`TestCollapseFormsHoleAndConserves`). Unaufgelöste Klumpen und echtes c bilden keine Löcher
  (`TestNoCollapseWithoutResolutionOrMass`), die eingebauten Szenen ebenfalls nicht
  (`TestBuiltinScenesFormNoHoles`).
- Mit stark verkleinertem c können auch Galaxienkerne kollabieren. Das ist gewollt und folgt aus derselben
  Übertreibung wie die großen Horizonte.
- Kosten: eine Summe O(N) alle 10 Schritte; die Suche selbst nur, wenn ein Kollaps möglich ist.

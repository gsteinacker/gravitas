# ADR-0003: Reduzierte, einstellbare Lichtgeschwindigkeit mit billigen Näherungen

- Status: accepted
- Datum: 2026-10-23
- Tasks: T-012 … T-015, T-020

## Kontext
Die Lichtgeschwindigkeit soll berücksichtigt werden, die Simulation aber möglichst groß bleiben.
Volle ART oder retardierte Felder für alle Paare sind nicht bezahlbar.

## Entscheidung
`c` ist ein Spielparameter. Wir modellieren:
1. Geschwindigkeitslimit über Eigengeschwindigkeit `u = γv`,
2. Ereignishorizont `r_s = 2GM/c²` mit Absorption,
3. Paczyński-Wiita-Potential nur für BH-Quellen,
4. optional retardierte Gravitation nur für BH-Quellen (Spike).
Details siehe [physics.md](../physics.md).

## Alternativen
- Post-Newton-Terme (1PN) für alle Paare: zu teuer, und im Baum nicht sauber nähern.
- Ignorieren: widerspricht der Anforderung.

## Konsequenzen
- Für normale Sterne praktisch null Zusatzkosten, für BHs O(N · N_BH).
- Effekte sind qualitativ richtig, quantitativ nicht. Das wird im UI ehrlich so benannt.

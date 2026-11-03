# Galaxy

Interaktive N-Body-Simulation: Zehntausende Sterne, Schwarze Löcher, Sternhaufen, Gaswolken und ganze Galaxien
per Mausklick, einstellbare Lichtgeschwindigkeit. Dazu Gas (SPH) mit Kühlung, Sternentstehung und Supernovae,
rotierende Löcher (Kerr, Frame Dragging) mit Akkretionsscheiben, Dunkle Materie und ein mitbewegter Modus mit
kosmischer Expansion für das kosmische Netz.
Native Mac-App (SwiftUI + Metal) mit Physik-Kern in Go: Menüleiste, Inspektor, Szenenübersicht, Lern-Seitenleiste
und Einstellungen mit Erklärung zu jedem Regler, Deutsch und Englisch.

## Installieren (macOS 14 oder neuer, Apple Silicon und Intel)

1. `Galaxy-<Version>.dmg` öffnen und **Galaxy** auf **Applications** (Programme) ziehen.
2. Beim ersten Start meldet macOS, dass die App nicht geprüft werden kann. Sie ist (noch) nicht bei Apple
   notarisiert. Einmalig freigeben:
   - **macOS 15 und neuer:** Galaxy starten und die Meldung schließen. Dann unter *Systemeinstellungen →
     Datenschutz & Sicherheit* bei „Galaxy wurde blockiert“ auf **Trotzdem öffnen** klicken.
   - **macOS 14:** Im Finder bei gedrückter ctrl-Taste auf Galaxy klicken, dann **Öffnen** wählen.
3. Beim ersten Start misst Galaxy etwa 0,1 s lang die Leistung des Rechners. Daraus wählt es die Sternzahl für
   flüssige 30 Schritte/s. Danach passt es Auflösung und Genauigkeit laufend an. In den *Einstellungen* (⌘,)
   lassen sich das Profil, eine feste Sternzahl oder die Grafikkarte wählen, dort gibt es auch „Leistung neu
   messen“.

Speicherstände (`.galaxy`) öffnen sich per Doppelklick. Einstellungen und Stände liegen unter
`~/Library/Application Support/Galaxy`.

**Bauen** (braucht Xcode mit Metal-Toolchain):
- `make app` erzeugt `bin/Galaxy.app` (Universal, ad-hoc signiert).
- `make dmg` erzeugt das Disk-Image.
- Mit einem Apple-Developer-Zertifikat signiert und notarisiert
  `make notarize SIGN_ID="Developer ID Application: …" NOTARY_PROFILE=…`. Das Profil wird vorher einmal mit
  `xcrun notarytool store-credentials` angelegt.

## Entwickeln

```sh
make run                                        # Start (Sternzahl nach Leistungsprofil, Spiralgalaxie)
make run ARGS="-n 50000 -c 2.5"                 # kleines c: das zentrale Loch frisst die Galaxie
make run ARGS="-preset 5"                       # Lokale Gruppe in echten Abständen (x/v: Dunkle Materie)
make run ARGS="-scene $PWD/meine.json"          # eigene Szene (docs/scenes.md); -load …/x.galaxy lädt einen Stand
make run ARGS="-preset 4 -c 3"                  # Doppel-Loch mit Linseneffekt
make run ARGS="-preset 2 -fx glow,doppler,trails"   # Kollision mit Bahnspuren
make run ARGS="-preset 14"                      # Galaxienentstehung aus Halo-Gas (Szenen ab 10: Menü / ⇧⌘N)
make shot PRESET=9                              # Screenshot bin/shot.png mit Messzeile
make plan                                       # Fortschritt im Backlog
make check                                      # fmt, vet, Tests, Backlog-Validierung (Go-Kern)
```

Alle Aktionen stehen auch in der Menüleiste und im Inspektor (⌥⌘I). `?` oder `F1` zeigt die Übersicht in der App.

**Szenen** ([Format und Quellen](docs/scenes.md)), im Menü *Ablage → Neu aus Szene* und in der Szenenübersicht
(⇧⌘N) nach Gruppen. Die ersten neun liegen auf den Zifferntasten, alle sind per `-preset N` startbar.

| Gruppe | Szenen |
|--------|--------|
| Galaxien | `1` Spiralgalaxie, `2` Galaxienkollision, `5` Lokale Gruppe, `6` Antennen, `7` Whirlpool (M51), 14 Galaxienentstehung, 15 Satelliten der Milchstraße (LMC, SMC, Sagittarius), 16 Balkenspirale |
| Sterne und Schwarze Löcher | `3` Sternhaufen, `4` Loch-Doppel im Haufen, `8` S-Sterne um Sgr A\* |
| Gas und Sternentstehung | `9` Pferdekopfnebel, 10 Direkter Kollaps, 11 Akkretionsscheibe |
| Kosmos | 12 Kosmisches Netz, 13 Strukturbildung (mitbewegt, mit Expansion) |

| Eingabe | Aktion |
|---------|--------|
| `1`–`9` / ⇧⌘N | Szene laden / Szenenübersicht |
| `b` `k` `w` `a` `Tab` | Werkzeug: Schwarzes Loch / Sternhaufen / Gaswolke / Galaxie / wechseln (`m` `n` ändern die Masse) |
| Klick | Startrampe: Objekt auf der Scheibenebene, Griffe ziehen (gelb mit/gegen die Drehung, cyan außen/innen, magenta oben/unten), `↩` startet, `⎋` verwirft; ⌘ beim Ziehen ohne Einrasten |
| Doppelklick / Objekt ziehen / ⌥-Ziehen | Sofort platzieren / Schleuder / Richtung und Tempo ziehen und starten |
| Ziehen / ⇧- oder Rechts-Ziehen / Rad, Pinch | Rotieren / Verschieben / Zoom |
| `<` `>` | Lichtgeschwindigkeit verringern / erhöhen (kleines c → große Horizonte) |
| `Leertaste` `.` `+` `-` `r` | Pause, Einzelschritt, Tempo, Neustart |
| `e` | **Lernmodus**: Einführung zur Szene und Erklärungen (←/→ Thema, ↑/↓ Zielgruppe: Schüler, Interessierte, Physik) |
| `x` `v` | Dunkle Materie: Gravitation an/aus, sichtbar machen |
| `F5` `F9` | Spielstand speichern / neuesten laden (`~/Library/Application Support/Galaxy/saves`) |
| `o` `u` | Gaskarte (Falschfarben) / Staubabsorption |
| `p` | Gravitation der Löcher: instantan / retardiert / retardiert + v (Ausbreitung mit c) |
| `g` `t` `l` `d` | Effekte: Glow, Spuren, Gravitationslinse, Doppler-Farben |
| `h` `?`/`F1` `f` `c` | Statusanzeige, Tastenübersicht, Vollbild, Ansicht zentrieren |

Doku: [Stack-Analyse](docs/analysis/01-tech-stack.md) · [Architektur](docs/architecture.md) ·
[Physik](docs/physics.md) · [Szenen](docs/scenes.md) · [ADRs](docs/adr) ·
[Performance-Log](docs/perf-log.md) · [Roadmap](planning/ROADMAP.md)

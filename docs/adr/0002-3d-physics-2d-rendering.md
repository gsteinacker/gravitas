# ADR-0002: 3D-Physik, projiziert auf eine 2D-Kamera

- Status: accepted
- Datum: 2026-10-23
- Tasks: T-002, T-004

## Kontext
2D-Simulationen sind einfacher, aber Scheibengalaxien, Kollisionen und Orbits um Schwarze Löcher wirken in
3D deutlich eindrucksvoller. Ebitengine ist eine 2D-Engine.

## Entscheidung
Die Physik rechnet in 3D (Octree). Das Rendering projiziert perspektivisch auf den Bildschirm. Die Kamera
lässt sich rotieren, verschieben und zoomen. Platzierungen per Maus liegen in der Fokusebene der Kamera.

## Alternativen
- Reine 2D-Physik (Quadtree): ~30 % schneller, aber visuell flacher und physikalisch „falsche“ 1/r²-Welt in 2D.
- Echtes 3D-Rendering (OpenGL/Metal direkt): mehr Aufwand, kaum Mehrwert für Punktwolken.

## Konsequenzen
- Tiefe muss visuell kommuniziert werden (Größe/Helligkeit nach Abstand, Rotation).
- Szenen-Presets können 2D-artig starten (dünne Scheibe), ohne Sonderfall im Code.

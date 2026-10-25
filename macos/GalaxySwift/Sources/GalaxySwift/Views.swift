import AppKit
import CGalaxy
import SwiftUI
import UniformTypeIdentifiers

extension UTType {
    static let galaxyState = UTType(exportedAs: "de.steinacker.galaxy.state", conformingTo: .data)
}

struct ContentView: View {
    var engine: Engine
    let renderer: Renderer
    @Binding var showInspector: Bool
    @Binding var showLearning: Bool

    var body: some View {
        NavigationSplitView(columnVisibility: Binding(
            get: { showLearning ? .all : .detailOnly },
            set: { showLearning = $0 != .detailOnly })
        ) {
            LearningSidebar(engine: engine)
                .navigationSplitViewColumnWidth(min: 260, ideal: 340, max: 520)
        } detail: {
            ZStack(alignment: .bottom) {
                MetalView(engine: engine, renderer: renderer)
                LaunchPadOverlay(engine: engine, pad: renderer.pad)
                if engine.state.showHUD != 0 {
                    StatusOverlay(engine: engine).frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
                }
                if !engine.message.isEmpty {
                    Text(engine.message)
                        .padding(.horizontal, 14).padding(.vertical, 8)
                        .background(.regularMaterial, in: Capsule())
                        .padding(.bottom, 24)
                        .transition(.opacity)
                }
            }
            .navigationTitle(engine.sceneName)
            .navigationSubtitle(engine.sceneInfo)
        }
        .inspector(isPresented: $showInspector) {
            InspectorView(engine: engine)
                .inspectorColumnWidth(min: 280, ideal: 320, max: 420)
        }
        .sheet(isPresented: Binding(get: { engine.state.showHelp != 0 }, set: { if !$0 && engine.state.showHelp != 0 { engine.action("help") } })) {
            KeyOverview(engine: engine)
        }
        .onChange(of: showLearning) { _, on in engine.set("explain", on ? 1 : 0) }
        .onChange(of: engine.state.explain) { _, on in showLearning = on != 0 }
    }
}

extension LabeledContent where Label == Text, Content == Text {
    /// verbatim shows the value as is (no Markdown, no localization lookup).
    init(_ title: String, verbatim value: String) {
        self.init { Text(verbatim: value) } label: { Text(verbatim: title) }
    }
}

struct InspectorView: View {
    var engine: Engine

    private var s: GalaxyState { engine.state }

    var body: some View {
        Form {
            if engine.launch.active != 0 { LaunchSection(engine: engine) }
            Section(L("Simulation", "Simulation")) {
                LabeledContent(L("Szene", "Scene"), verbatim: engine.sceneName)
                HStack {
                    Button(s.paused != 0 ? L("Weiter", "Resume") : L("Pause", "Pause")) { engine.action("pause") }
                    Button(L("Schritt", "Step")) { engine.action("step") }
                    Button(L("Neustart", "Restart")) { engine.action("restart") }
                }
                Picker(L("Schritte/s", "Steps/s"), selection: Binding(get: { Int(s.speed) }, set: { engine.set("speed", Double($0)) })) {
                    ForEach(Array(["10", "30", "60", "120", "240", "max"].enumerated()), id: \.offset) { Text($1).tag($0) }
                }
                .pickerStyle(.segmented)
                LabeledContent(L("Zeit", "Time"), verbatim: timeText)
                LabeledContent(L("Leistung", "Performance"),
                               verbatim: String(format: L("%.0f Schritte/s · %.0f FPS", "%.0f steps/s · %.0f FPS"), s.sps, s.fps))
                LabeledContent(L("Körper", "Bodies"), verbatim: "\(s.bodies) · " + String(format: L("Auflösung %g×", "resolution %g×"), s.renderScale))
                LabeledContent(L("Energiedrift", "Energy drift"), verbatim: driftText)
            }
            Section(L("Physik", "Physics")) {
                LabeledContent {
                    Slider(value: Binding(get: { log10(max(s.c, 1)) }, set: { engine.set("c", pow(10, $0)) }), in: 0...3)
                    Text(verbatim: s.c > 0 ? String(format: "%.4g", s.c) : "∞").monospacedDigit().lineLimit(1).frame(width: 72, alignment: .trailing)
                } label: {
                    Text(verbatim: "c").help(L("Lichtgeschwindigkeit (Simulationseinheiten)", "Speed of light (simulation units)"))
                }
                .disabled(s.c <= 0)
                Picker(L("Loch-Gravitation", "Hole gravity"), selection: Binding(get: { Int(s.gravity) }, set: { engine.set("gravity", Double($0)) })) {
                    Text(L("sofort", "instant")).tag(0)
                    Text(L("retardiert", "retarded")).tag(1)
                    Text(L("ret. + v", "ret. + v")).tag(2)
                }
                .pickerStyle(.segmented)
                Toggle(L("Dunkle Materie wirkt", "Dark matter gravity"),
                       isOn: Binding(get: { s.darkOff == 0 }, set: { engine.set("darkOff", $0 ? 0 : 1) }))
                    .disabled(s.darkBodies == 0)
                LabeledContent {
                    Slider(value: Binding(get: { s.thetaUser }, set: { engine.set("theta", $0) }), in: 0.5...1.0)
                    Text(String(format: "%.2f%@", s.theta, s.theta > s.thetaUser + 1e-9 ? "*" : ""))
                        .monospacedDigit().frame(width: 52, alignment: .trailing)
                } label: {
                    Text(verbatim: "θ").help(L("Öffnungswinkel des Barnes-Hut-Baums: kleiner ist genauer, * = vom Regler angehoben",
                                               "Barnes-Hut opening angle: smaller is more accurate, * = raised by the governor"))
                }
            }
            Section(L("Werkzeug (Klick)", "Tool (click)")) {
                Picker("", selection: Binding(get: { Int(s.tool) }, set: { engine.set("tool", Double($0)) })) {
                    Text(L("Schwarzes Loch", "Black hole")).tag(0)
                    Text(L("Sternhaufen", "Star cluster")).tag(1)
                    Text(L("Gaswolke", "Gas cloud")).tag(2)
                }
                .pickerStyle(.segmented)
                .labelsHidden()
                SliderRow(title: L("Masse", "Mass"), value: Binding(get: { log10(s.mass) }, set: { engine.set("mass", pow(10, $0)) }),
                          range: -2...log10(20), text: engine.massText(s.mass))
                if s.tool == 2 {
                    SliderRow(title: L("Temperatur", "Temperature"),
                              help: L("Temperatur des Gases: kalt (10–100 K) kollabiert, heiß (10⁴ K) bleibt aufgebläht",
                                      "gas temperature: cold (10–100 K) collapses, hot (10⁴ K) stays puffed up"),
                              value: Binding(get: { log10(s.cloudK) }, set: { engine.set("cloudK", pow(10, $0)) }), range: 1...4,
                              text: String(format: "%.0f K", s.cloudK))
                    SliderRow(title: L("Turbulenz", "Turbulence"),
                              help: L("Wirbelbewegung als Anteil von √(GM/R): unter ~0,6 gebunden (Kollaps, Sternentstehung), darüber zerstreut sie sich",
                                      "turbulent motion as a fraction of √(GM/R): below ~0.6 bound (collapse, star formation), above it disperses"),
                              value: Binding(get: { s.cloudTurb }, set: { engine.set("cloudTurb", $0) }), range: 0...2,
                              text: String(format: L("%.2f × virial", "%.2f × virial"), s.cloudTurb))
                }
            }
            Section(L("Darstellung", "View")) {
                Toggle(L("Leuchten", "Glow"), isOn: engine.fxBinding(Engine.glow))
                Toggle(L("Spuren", "Trails"), isOn: engine.fxBinding(Engine.trails))
                Toggle(L("Gravitationslinse", "Lensing"), isOn: engine.fxBinding(Engine.lens))
                Toggle(L("Doppler-Farben", "Doppler colors"), isOn: engine.fxBinding(Engine.doppler))
                Toggle(L("Staub", "Dust"), isOn: engine.fxBinding(Engine.dust))
                Toggle(L("Gas zeigen", "Show gas"), isOn: engine.fxBinding(Engine.gas))
                Toggle(L("Dunkle Materie zeigen", "Show dark matter"), isOn: engine.fxBinding(Engine.dark))
            }
        }
        .formStyle(.grouped)
    }

    private var timeText: String {
        guard s.yearsPer > 0 else { return String(format: "t = %.2f", s.time) }
        let y = s.time * s.yearsPer
        let (v, unit) = y >= 1e9 ? (y / 1e9, "Gyr") : y >= 1e6 ? (y / 1e6, "Myr") : y >= 1e3 ? (y / 1e3, "kyr") : (y, "yr")
        return String(format: "t = %.2f (%.3g %@)", s.time, v, unit)
    }

    private var driftText: String {
        if s.driftQuality == 0 || s.drift.isNaN { return L("wird gemessen …", "measuring…") }
        return String(format: s.driftQuality == 1 ? "≈ %+.0e" : "%+.1e", s.drift)
    }
}

/// LaunchSection plans the start of a placed object (T-064): plane, height, direction and speed
/// relative to the galaxy, with the circular and escape speed at the position as references.
/// SliderRow is a slider with a fixed label column, so labels do not wrap in the narrow inspector.
struct SliderRow: View {
    let title: String
    var help = ""
    @Binding var value: Double
    let range: ClosedRange<Double>
    let text: String

    var body: some View {
        HStack(spacing: 8) {
            Text(title).frame(width: 64, alignment: .leading).help(help)
            Slider(value: $value, in: range)
            Text(verbatim: text).monospacedDigit().lineLimit(1).minimumScaleFactor(0.7).frame(width: 84, alignment: .trailing)
        }
    }
}

struct LaunchSection: View {
    var engine: Engine

    private var l: GalaxyLaunch { engine.launch }

    /// row is a slider with a fixed label column, so labels do not wrap in the narrow inspector.
    private func row(_ title: String, _ help: String? = nil, value: Binding<Double>, range: ClosedRange<Double>, text: String) -> some View {
        HStack(spacing: 8) {
            Text(title).frame(width: 64, alignment: .leading).help(help ?? "")
            Slider(value: value, in: range)
            Text(verbatim: text).monospacedDigit().lineLimit(1).minimumScaleFactor(0.8).frame(width: 66, alignment: .trailing)
        }
    }

    var body: some View {
        let maxSpeed = max(1.6 * l.escape, 1.2 * l.speed, 1e-3)
        let maxHeight = max(1.5 * l.radius, 0.5, abs(l.height))
        Section(L("Start", "Launch")) {
            Picker(L("Ebene", "Plane"), selection: Binding(get: { Int(l.plane) }, set: { engine.setLaunch("plane", Double($0)) })) {
                Text(L("Scheibe", "Disk")).tag(0)
                Text(L("Bildebene", "View plane")).tag(1)
            }
            .pickerStyle(.segmented)
            row(L("Höhe", "Height"), L("Abstand von der Ebene", "distance from the plane"),
                value: Binding(get: { l.height }, set: { engine.setLaunch("height", $0) }), range: -maxHeight...maxHeight,
                text: engine.lengthText(l.height))
            row(L("Richtung", "Direction"),
                L("0° mit der Drehung der Scheibe (Kreisbahn), 90° nach außen, ±180° gegen die Drehung, −90° nach innen",
                  "0° along the rotation of the disk (circular orbit), 90° outward, ±180° against it, −90° inward"),
                value: Binding(get: { l.azimuth * 180 / .pi }, set: { engine.setLaunch("azimuth", $0 * .pi / 180) }), range: -180...180,
                text: String(format: "%.0f°", l.azimuth * 180 / .pi))
            row(L("Neigung", "Elevation"), L("Winkel aus der Ebene heraus", "angle out of the plane"),
                value: Binding(get: { l.elevation * 180 / .pi }, set: { engine.setLaunch("elevation", $0 * .pi / 180) }), range: -90...90,
                text: String(format: "%.0f°", l.elevation * 180 / .pi))
            row(L("Tempo", "Speed"), L("relativ zur Galaxie, deren Zentrum am nächsten liegt", "relative to the galaxy whose center is nearest"),
                value: Binding(get: { l.speed }, set: { engine.setLaunch("speed", $0) }), range: 0...maxSpeed,
                text: engine.speedText(l.speed))
            Text(verbatim: String(format: L("Abstand %@ · Kreisbahn %@ · Flucht %@ · Stoßparameter %@",
                                            "distance %@ · circular %@ · escape %@ · impact parameter %@"),
                                  engine.lengthText(l.radius), engine.speedText(l.circular), engine.speedText(l.escape),
                                  engine.lengthText(l.impact)))
                .font(.callout).foregroundStyle(.secondary)
                .help(L("Stoßparameter: wie knapp die Startrichtung geradlinig am Zentrum vorbeiführt",
                        "impact parameter: how closely the launch direction passes the center on a straight line"))
            HStack {
                Button(L("Ruhe", "Rest")) { engine.launchPreset(0) }
                Button(L("Kreisbahn", "Circular")) { engine.launchPreset(1) }
                Button(L("Flucht", "Escape")) { engine.launchPreset(2) }
                Button(L("Zentrum", "Center")) { engine.launchPreset(3) }
                Button(L("Doppel", "Binary")) { engine.launchPreset(4) }.disabled(l.holeAttractor == 0)
            }
            .controlSize(.small)
            HStack {
                Button(L("Starten", "Launch")) { engine.launchGo() }.keyboardShortcut(.defaultAction)
                Button(L("Abbrechen", "Cancel")) { engine.launchCancel() }
            }
        }
    }
}

struct LearningSidebar: View {
    var engine: Engine

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            Picker(L("Zielgruppe", "Audience"), selection: Binding(get: { Int(engine.state.level) }, set: { engine.set("level", Double($0)) })) {
                Text(L("Schüler", "Pupils")).tag(0)
                Text(L("Interessierte", "Laypeople")).tag(1)
                Text(L("Physik", "Physics")).tag(2)
            }
            .pickerStyle(.segmented)
            .labelsHidden()
            .padding(10)
            List(selection: Binding(get: { engine.topicIndex }, set: { if let i = $0 { engine.set("topic", Double(i)) } })) {
                ForEach(Array(engine.topics.enumerated()), id: \.offset) { i, t in Text(t).tag(i) }
            }
            .frame(minHeight: 140, maxHeight: 260)
            Divider()
            ScrollView {
                VStack(alignment: .leading, spacing: 10) {
                    Text(engine.explanationTitle).font(.title2.bold())
                    Text(engine.explanationBody).textSelection(.enabled)
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                .padding()
            }
        }
    }
}

/// groupDigits writes 50000 as "50 000" with a no-break space.
func groupDigits(_ n: Int) -> String {
    var s = String(n)
    var i = s.count - 3
    while i > 0 {
        s.insert("\u{00a0}", at: s.index(s.startIndex, offsetBy: i))
        i -= 3
    }
    return s
}

struct SettingsView: View {
    var engine: Engine
    @AppStorage("profile") private var profile = ""

    var body: some View {
        Form {
            Section(L("Leistung", "Performance")) {
                Picker(L("Profil", "Profile"), selection: $profile) {
                    Text(L("Automatisch", "Automatic")).tag("")
                    Text(L("Niedrig", "Low")).tag("low")
                    Text(L("Mittel", "Medium")).tag("medium")
                    Text(L("Hoch", "High")).tag("high")
                    Text(L("Maximal", "Maximum")).tag("max")
                }
                .onChange(of: profile) { _, p in engine.setProfile(p) }
                Picker(L("Sterne pro Szene", "Stars per scene"),
                       selection: Binding(get: { Int(engine.settings.stars) }, set: { engine.setStars($0) })) {
                    ForEach(Engine.starChoices, id: \.self) { n in
                        Text(verbatim: n == 0 ? L("Automatisch", "Automatic") : groupDigits(n)).tag(n)
                    }
                }
                Picker(L("Grafikkarte (Metal)", "GPU (Metal)"),
                       selection: Binding(get: { Int(engine.settings.gpu) }, set: { engine.setGPU($0) })) {
                    Text(L("Automatisch", "Automatic")).tag(0)
                    Text(L("Ein", "On")).tag(1)
                    Text(L("Aus", "Off")).tag(2)
                }
                LabeledContent(L("Aktuell", "Current"),
                               verbatim: String(format: L("%@ Sterne, ab nächster Szene", "%@ stars, from the next scene"), groupDigits(Int(engine.state.stars))))
                LabeledContent(L("Rechner", "Computer"), verbatim: engine.hardware)
                LabeledContent(L("Messung", "Measurement"),
                               verbatim: String(format: L("%@ Sterne bei 30 Schritten/s", "%@ stars at 30 steps/s"), groupDigits(Int(engine.settings.capacity))))
                Button(L("Leistung neu messen", "Measure Again")) { engine.recalibrate() }
            }
            Section(L("Lernmodus", "Learning mode")) {
                Picker(L("Zielgruppe", "Audience"), selection: Binding(get: { Int(engine.state.level) }, set: { engine.set("level", Double($0)) })) {
                    Text(L("Schüler", "Pupils")).tag(0)
                    Text(L("Interessierte", "Laypeople")).tag(1)
                    Text(L("Physik", "Physics")).tag(2)
                }
            }
        }
        .formStyle(.grouped)
        .frame(width: 480)
    }
}

/// KeyOverview lists the keys of the simulation view (Help menu, "?" or F1).
struct KeyOverview: View {
    var engine: Engine

    private let keys: [(String, String)] = [
        ("1 – 9", L("Szene laden", "load scene")),
        (L("Klick", "Click"), L("Startrampe öffnen: Objekt auf der Scheibenebene mit Griffen", "open the launch pad: object on the disk plane with handles")),
        (L("Objekt ziehen", "Drag object"), L("Schleuder: zurückziehen und loslassen startet", "slingshot: pull back and release to launch")),
        (L("Griffe ziehen", "Drag handles"), L("gelb mit/gegen die Drehung, cyan außen/innen, magenta oben/unten, Spitze frei", "yellow pro/retrograde, cyan out/in, magenta up/down, tip freely")),
        (L("Ring, ↕-Knopf", "Ring, ↕ knob"), L("auf der Ebene verschieben, Höhe", "move on the plane, height")),
        (L("⌘ beim Ziehen", "⌘ while dragging"), L("ohne Einrasten an Kreisbahn, Flucht, Achsen", "no snapping to circular, escape, axes")),
        (L("↩  ⎋", "↩  ⎋"), L("starten / verwerfen", "launch / discard")),
        (L("Doppelklick", "Double click"), L("sofort platzieren", "place at once")),
        (L("⌥ + Ziehen", "⌥ + drag"), L("Richtung und Tempo ziehen und starten", "drag direction and speed, then launch")),
        ("b  k  w  ⇥", L("Werkzeug: Loch / Sternhaufen / Gaswolke / wechseln", "tool: hole / cluster / gas cloud / switch")),
        ("m  n", L("Masse + / −", "mass + / −")),
        (L("Ziehen", "Drag"), L("drehen", "rotate")),
        (L("⇧ / rechts ziehen", "⇧ / right drag"), L("verschieben", "pan")),
        (L("Mausrad, Pinch", "Wheel, pinch"), L("zoomen", "zoom")),
        (L("Leertaste", "Space"), L("Pause", "pause")),
        (".", L("Einzelschritt", "single step")),
        ("+  −", L("Tempo", "speed")),
        ("<  >", L("Lichtgeschwindigkeit", "speed of light")),
        ("p", L("Loch-Gravitation: sofort / retardiert / retardiert + v", "hole gravity: instant / retarded / retarded + v")),
        ("x  v", L("Dunkle Materie: Gravitation an/aus, zeigen", "dark matter: gravity on/off, show")),
        ("g  t  l  d", L("Leuchten, Spuren, Linse, Doppler", "glow, trails, lens, Doppler")),
        ("o  u", L("Gaskarte, Staub", "gas map, dust")),
        ("e", L("Lernmodus (←/→ Thema, ↑/↓ Zielgruppe)", "learning mode (←/→ topic, ↑/↓ audience)")),
        ("r  c", L("Neustart, Ansicht zentrieren", "restart, center view")),
        ("h", L("Statusanzeige im Bild", "status overlay")),
        ("F5  F9", L("schnell sichern / letzten Stand laden", "quick save / load latest")),
        ("?  F1", L("diese Übersicht", "this overview")),
    ]

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text(L("Tasten und Maus", "Keys and mouse")).font(.title2.bold())
            Grid(alignment: .leading, horizontalSpacing: 18, verticalSpacing: 6) {
                ForEach(keys, id: \.0) { k, v in
                    GridRow {
                        Text(verbatim: k).font(.body.monospaced()).foregroundStyle(.orange)
                        Text(verbatim: v)
                    }
                }
            }
            HStack {
                Spacer()
                Button(L("Schließen", "Close")) { engine.action("help") }.keyboardShortcut(.defaultAction)
            }
        }
        .padding(24)
        .frame(minWidth: 520)
    }
}

/// StatusOverlay is the optional status text in the picture (key h), like the HUD of the Go app.
struct StatusOverlay: View {
    var engine: Engine

    var body: some View {
        let s = engine.state
        VStack(alignment: .leading, spacing: 3) {
            Text(verbatim: engine.sceneName + (s.paused != 0 ? L("   [PAUSE]", "   [PAUSED]") : "")).bold()
            if !engine.sceneInfo.isEmpty { Text(verbatim: engine.sceneInfo) }
            Text(verbatim: String(format: L("%@ Körper (Schwarze Löcher: %d) · Dunkle Materie %d · Gas %d · neue Sterne %d",
                                            "%@ bodies (black holes: %d) · dark matter %d · gas %d · new stars %d"),
                                  groupDigits(Int(s.bodies)), s.blackHoles, s.darkBodies, s.gasBodies, s.starsFormed))
            Text(verbatim: String(format: L("%.0f Schritte/s · %.0f FPS · Teilschritte %d, Gas %d · θ %.2f · Auflösung %g×",
                                            "%.0f steps/s · %.0f FPS · substeps %d, gas %d · θ %.2f · resolution %g×"),
                                  s.sps, s.fps, s.substeps, max(1, s.hydroSubsteps), s.theta, s.renderScale))
            Text(verbatim: s.c > 0 ? String(format: "c = %.4g", s.c) : L("c = unendlich (Newton)", "c = infinite (Newtonian)"))
        }
        .font(.callout.monospacedDigit())
        .padding(10)
        .background(.black.opacity(0.45), in: RoundedRectangle(cornerRadius: 8))
        .foregroundStyle(.white)
        .padding(12)
        .allowsHitTesting(false)
    }
}

import AppKit
import CGalaxy
import SwiftUI
import UniformTypeIdentifiers

extension UTType {
    static let galaxyState = UTType(exportedAs: "de.steinacker.galaxy.state", conformingTo: .data)
}

struct ContentView: View {
    @ObservedObject var engine: Engine
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
    @ObservedObject var engine: Engine

    private var s: GalaxyState { engine.state }

    var body: some View {
        Form {
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
                LabeledContent(L("Licht c", "Light c")) {
                    Slider(value: Binding(get: { log10(max(s.c, 1)) }, set: { engine.set("c", pow(10, $0)) }), in: 0...3)
                    Text(verbatim: s.c > 0 ? String(format: "%.4g", s.c) : "∞").monospacedDigit().frame(width: 52, alignment: .trailing)
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
                LabeledContent(L("Baum θ", "Tree θ")) {
                    Slider(value: Binding(get: { s.thetaUser }, set: { engine.set("theta", $0) }), in: 0.5...1.0)
                    Text(String(format: "θ %.2f%@", s.theta, s.theta > s.thetaUser + 1e-9 ? "*" : ""))
                        .monospacedDigit().frame(width: 52, alignment: .trailing)
                }
            }
            Section(L("Werkzeug (Klick)", "Tool (click)")) {
                Picker("", selection: Binding(get: { Int(s.tool) }, set: { engine.set("tool", Double($0)) })) {
                    Text(L("Schwarzes Loch", "Black hole")).tag(0)
                    Text(L("Sternhaufen", "Star cluster")).tag(1)
                }
                .pickerStyle(.segmented)
                .labelsHidden()
                LabeledContent(L("Masse", "Mass")) {
                    Slider(value: Binding(get: { log10(s.mass) }, set: { engine.set("mass", pow(10, $0)) }), in: -2...log10(20))
                    Text(String(format: "%.2f", s.mass)).monospacedDigit().frame(width: 52, alignment: .trailing)
                }
            }
            Section(L("Darstellung", "View")) {
                Toggle(L("Leuchten", "Glow"), isOn: engine.fxBinding(Engine.glow))
                Toggle(L("Spuren", "Trails"), isOn: engine.fxBinding(Engine.trails))
                Toggle(L("Gravitationslinse", "Lensing"), isOn: engine.fxBinding(Engine.lens))
                Toggle(L("Doppler-Farben", "Doppler colors"), isOn: engine.fxBinding(Engine.doppler))
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
        return String(format: s.driftQuality == 1 ? "~%+.0e" : "%+.1e", s.drift)
    }
}

struct LearningSidebar: View {
    @ObservedObject var engine: Engine

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

struct SettingsView: View {
    @ObservedObject var engine: Engine
    @AppStorage("profile") private var profile = ""

    var body: some View {
        Form {
            Picker(L("Leistungsprofil", "Performance profile"), selection: $profile) {
                Text(L("Automatisch", "Automatic")).tag("")
                Text(L("Niedrig", "Low")).tag("low")
                Text(L("Mittel", "Medium")).tag("medium")
                Text(L("Hoch", "High")).tag("high")
                Text(L("Maximal", "Maximum")).tag("max")
            }
            .onChange(of: profile) { _, p in engine.setProfile(p) }
            LabeledContent(L("Aktuell", "Current"), verbatim: String(format: L("%d Sterne, ab nächster Szene", "%d stars, from the next scene"), engine.state.stars))
            LabeledContent(L("Rechner", "Computer"), verbatim: engine.hardware)
            Picker(L("Lernmodus", "Learning mode"), selection: Binding(get: { Int(engine.state.level) }, set: { engine.set("level", Double($0)) })) {
                Text(L("Schüler", "Pupils")).tag(0)
                Text(L("Interessierte", "Laypeople")).tag(1)
                Text(L("Physik", "Physics")).tag(2)
            }
        }
        .formStyle(.grouped)
        .frame(width: 460)
    }
}

import AppKit
import CGalaxy
import SwiftUI
import UniformTypeIdentifiers

@main
struct GalaxyApp: App {
    @NSApplicationDelegateAdaptor(AppDelegate.self) var delegate
    @State private var engine: Engine
    private let renderer: Renderer
    @State private var showInspector = true
    @State private var showLearning: Bool
    @State private var importing = false

    init() {
        let e = Engine()
        if let err = e.startError {
            FileHandle.standardError.write(("galaxy: " + err + "\n").data(using: .utf8)!)
            exit(2)
        }
        _engine = State(initialValue: e)
        guard let r = Renderer(engine: e) else { fatalError("Metal is not available") }
        if let shot = e.shot {
            r.shot = (shot.url, shot.after, CFAbsoluteTimeGetCurrent())
        }
        renderer = r
        AppDelegate.engine = e
        _showLearning = State(initialValue: e.state.explain != 0)
    }

    var body: some Scene {
        WindowGroup("Galaxy") {
            ContentView(engine: engine, renderer: renderer, showInspector: $showInspector, showLearning: $showLearning)
                .frame(minWidth: 900, minHeight: 600)
                .fileImporter(isPresented: $importing, allowedContentTypes: [.galaxyState, .json]) { result in
                    guard case let .success(url) = result else { return }
                    let access = url.startAccessingSecurityScopedResource()
                    defer { if access { url.stopAccessingSecurityScopedResource() } }
                    if let err = engine.open(url) { alert(L("Datei kann nicht geöffnet werden", "Cannot open the file"), err) }
                }
        }
        .defaultSize(width: 1440, height: 900)
        .commands { commands }

        Settings { SettingsView(engine: engine) }
    }

    @CommandsBuilder private var commands: some Commands {
        CommandGroup(replacing: .newItem) {
            Menu(L("Neu aus Szene", "New from Scene")) {
                ForEach(Array(engine.presets.enumerated()), id: \.offset) { i, name in
                    Button(name) { engine.action("scene\(i)") }
                        .keyboardShortcut(i < 9 ? KeyEquivalent(Character("\(i + 1)")) : "0", modifiers: [])
                }
            }
            Button(L("Öffnen …", "Open…")) { importing = true }.keyboardShortcut("o")
            Divider()
            Button(L("Stand sichern …", "Save State…")) { save(type: .galaxyState, ext: "galaxy") { engine.save($0) } }
                .keyboardShortcut("s")
            Button(L("Schnell sichern", "Quick Save")) { engine.action("quickSave") }
                .keyboardShortcut(KeyEquivalent(Character(UnicodeScalar(NSF5FunctionKey)!)), modifiers: [])
            Button(L("Letzten Stand laden", "Load Latest")) { engine.action("quickLoad") }
                .keyboardShortcut(KeyEquivalent(Character(UnicodeScalar(NSF9FunctionKey)!)), modifiers: [])
            Divider()
            Button(L("Bild exportieren …", "Export Image…")) { save(type: .png, ext: "png") { renderer.exportURL = $0 } }
                .keyboardShortcut("e", modifiers: [.command, .shift])
        }
        CommandMenu(L("Simulation", "Simulation")) {
            Toggle(L("Pause", "Pause"), isOn: Binding(get: { engine.state.paused != 0 }, set: { engine.set("paused", $0 ? 1 : 0) }))
            Button(L("Einzelschritt", "Single Step")) { engine.action("step") }.keyboardShortcut(".", modifiers: [])
            Divider()
            Button(L("Schneller", "Faster")) { engine.action("faster") }
            Button(L("Langsamer", "Slower")) { engine.action("slower") }.keyboardShortcut("-", modifiers: [])
            Divider()
            Button(L("Szene neu starten", "Restart Scene")) { engine.action("restart") }.keyboardShortcut("r", modifiers: [])
        }
        CommandMenu(L("Physik", "Physics")) {
            Button(L("Licht schneller (>)", "Faster Light (>)")) { engine.action("lightFaster") }
            Button(L("Licht langsamer (<)", "Slower Light (<)")) { engine.action("lightSlower") }
            Divider()
            Picker(L("Gravitation der Löcher", "Black Hole Gravity"),
                   selection: Binding(get: { Int(engine.state.gravity) }, set: { engine.set("gravity", Double($0)) })) {
                Text(L("Sofort (Newton)", "Instant (Newton)")).tag(0)
                Text(L("Retardiert", "Retarded")).tag(1)
                Text(L("Retardiert + v·Δt", "Retarded + v·Δt")).tag(2)
            }
            Toggle(L("Dunkle Materie wirkt", "Dark Matter Gravity"),
                   isOn: Binding(get: { engine.state.darkOff == 0 }, set: { engine.set("darkOff", $0 ? 0 : 1) }))
                .keyboardShortcut("x", modifiers: [])
        }
        CommandMenu(L("Werkzeug", "Tool")) {
            Picker(L("Werkzeug", "Tool"), selection: Binding(get: { Int(engine.state.tool) }, set: { engine.set("tool", Double($0)) })) {
                Text(L("Schwarzes Loch", "Black Hole")).tag(0)
                Text(L("Sternhaufen", "Star Cluster")).tag(1)
                Text(L("Gaswolke", "Gas Cloud")).tag(2)
            }
            .pickerStyle(.inline)
            Divider()
            Button(L("Mehr Masse", "More Mass")) { engine.action("massUp") }.keyboardShortcut("m", modifiers: [])
            Button(L("Weniger Masse", "Less Mass")) { engine.action("massDown") }.keyboardShortcut("n", modifiers: [])
        }
        CommandGroup(after: .toolbar) {
            Toggle(L("Leuchten", "Glow"), isOn: engine.fxBinding(Engine.glow)).keyboardShortcut("g", modifiers: [])
            Toggle(L("Spuren", "Trails"), isOn: engine.fxBinding(Engine.trails)).keyboardShortcut("t", modifiers: [])
            Toggle(L("Gravitationslinse", "Lensing"), isOn: engine.fxBinding(Engine.lens)).keyboardShortcut("l", modifiers: [])
            Toggle(L("Doppler-Farben", "Doppler Colors"), isOn: engine.fxBinding(Engine.doppler)).keyboardShortcut("d", modifiers: [])
            Toggle(L("Staub", "Dust"), isOn: engine.fxBinding(Engine.dust)).keyboardShortcut("u", modifiers: [])
            Toggle(L("Gas zeigen", "Show Gas"), isOn: engine.fxBinding(Engine.gas)).keyboardShortcut("o", modifiers: [])
            Toggle(L("Dunkle Materie zeigen", "Show Dark Matter"), isOn: engine.fxBinding(Engine.dark)).keyboardShortcut("v", modifiers: [])
            Divider()
            Toggle(L("Inspektor", "Inspector"), isOn: $showInspector).keyboardShortcut("i", modifiers: [.command, .option])
            Toggle(L("Lernmodus", "Learning Mode"), isOn: $showLearning).keyboardShortcut("e", modifiers: [])
            Button(L("Ansicht zentrieren", "Center View")) { engine.action("center") }.keyboardShortcut("c", modifiers: [])
            Toggle(L("Statusanzeige im Bild", "Status Overlay"), isOn: Binding(get: { engine.state.showHUD != 0 }, set: { _ in engine.action("hud") }))
                .keyboardShortcut("h", modifiers: [])
        }
        CommandGroup(replacing: .help) {
            Button(L("Tastenübersicht", "Keys and Mouse")) { engine.action("help") }
                .keyboardShortcut(KeyEquivalent(Character(UnicodeScalar(NSF1FunctionKey)!)), modifiers: [])
        }
    }

    private func save(type: UTType, ext: String, _ done: @escaping (URL) -> Void) {
        let p = NSSavePanel()
        p.allowedContentTypes = [type]
        let stamp = ISO8601DateFormatter().string(from: Date()).prefix(16).replacingOccurrences(of: ":", with: ".")
        p.nameFieldStringValue = "\(engine.sceneName) \(stamp).\(ext)"
        if let w = NSApp.keyWindow {
            p.beginSheetModal(for: w) { if $0 == .OK, let u = p.url { done(u) } }
        } else if p.runModal() == .OK, let u = p.url {
            done(u)
        }
    }
}

func alert(_ title: String, _ text: String) {
    let a = NSAlert()
    a.alertStyle = .warning
    a.messageText = title
    a.informativeText = text
    a.runModal()
}

/// AppDelegate opens documents from the Finder and ends the app with its window.
final class AppDelegate: NSObject, NSApplicationDelegate {
    @MainActor static var engine: Engine?

    func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool { true }

    @MainActor func application(_ application: NSApplication, open urls: [URL]) {
        for u in urls {
            NSLog("Galaxy: open \(u.path)")
            if let err = AppDelegate.engine?.open(u) { alert(L("Datei kann nicht geöffnet werden", "Cannot open the file"), err) }
        }
    }
}

import CGalaxy
import Foundation
import Observation
import SwiftUI

/// uiGerman is the language of the user interface: the system's, unless -lang chose one. The Go
/// core decides (i18n), so both sides always agree.
nonisolated(unsafe) var uiGerman = (Locale.preferredLanguages.first ?? "en").hasPrefix("de")

/// Picks the German or English text, like i18n.T in the Go core.
func L(_ de: String, _ en: String) -> String { uiGerman ? de : en }

/// Engine wraps the Go core. All calls happen on the main thread.
@MainActor
@Observable
final class Engine {
    static let floatsPerBody = 12

    /// state is the slowly changing part of the core's state (settings, modes, counts), assigned
    /// only when it changes: menus, pickers and the menu bar read it, and SwiftUI rebuilds what reads
    /// a changed property, which closes an open menu. Values that change every frame are in live.
    private(set) var state = GalaxyState()
    /// live is the full state including time, rates and drift, for the small views that show them.
    private(set) var live = GalaxyState()
    private(set) var sceneName = ""
    private(set) var sceneInfo = ""
    private(set) var message = ""
    private(set) var topics: [String] = []
    private(set) var topicIndex = 0
    private(set) var explanationTitle = ""
    private(set) var explanationBody = ""
    private(set) var settings = GalaxySettings()
    private(set) var launch = GalaxyLaunch()
    /// shotInfo is the explanation the screenshot mode shows in the window (-info, T-061): a
    /// popover is a window of its own, which the screenshot mode does not capture.
    var shotInfo: SettingInfo?
    /// showIntro shows the opening crawl (T-060): at the first start, then only if chosen in the
    /// settings (introAtStart), from the Help menu, or with -intro.
    var showIntro = false
    let presets: [String]
    /// catalog lists the built-in scenes by group with a short description (T-078).
    let catalog: [SceneEntry]
    /// showScenes opens the scene overview (File menu, inspector, -scenes).
    var showScenes = false
    let hardware: String

    /// init starts the Go core with the command line (flags: engine/options.go, CLAUDE.md);
    /// startError is set if the arguments were invalid.
    init() {
        var args = CommandLine.arguments.dropFirst().joined(separator: "\u{1f}")
        if !args.contains("-lang") { // the Go core follows the system language like the app
            args += (args.isEmpty ? "" : "\u{1f}") + "-lang\u{1f}" + (Locale.preferredLanguages.first ?? "en")
        }
        if let err = args.withCString({ galaxy_start(UnsafeMutablePointer(mutating: $0)) }) {
            startError = String(cString: err)
            galaxy_free(err)
            _ = "".withCString { galaxy_start(UnsafeMutablePointer(mutating: $0)) }
        }
        uiGerman = Engine.text("lang") == "de"
        presets = Engine.text("presets").split(separator: "\n").map(String.init)
        catalog = (try? JSONDecoder().decode([SceneEntry].self, from: Data(Engine.text("sceneCatalog").utf8))) ?? []
        showScenes = Engine.text("scenes") == "1"
        hardware = Engine.text("hardware")
        var after = 0.0
        if let p = galaxy_shot(&after) {
            let path = String(cString: p)
            galaxy_free(p)
            if !path.isEmpty { shot = (URL(fileURLWithPath: path), after) }
        }
        refresh()
        if shot != nil { shotInfo = settingInfo(Engine.text("info")) }
        if Engine.text("intro") == "1" {
            showIntro = true
        } else if shot == nil {
            let d = UserDefaults.standard
            showIntro = d.object(forKey: "introAtStart") == nil || d.bool(forKey: "introAtStart")
            if d.object(forKey: "introAtStart") == nil { d.set(false, forKey: "introAtStart") } // once by default
        }
    }

    /// startError describes invalid command line arguments.
    private(set) var startError: String?
    /// shot is the screenshot to take (-shot, -shot-after), nil in the interactive app.
    private(set) var shot: (url: URL, after: Double)?

    func shotLine(_ url: URL, fps: Double) -> String {
        guard let p = url.path.withCString({ galaxy_shot_line(UnsafeMutablePointer(mutating: $0), fps) }) else { return "" }
        defer { galaxy_free(p) }
        return String(cString: p)
    }

    static func text(_ what: String) -> String {
        guard let p = what.withCString({ galaxy_text(UnsafeMutablePointer(mutating: $0)) }) else { return "" }
        defer { galaxy_free(p) }
        return String(cString: p)
    }

    /// refresh copies the published state for SwiftUI (a few times per second).
    func refresh() {
        var s = GalaxyState()
        galaxy_state(&s)
        live = s
        // state holds only values that change rarely: menus, the inspector and its popovers rebuild
        // whenever it changes, which closes open submenus and (i) popovers (T-077). Every value that
        // changes while the simulation runs must be zeroed here and read from live.
        var slow = s // without the values that change all the time
        slow.sps = 0; slow.fps = 0; slow.time = 0; slow.drift = 0; slow.driftQuality = 0
        slow.substeps = 0; slow.hydroSubsteps = 0; slow.starsFormed = 0; slow.bodies = 0
        slow.theta = 0; slow.renderScale = 0; slow.scaleFactor = 0
        slow.supernovae = 0; slow.sfr = 0; slow.snRate = 0; slow.gasShare = 0; slow.infallRate = 0
        slow.barA2 = 0; slow.barOmega = 0 // measured every 0.5 s (T-090); read from live
        slow.darkBodies = min(slow.darkBodies, 1); slow.gasBodies = min(slow.gasBodies, 1) // only whether there are any
        slow.blackHoles = min(slow.blackHoles, 1)
        if !same(slow, state) { state = slow }
        let name = Engine.text("sceneName")
        if name != sceneName {
            sceneName = name
            sceneInfo = Engine.text("sceneInfo")
        }
        let m = Engine.text("message")
        if m != message { message = m }
        let t = Engine.text("topics").split(separator: "\n", omittingEmptySubsequences: false).map(String.init)
        if t != topics { topics = t }
        let ti = Int(galaxy_topic_index())
        if ti != topicIndex { topicIndex = ti }
        var la = GalaxyLaunch()
        galaxy_launch(&la)
        if !same(la, launch) { launch = la }
        var st = GalaxySettings()
        galaxy_settings(&st)
        if st.stars != settings.stars || st.gpu != settings.gpu || st.capacity != settings.capacity { settings = st }
        let title = Engine.text("title"), body = Engine.text("body")
        if title != explanationTitle { explanationTitle = title }
        if body != explanationBody { explanationBody = body }
    }

    /// same compares two C structs byte by byte (NaN-safe; Observable notifies on every assignment).
    private func same<T>(_ a: T, _ b: T) -> Bool {
        withUnsafeBytes(of: a) { x in withUnsafeBytes(of: b) { y in x.elementsEqual(y) } }
    }

    func action(_ name: String) {
        _ = name.withCString { galaxy_action(UnsafeMutablePointer(mutating: $0)) }
        refresh()
    }

    func set(_ name: String, _ v: Double) {
        name.withCString { galaxy_set(UnsafeMutablePointer(mutating: $0), v) }
        refresh()
    }

    func key(_ ch: Character) -> Bool {
        guard let u = ch.unicodeScalars.first else { return false }
        let ok = galaxy_key(Int32(u.value)) != 0
        refresh()
        return ok
    }

    static let starChoices = [0, 10_000, 25_000, 50_000, 100_000, 200_000, 500_000, 1_000_000]

    func setLaunch(_ name: String, _ v: Double) {
        name.withCString { galaxy_launch_set(UnsafeMutablePointer(mutating: $0), v) }
        refresh()
    }

    func launchPreset(_ p: Int) { galaxy_launch_preset(Int32(p)); refresh() }
    func launchGo() { galaxy_launch_go(); refresh() }
    func launchCancel() { galaxy_launch_cancel(); refresh() }

    /// num writes three significant digits without exponent: 0.0123, 4.56, 4 480, 12 300.
    static func num(_ x: Double) -> String {
        abs(x) >= 1000 ? groupDigits(Int(x.rounded())) : String(format: "%.3g", x)
    }

    /// speedText shows a speed in km/s if the scene has units, else in simulation units.
    func speedText(_ v: Double) -> String {
        launch.hasUnits != 0 ? Engine.num(v * launch.kmPerS) + " km/s" : Engine.num(v)
    }

    /// massText shows a mass in solar masses if the scene has units, e.g. "1,0·10¹⁰ M☉".
    func massText(_ m: Double) -> String {
        guard launch.hasUnits != 0, launch.msunPerUnit > 0 else { return Engine.num(m) }
        let msun = m * launch.msunPerUnit
        let e = Int(floor(log10(msun)))
        let superscripts: [Character] = ["⁰", "¹", "²", "³", "⁴", "⁵", "⁶", "⁷", "⁸", "⁹"]
        let exp = String(String(abs(e)).map { superscripts[Int(String($0))!] })
        return String(format: "%.1f·10%@ M☉", msun / pow(10, Double(e)), (e < 0 ? "⁻" : "") + exp)
    }

    /// lengthText shows a length in the unit that suits the scale of the scene: AU, pc, kpc or Mpc.
    func lengthText(_ l: Double) -> String {
        guard launch.hasUnits != 0 else { return Engine.num(l) }
        let pc = l * launch.pcPerUnit
        let scale = max(abs(l), 1) * launch.pcPerUnit // small values keep the unit of the scene
        switch scale {
        case ..<0.05: return Engine.num(pc * 206_264.8) + " AU"
        case ..<1000: return Engine.num(pc) + " pc"
        case ..<1e6: return Engine.num(pc / 1000) + " kpc"
        default: return Engine.num(pc / 1e6) + " Mpc"
        }
    }

    func setStars(_ n: Int) { galaxy_set_stars(Int32(n)); refresh() }
    func setGPU(_ mode: Int) { galaxy_set_gpu(Int32(mode)); refresh() }
    func recalibrate() { galaxy_recalibrate(); refresh() }

    func setProfile(_ id: String) {
        id.withCString { galaxy_set_profile(UnsafeMutablePointer(mutating: $0)) }
        refresh()
    }

    /// open loads a scene (.json) or saved state; it returns an error message.
    func open(_ url: URL) -> String? {
        let err = url.path.withCString { galaxy_open(UnsafeMutablePointer(mutating: $0)) }
        refresh()
        guard let err else { return nil }
        defer { galaxy_free(err) }
        return String(cString: err)
    }

    func save(_ url: URL) { url.path.withCString { galaxy_save(UnsafeMutablePointer(mutating: $0)) } }

    // Effects as bits, like GalaxyState.fx.
    static let glow = 1, doppler = 2, lens = 4, trails = 8, dark = 16, gas = 32, dust = 64

    func fx(_ bit: Int) -> Bool { Int(state.fx) & bit != 0 }

    /// binding for an effect checkbox or menu toggle.
    func fxBinding(_ bit: Int) -> Binding<Bool> {
        Binding(get: { self.fx(bit) }, set: { on in
            var f = Int(self.state.fx)
            f = on ? f | bit : f & ~bit
            self.set("fx", Double(f))
        })
    }
}

/// SceneEntry is a built-in scene in the overview (scene.Entry in Go).
struct SceneEntry: Decodable, Identifiable {
    let index: Int
    let name: String
    let group: String
    let groupTitle: String
    let summary: String
    let key: String
    var id: Int { index }
}

extension Array where Element == SceneEntry {
    /// groups splits the catalog into its groups, in order.
    var groups: [(title: String, scenes: [SceneEntry])] {
        var out: [(title: String, scenes: [SceneEntry])] = []
        for e in self {
            if out.last?.title == e.groupTitle {
                out[out.count - 1].scenes.append(e)
            } else {
                out.append((e.groupTitle, [e]))
            }
        }
        return out
    }
}

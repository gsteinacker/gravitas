import CGalaxy
import Foundation
import Observation
import SwiftUI

/// Picks the German or English text, like i18n.T in the Go core.
func L(_ de: String, _ en: String) -> String {
    (Locale.preferredLanguages.first ?? "en").hasPrefix("de") ? de : en
}

/// Engine wraps the Go core. All calls happen on the main thread.
@MainActor
@Observable
final class Engine {
    static let floatsPerBody = 12

    private(set) var state = GalaxyState()
    private(set) var sceneName = ""
    private(set) var sceneInfo = ""
    private(set) var message = ""
    private(set) var topics: [String] = []
    private(set) var topicIndex = 0
    private(set) var explanationTitle = ""
    private(set) var explanationBody = ""
    let presets: [String]
    let hardware: String

    init(preset: Int, stars: Int) {
        let lang = Locale.preferredLanguages.first ?? "en"
        lang.withCString { galaxy_init(Int32(preset), Int32(stars), UnsafeMutablePointer(mutating: $0)) }
        presets = Engine.text("presets").split(separator: "\n").map(String.init)
        hardware = Engine.text("hardware")
        refresh()
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
        state = s
        let name = Engine.text("sceneName")
        if name != sceneName {
            sceneName = name
            sceneInfo = Engine.text("sceneInfo")
        }
        message = Engine.text("message")
        let t = Engine.text("topics").split(separator: "\n", omittingEmptySubsequences: false).map(String.init)
        if t != topics { topics = t }
        topicIndex = Int(galaxy_topic_index())
        let title = Engine.text("title"), body = Engine.text("body")
        if title != explanationTitle { explanationTitle = title }
        if body != explanationBody { explanationBody = body }
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

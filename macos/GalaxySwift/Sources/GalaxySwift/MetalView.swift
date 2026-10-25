import AppKit
import CGalaxy
@preconcurrency import MetalKit
import SwiftUI

/// SimulationView is the MTKView with mouse and keyboard handling (game keys as in the Go app).
final class SimulationView: MTKView {
    var engine: Engine!
    private var press: CGPoint?, last: CGPoint = .zero
    private var moved = false, launching = false, panning = false

    override var acceptsFirstResponder: Bool { true }
    override func acceptsFirstMouse(for event: NSEvent?) -> Bool { true }

    /// pixel converts a window point of the event into render pixels (origin top left).
    private func pixel(_ e: NSEvent) -> CGPoint {
        let p = convert(e.locationInWindow, from: nil)
        let s = drawableSize.width / max(bounds.width, 1)
        return CGPoint(x: p.x * s, y: (bounds.height - p.y) * s)
    }

    override func mouseDown(with e: NSEvent) {
        window?.makeFirstResponder(self)
        press = pixel(e)
        last = press!
        moved = false
        launching = e.modifierFlags.contains(.option)
        panning = e.modifierFlags.contains(.shift)
        if launching { galaxy_launch_begin(press!.x, press!.y) } // ⌥-drag: aim and launch at once
    }

    override func rightMouseDown(with e: NSEvent) {
        press = pixel(e)
        last = press!
        moved = false
        launching = false
        panning = true
    }

    override func mouseDragged(with e: NSEvent) { drag(e) }
    override func rightMouseDragged(with e: NSEvent) { drag(e) }

    private func drag(_ e: NSEvent) {
        guard let p0 = press else { return }
        let p = pixel(e)
        let threshold = 4 * drawableSize.width / max(bounds.width, 1)
        if abs(p.x - p0.x) > threshold || abs(p.y - p0.y) > threshold { moved = true }
        if launching {
            galaxy_launch_drag(p.x, p.y)
        }
        if moved && !launching {
            if panning {
                galaxy_pan(p.x - last.x, p.y - last.y)
            } else {
                galaxy_rotate(p.x - last.x, p.y - last.y)
            }
        }
        last = p
    }

    override func mouseUp(with e: NSEvent) {
        guard let p0 = press else { return }
        let p = pixel(e)
        if launching {
            galaxy_launch_go()
        } else if !moved {
            // A click plans a launch (shown in the view and the inspector, Return starts it);
            // a double click starts at once.
            galaxy_launch_begin(p.x, p.y)
            if e.clickCount >= 2 { galaxy_launch_go() }
        }
        _ = p0
        press = nil
        engine.refresh()
    }

    override func rightMouseUp(with e: NSEvent) { press = nil }

    override func scrollWheel(with e: NSEvent) {
        let dy = e.hasPreciseScrollingDeltas ? e.scrollingDeltaY / 10 : e.scrollingDeltaY
        galaxy_zoom(pow(0.9, dy))
    }

    override func magnify(with e: NSEvent) { galaxy_zoom(1 / (1 + e.magnification)) }

    /// keyDown runs the game keys by typed character, so they work on every keyboard layout.
    override func keyDown(with e: NSEvent) {
        if e.keyCode == 49 { // space
            engine.action("pause")
            return
        }
        if engine.launch.active != 0 {
            switch e.keyCode {
            case 36, 76: galaxy_launch_go(); engine.refresh(); return // return, enter
            case 53: galaxy_launch_cancel(); engine.refresh(); return // escape
            default: break
            }
        }
        if engine.state.explain != 0 {
            switch e.keyCode {
            case 123: engine.action("prevTopic"); return
            case 124: engine.action("nextTopic"); return
            case 125: engine.action("nextLevel"); return
            case 126: engine.action("prevLevel"); return
            case 53: engine.set("explain", 0); return
            default: break
            }
        }
        for ch in e.characters ?? "" where !engine.key(ch) {
            super.keyDown(with: e)
        }
    }
}

struct MetalView: NSViewRepresentable {
    let engine: Engine
    let renderer: Renderer

    func makeNSView(context: Context) -> SimulationView {
        let v = SimulationView(frame: .zero, device: renderer.device)
        v.engine = engine
        v.colorPixelFormat = .bgra8Unorm
        v.autoResizeDrawable = false // the renderer sets the size from the governor's render scale
        v.preferredFramesPerSecond = 120
        v.delegate = renderer
        DispatchQueue.main.async { v.window?.makeFirstResponder(v) }
        return v
    }

    func updateNSView(_ v: SimulationView, context: Context) {}
}

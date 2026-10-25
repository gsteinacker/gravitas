import AppKit
import CGalaxy
@preconcurrency import MetalKit
import SwiftUI

/// SimulationView is the MTKView with mouse and keyboard handling (game keys as in the Go app).
final class SimulationView: MTKView {
    var engine: Engine!
    private var press: CGPoint?, last: CGPoint = .zero
    private var moved = false, launching = false, panning = false
    private var arrow: CAShapeLayer?, label: CATextLayer?, pressPoint = CGPoint.zero

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
        pressPoint = convert(e.locationInWindow, from: nil)
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
            showArrow(to: convert(e.locationInWindow, from: nil), speed: galaxy_launch_speed(p0.x, p0.y, p.x, p.y))
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
            galaxy_launch(p0.x, p0.y, p.x, p.y)
        } else if !moved {
            galaxy_click(p.x, p.y)
        }
        press = nil
        hideArrow()
        engine.refresh()
    }

    /// showArrow previews the launch of an ⌥-drag: from the press point to the cursor, with the speed.
    private func showArrow(to q: CGPoint, speed: Double) {
        wantsLayer = true
        if arrow == nil {
            let a = CAShapeLayer()
            a.strokeColor = NSColor(red: 1, green: 0.59, blue: 0.2, alpha: 1).cgColor
            a.fillColor = nil
            a.lineWidth = 2
            let t = CATextLayer()
            t.fontSize = 13
            t.foregroundColor = a.strokeColor
            t.contentsScale = window?.backingScaleFactor ?? 2
            layer?.addSublayer(a)
            layer?.addSublayer(t)
            arrow = a
            label = t
        }
        let p0 = pressPoint
        let path = CGMutablePath()
        path.addEllipse(in: CGRect(x: p0.x - 7, y: p0.y - 7, width: 14, height: 14))
        path.move(to: p0)
        path.addLine(to: q)
        let dx = q.x - p0.x, dy = q.y - p0.y, l = hypot(dx, dy)
        if l > 5 {
            let ux = dx / l, uy = dy / l
            for side in [1.0, -1.0] {
                path.move(to: q)
                path.addLine(to: CGPoint(x: q.x - 10 * ux + 6 * side * uy, y: q.y - 10 * uy - 6 * side * ux))
            }
        }
        CATransaction.begin()
        CATransaction.setDisableActions(true)
        arrow?.path = path
        label?.string = String(format: "v = %.2f", speed)
        label?.frame = CGRect(x: q.x + 12, y: q.y - 24, width: 120, height: 18)
        CATransaction.commit()
    }

    private func hideArrow() {
        arrow?.removeFromSuperlayer()
        label?.removeFromSuperlayer()
        arrow = nil
        label = nil
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

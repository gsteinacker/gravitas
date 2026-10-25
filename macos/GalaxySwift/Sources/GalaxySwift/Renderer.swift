import AppKit
import Observation
import CGalaxy
@preconcurrency import Metal
@preconcurrency import MetalKit
import simd

/// Uniforms must match the struct in the shader source (five float4).
struct Uniforms {
    var view: SIMD4<Float>
    var target: SIMD4<Float>
    var cam: SIMD4<Float>
    var eye: SIMD4<Float>
    var misc: SIMD4<Float>
    var haze: SIMD4<Float>
}

/// Camera is the projection of view.Camera, for the camera position used by the Doppler shift.
struct Camera {
    var tx, ty, tz, yaw, pitch, distance, focal, w, h: Double

    init(_ i: GalaxyFrameInfo, w: Double, h: Double) {
        tx = i.camX; ty = i.camY; tz = i.camZ; yaw = i.yaw; pitch = i.pitch; distance = i.distance
        self.w = w; self.h = h
        focal = h / (2 * tan(max(i.fov, 0.1) / 2))
    }

    var position: SIMD3<Double> {
        let sy = sin(yaw), cy = cos(yaw), sp = sin(pitch), cp = cos(pitch)
        let fwd = SIMD3<Double>(cp * sy, cp * cy, -sp)
        return SIMD3(tx, ty, tz) - fwd * distance
    }
}

@MainActor
final class Renderer: NSObject, MTKViewDelegate {
    let device: MTLDevice
    let queue: MTLCommandQueue
    let engine: Engine
    private var sprite, dust, fog, trail, compose, lens, copy, blur, final, hole: MTLRenderPipelineState!
    private var stars, dustImage, fogA, fogB, trailA, trailB, scene, lensed, small, blurA, blurB, sky: MTLTexture?
    private var bodies: [MTLBuffer] = []
    private var attn: [MTLBuffer] = [] // dust transmission per body, one per frame in flight
    private var slot = 0
    private var capacity = 0
    private var current = -1, filled = 0
    private var info = GalaxyFrameInfo()
    private let inFlight = DispatchSemaphore(value: 3)
    private var look = GalaxyLook()
    private var holes: [SIMD4<Float>] = []
    private var lensHoles: [SIMD4<Float>] = []
    private var worldHoles = [SIMD4<Float>](repeating: .zero, count: 16)
    private var lastDraw = CFAbsoluteTimeGetCurrent(), fps = 60.0
    private var lastRefresh = 0.0
    private var size = CGSize.zero
    private var scale = 1.0
    private var gizmoLayers: [CAShapeLayer] = []
    private var gizmoBuf = [Float](repeating: 0, count: 5 * 4096)
    private var markBuf = [Float](repeating: 0, count: 3 * 256)
    /// pad publishes the screen anchors of the launch pad (labels, speedometer) to SwiftUI.
    let pad = PadAnchors()
    var exportURL: URL?
    var shot: (url: URL, after: Double, start: Double)?

    init?(engine: Engine) {
        guard let d = MTLCreateSystemDefaultDevice(), let q = d.makeCommandQueue() else { return nil }
        device = d
        queue = q
        self.engine = engine
        super.init()
        do {
            let lib = try d.makeLibrary(URL: Renderer.shaderLibrary())
            func pipe(_ v: String, _ f: String, _ format: MTLPixelFormat, blend: (MTLBlendFactor, MTLBlendFactor)? = nil) throws -> MTLRenderPipelineState {
                let p = MTLRenderPipelineDescriptor()
                p.vertexFunction = lib.makeFunction(name: v)
                p.fragmentFunction = lib.makeFunction(name: f)
                p.colorAttachments[0].pixelFormat = format
                if let (src, dst) = blend {
                    p.colorAttachments[0].isBlendingEnabled = true
                    p.colorAttachments[0].sourceRGBBlendFactor = src
                    p.colorAttachments[0].sourceAlphaBlendFactor = src
                    p.colorAttachments[0].destinationRGBBlendFactor = dst
                    p.colorAttachments[0].destinationAlphaBlendFactor = dst
                }
                return try d.makeRenderPipelineState(descriptor: p)
            }
            let hdr = MTLPixelFormat.rgba16Float
            sprite = try pipe("spriteVertex", "spriteFragment", hdr, blend: (.one, .one))
            dust = try pipe("dustVertex", "spriteFragment", hdr, blend: (.one, .oneMinusSourceAlpha))
            fog = try pipe("fogVertex", "spriteFragment", hdr, blend: (.one, .one))
            trail = try pipe("screenVertex", "trailFragment", hdr)
            compose = try pipe("screenVertex", "composeFragment", hdr)
            lens = try pipe("screenVertex", "lensFragment", hdr)
            copy = try pipe("screenVertex", "copyFragment", hdr)
            blur = try pipe("screenVertex", "blurFragment", hdr)
            final = try pipe("screenVertex", "finalFragment", .bgra8Unorm)
            hole = try pipe("holeVertex", "holeFragment", .bgra8Unorm, blend: (.one, .oneMinusSourceAlpha))
        } catch {
            NSLog("Galaxy: shaders: \(error)")
            return nil
        }
    }

    /// shaderLibrary is Galaxy.metallib in the app bundle, or next to the executable during
    /// development (make app-swift puts it there).
    static func shaderLibrary() -> URL {
        if let u = Bundle.main.url(forResource: "Galaxy", withExtension: "metallib") { return u }
        return URL(fileURLWithPath: CommandLine.arguments[0]).resolvingSymlinksInPath()
            .deletingLastPathComponent().appendingPathComponent("Galaxy.metallib")
    }

    func mtkView(_ view: MTKView, drawableSizeWillChange size: CGSize) {}

    private func texture(_ w: Int, _ h: Int, _ format: MTLPixelFormat = .rgba16Float) -> MTLTexture {
        let d = MTLTextureDescriptor.texture2DDescriptor(pixelFormat: format, width: max(1, w), height: max(1, h), mipmapped: false)
        d.usage = [.renderTarget, .shaderRead]
        d.storageMode = .private
        return device.makeTexture(descriptor: d)!
    }

    private func resize(_ s: CGSize) {
        guard s != size else { return }
        size = s
        let w = Int(s.width), h = Int(s.height)
        stars = texture(w, h); dustImage = texture(w, h); trailA = texture(w, h); trailB = texture(w, h)
        scene = texture(w, h); lensed = texture(w, h)
        // The bloom works at a quarter of the window resolution at every render scale.
        let div = max(1, Int((4 * scale).rounded()))
        small = texture(w / div, h / div); blurA = texture(w / div, h / div); blurB = texture(w / div, h / div)
        fogA = texture(w / div, h / div); fogB = texture(w / div, h / div)
        sky = makeSky(w, h)
    }

    /// makeSky is the static background with faint stars.
    private func makeSky(_ w: Int, _ h: Int) -> MTLTexture {
        var pix = [UInt8](repeating: 0, count: w * h * 4)
        for i in stride(from: 0, to: pix.count, by: 4) { pix[i] = 2; pix[i + 1] = 2; pix[i + 2] = 8; pix[i + 3] = 255 }
        var rng = SystemRandomNumberGenerator()
        let n = Int(Double(w * h) * 1500 / 1_000_000 / (scale * scale))
        for _ in 0..<n {
            let v = UInt8.random(in: 20..<90, using: &rng), tint = UInt8.random(in: 0..<25, using: &rng)
            let i = (Int.random(in: 0..<h, using: &rng) * w + Int.random(in: 0..<w, using: &rng)) * 4
            pix[i] = v - tint / 2; pix[i + 1] = v - tint / 3; pix[i + 2] = v
        }
        let d = MTLTextureDescriptor.texture2DDescriptor(pixelFormat: .rgba8Unorm, width: w, height: h, mipmapped: false)
        d.usage = .shaderRead
        let t = device.makeTexture(descriptor: d)!
        t.replace(region: MTLRegionMake2D(0, 0, w, h), mipmapLevel: 0, withBytes: pix, bytesPerRow: w * 4)
        return t
    }

    private func ensureBuffers(_ n: Int) {
        guard n > capacity else { return }
        capacity = max(n + n / 4, 10_000)
        bodies = (0..<3).map { _ in
            device.makeBuffer(length: capacity * Engine.floatsPerBody * 4, options: .storageModeShared)!
        }
        attn = (0..<3).map { _ in device.makeBuffer(length: capacity * 16, options: .storageModeShared)! }
        current = -1
    }

    func draw(in view: MTKView) {
        let now = CFAbsoluteTimeGetCurrent()
        let dt = now - lastDraw
        lastDraw = now
        if dt > 0 { fps += 0.1 * (1 / dt - fps) }

        let device = Double(view.window?.backingScaleFactor ?? 2)
        let thermal = Int32(ProcessInfo.processInfo.thermalState.rawValue)
        galaxy_tick(fps, device, thermal, ProcessInfo.processInfo.isLowPowerModeEnabled ? 1 : 0)
        if galaxy_fullscreen_requested() != 0 { view.window?.toggleFullScreen(nil) }
        scale = galaxy_render_scale()
        let want = CGSize(width: (view.bounds.width * scale).rounded(), height: (view.bounds.height * scale).rounded())
        if view.drawableSize != want { view.drawableSize = want }
        resize(want)
        galaxy_viewport(Double(want.width), Double(want.height))
        if now - lastRefresh > 0.25 {
            lastRefresh = now
            engine.refresh()
        }

        inFlight.wait()
        var st = GalaxyState()
        galaxy_state(&st)
        ensureBuffers(Int(max(st.bodies, 1000)))
        let next = (current + 1) % 3
        let ptr = bodies[next].contents().bindMemory(to: Float.self, capacity: capacity * Engine.floatsPerBody)
        if galaxy_frame(ptr, Int32(capacity), current < 0 ? 1 : 0, &info) != 0 {
            current = next
            filled = Int(info.bodies)
        }
        if current >= 0 { prepare(Int(st.fx)) }
        updateGizmo(view)
        guard current >= 0, let pass = view.currentRenderPassDescriptor, let drawable = view.currentDrawable,
              let cmd = queue.makeCommandBuffer() else {
            inFlight.signal()
            return
        }
        cmd.addCompletedHandler { [inFlight] _ in inFlight.signal() }
        render(cmd, pass: pass, fx: Int(st.fx))
        if let url = shotDue(now), let w = view.window {
            snapshotWindow(w, url.deletingPathExtension().appendingPathExtension("window.png"))
            if let sheet = w.attachedSheet {
                snapshotWindow(sheet, url.deletingPathExtension().appendingPathExtension("sheet.png"))
            }
        }
        if let url = exportURL ?? shotDue(now) {
            exportURL = nil
            export(cmd, url: url, fx: Int(st.fx))
        }
        cmd.present(drawable)
        cmd.commit()
    }

    private func uniforms(_ fx: Int) -> Uniforms {
        let cam = Camera(info, w: Double(size.width), h: Double(size.height))
        let eye = cam.position
        return Uniforms(
            view: SIMD4(Float(size.width), Float(size.height), Float(cam.focal), Float(scale)),
            target: SIMD4(Float(info.camX), Float(info.camY), Float(info.camZ), Float(info.yaw)),
            cam: SIMD4(Float(info.pitch), Float(info.distance), Float(info.meanStarMass), Float(info.c)),
            eye: SIMD4(Float(eye.x), Float(eye.y), Float(eye.z), Float(look.exposure)),
            misc: SIMD4(Float(fx), Float(info.youngAge), Float(look.emNorm), Float(look.dustKappa)),
            haze: SIMD4(look.darkAlpha, look.gasAlpha, Float(look.worldHoles), 0))
    }

    private func screenPass(_ cmd: MTLCommandBuffer, _ target: MTLTexture, _ p: MTLRenderPipelineState, load: MTLLoadAction = .dontCare,
                            _ setup: (MTLRenderCommandEncoder) -> Void) {
        let d = MTLRenderPassDescriptor()
        d.colorAttachments[0].texture = target
        d.colorAttachments[0].loadAction = load
        d.colorAttachments[0].clearColor = MTLClearColor(red: 0, green: 0, blue: 0, alpha: 0)
        d.colorAttachments[0].storeAction = .store
        let e = cmd.makeRenderCommandEncoder(descriptor: d)!
        e.setRenderPipelineState(p)
        setup(e)
        e.drawPrimitives(type: .triangleStrip, vertexStart: 0, vertexCount: 4)
        e.endEncoding()
    }

    private func render(_ cmd: MTLCommandBuffer, pass: MTLRenderPassDescriptor, fx: Int) {
        var u = uniforms(fx)
        // Dust first: the column density in front attenuates the background.
        let dd = MTLRenderPassDescriptor()
        dd.colorAttachments[0].texture = dustImage
        dd.colorAttachments[0].loadAction = .clear
        dd.colorAttachments[0].clearColor = MTLClearColor(red: 0, green: 0, blue: 0, alpha: 0)
        dd.colorAttachments[0].storeAction = .store
        let de = cmd.makeRenderCommandEncoder(descriptor: dd)!
        if fx & Engine.dust != 0, look.dustKappa > 0, filled > 0 {
            de.setRenderPipelineState(dust)
            de.setVertexBuffer(bodies[current], offset: 0, index: 0)
            de.setVertexBytes(&u, length: MemoryLayout<Uniforms>.stride, index: 1)
            de.drawPrimitives(type: .triangleStrip, vertexStart: 0, vertexCount: 4, instanceCount: filled)
        }
        de.endEncoding()
        // Stars and emission, additive, into the HDR image.
        let d = MTLRenderPassDescriptor()
        d.colorAttachments[0].texture = stars
        d.colorAttachments[0].loadAction = .clear
        d.colorAttachments[0].clearColor = MTLClearColor(red: 0, green: 0, blue: 0, alpha: 0)
        d.colorAttachments[0].storeAction = .store
        let e = cmd.makeRenderCommandEncoder(descriptor: d)!
        e.setRenderPipelineState(sprite)
        e.setVertexBuffer(bodies[current], offset: 0, index: 0)
        e.setVertexBytes(&u, length: MemoryLayout<Uniforms>.stride, index: 1)
        e.setVertexBuffer(attn[slot], offset: 0, index: 2)
        e.setVertexBytes(worldHoles, length: 16 * 16, index: 3)
        if filled > 0 {
            e.drawPrimitives(type: .triangleStrip, vertexStart: 0, vertexCount: 4, instanceCount: filled)
        }
        e.endEncoding()

        var starImage = stars!
        if fx & Engine.trails != 0 {
            screenPass(cmd, trailB!, trail) { e in
                e.setFragmentTexture(stars, index: 0)
                e.setFragmentTexture(trailA, index: 1)
            }
            swap(&trailA, &trailB)
            starImage = trailA!
        }
        var fogOn: Float = 0
        if fx & Engine.dark != 0, look.darkAlpha > 0, filled > 0 {
            // Dark matter as a soft fog: density sprites at quarter resolution, blurred.
            let fd = MTLRenderPassDescriptor()
            fd.colorAttachments[0].texture = fogA
            fd.colorAttachments[0].loadAction = .clear
            fd.colorAttachments[0].clearColor = MTLClearColor(red: 0, green: 0, blue: 0, alpha: 0)
            fd.colorAttachments[0].storeAction = .store
            let fe = cmd.makeRenderCommandEncoder(descriptor: fd)!
            fe.setRenderPipelineState(fog)
            fe.setVertexBuffer(bodies[current], offset: 0, index: 0)
            fe.setVertexBytes(&u, length: MemoryLayout<Uniforms>.stride, index: 1)
            fe.drawPrimitives(type: .triangleStrip, vertexStart: 0, vertexCount: 4, instanceCount: filled)
            fe.endEncoding()
            var h = SIMD2<Float>(1, 0), v = SIMD2<Float>(0, 1)
            screenPass(cmd, fogB!, blur) { e in
                e.setFragmentTexture(fogA, index: 0)
                e.setFragmentBytes(&h, length: 8, index: 0)
            }
            screenPass(cmd, fogA!, blur) { e in
                e.setFragmentTexture(fogB, index: 0)
                e.setFragmentBytes(&v, length: 8, index: 0)
            }
            var h2 = SIMD2<Float>(2.5, 0), v2 = SIMD2<Float>(0, 2.5) // a second, wider pass
            screenPass(cmd, fogB!, blur) { e in
                e.setFragmentTexture(fogA, index: 0)
                e.setFragmentBytes(&h2, length: 8, index: 0)
            }
            screenPass(cmd, fogA!, blur) { e in
                e.setFragmentTexture(fogB, index: 0)
                e.setFragmentBytes(&v2, length: 8, index: 0)
            }
            fogOn = 1
        }
        screenPass(cmd, scene!, compose) { e in
            e.setFragmentTexture(starImage, index: 0)
            e.setFragmentTexture(sky, index: 1)
            e.setFragmentTexture(dustImage, index: 2)
            e.setFragmentTexture(fogA, index: 3)
            e.setFragmentBytes(&fogOn, length: 4, index: 0)
        }
        var out = scene!
        if fx & Engine.lens != 0, !lensHoles.isEmpty {
            var l = [SIMD4<Float>](repeating: .zero, count: 17)
            for (i, h) in lensHoles.prefix(16).enumerated() { l[i] = h }
            l[16] = SIMD4(Float(min(16, lensHoles.count)), Float(size.width), Float(size.height), 0)
            screenPass(cmd, lensed!, lens) { e in
                e.setFragmentTexture(scene, index: 0)
                e.setFragmentBytes(&l, length: 17 * 16, index: 0)
            }
            out = lensed!
        }
        var strength: Float = 0
        if fx & Engine.glow != 0 {
            screenPass(cmd, small!, copy) { e in e.setFragmentTexture(out, index: 0) }
            var h = SIMD2<Float>(1, 0), v = SIMD2<Float>(0, 1)
            screenPass(cmd, blurA!, blur) { e in
                e.setFragmentTexture(small, index: 0)
                e.setFragmentBytes(&h, length: 8, index: 0)
            }
            screenPass(cmd, blurB!, blur) { e in
                e.setFragmentTexture(blurA, index: 0)
                e.setFragmentBytes(&v, length: 8, index: 0)
            }
            strength = 0.9
        }
        finalPass(cmd, target: pass.colorAttachments[0].texture!, scene: out, strength: strength, uniforms: u)
    }

    private func finalPass(_ cmd: MTLCommandBuffer, target: MTLTexture, scene out: MTLTexture, strength: Float, uniforms: Uniforms) {
        var s = strength, u = uniforms
        let d = MTLRenderPassDescriptor()
        d.colorAttachments[0].texture = target
        d.colorAttachments[0].loadAction = .dontCare
        d.colorAttachments[0].storeAction = .store
        let e = cmd.makeRenderCommandEncoder(descriptor: d)!
        e.setRenderPipelineState(final)
        e.setFragmentTexture(out, index: 0)
        e.setFragmentTexture(blurB, index: 1)
        e.setFragmentBytes(&s, length: 4, index: 0)
        e.drawPrimitives(type: .triangleStrip, vertexStart: 0, vertexCount: 4)
        if !holes.isEmpty {
            e.setRenderPipelineState(hole)
            e.setVertexBytes(holes, length: holes.count * 16, index: 0)
            e.setVertexBytes(&u, length: MemoryLayout<Uniforms>.stride, index: 1)
            e.drawPrimitives(type: .triangleStrip, vertexStart: 0, vertexCount: 4, instanceCount: holes.count)
        }
        e.endEncoding()
    }

    /// updateGizmo draws the launch pad (engine.Gizmo) as vector layers over the Metal image and
    /// passes the anchors for labels, speedometer and toolbar to the SwiftUI overlay.
    private func updateGizmo(_ view: MTKView) {
        let yellow = NSColor(red: 1, green: 0.84, blue: 0.25, alpha: 1)
        let cyan = NSColor(red: 0.3, green: 0.85, blue: 1, alpha: 1)
        let magenta = NSColor(red: 1, green: 0.4, blue: 0.9, alpha: 1)
        let orange = NSColor(red: 1, green: 0.59, blue: 0.2, alpha: 1)
        // Line styles in the order of the engine's Gizmo colors.
        let lines: [(NSColor, CGFloat, [NSNumber]?)] = [
            (NSColor(white: 1, alpha: 0.18), 1, [4, 4]),                            // plane rings
            (NSColor(white: 1, alpha: 0.4), 1, nil),                                // to the position
            (NSColor(white: 1, alpha: 0.8), 1, [3, 3]),                             // drop to the plane
            (orange, 3, nil),                                                       // velocity
            (NSColor(red: 1, green: 0.85, blue: 0.35, alpha: 0.85), 1.5, [6, 4]),   // preview path
            (NSColor(white: 1, alpha: 0.55), 1.2, nil),                             // compass
            (yellow, 2, nil), (cyan, 2, nil), (magenta, 2, nil),                    // axes
            (NSColor(red: 1, green: 0.35, blue: 0.25, alpha: 0.9), 2, [2, 3]),      // slingshot band
        ]
        // Marker styles in the order of the engine's Marker kinds: fill, stroke, radius (points).
        let ghostFill = engine.state.tool == 0 ? NSColor.black : NSColor(white: 1, alpha: 0.75)
        let markers: [(NSColor?, NSColor?, CGFloat)] = [
            (ghostFill.withAlphaComponent(0.85), orange, 9),       // ghost
            (yellow, nil, 6), (nil, yellow, 5),                    // prograde, retrograde
            (cyan, nil, 6), (nil, cyan, 5),                        // out, in
            (magenta, nil, 6), (nil, magenta, 5),                  // up, down
            (NSColor.white, nil, 5),                               // height
            (orange, NSColor.white, 6),                            // tip
        ]
        if gizmoLayers.isEmpty {
            view.wantsLayer = true
            for (color, width, dash) in lines {
                let l = CAShapeLayer()
                l.strokeColor = color.cgColor
                l.fillColor = nil
                l.lineWidth = width
                l.lineDashPattern = dash
                l.lineCap = .round
                view.layer?.addSublayer(l)
                gizmoLayers.append(l)
            }
            for (fill, stroke, _) in markers {
                let l = CAShapeLayer()
                l.fillColor = fill?.cgColor
                l.strokeColor = stroke?.cgColor
                l.lineWidth = 2
                view.layer?.addSublayer(l)
                gizmoLayers.append(l)
            }
        }
        var nMarks: Int32 = 0
        let n = gizmoBuf.withUnsafeMutableBufferPointer { segs in
            markBuf.withUnsafeMutableBufferPointer { marks in
                Int(galaxy_gizmo(segs.baseAddress!, 4096, marks.baseAddress!, 256, &nMarks))
            }
        }
        let paths = (0..<gizmoLayers.count).map { _ in CGMutablePath() }
        let k = 1 / CGFloat(scale), h = view.bounds.height
        for i in 0..<n {
            let c = min(lines.count - 1, max(0, Int(gizmoBuf[5 * i + 4])))
            paths[c].move(to: CGPoint(x: CGFloat(gizmoBuf[5 * i]) * k, y: h - CGFloat(gizmoBuf[5 * i + 1]) * k))
            paths[c].addLine(to: CGPoint(x: CGFloat(gizmoBuf[5 * i + 2]) * k, y: h - CGFloat(gizmoBuf[5 * i + 3]) * k))
        }
        var anchors: [Int: CGPoint] = [:]
        for i in 0..<Int(nMarks) {
            let kind = Int(markBuf[3 * i + 2])
            let x = CGFloat(markBuf[3 * i]) * k, y = CGFloat(markBuf[3 * i + 1]) * k
            if kind < markers.count {
                let r = markers[kind].2
                let rect = CGRect(x: x - r, y: h - y - r, width: 2 * r, height: 2 * r)
                if kind == 7 { paths[lines.count + kind].addRect(rect) } else { paths[lines.count + kind].addEllipse(in: rect) }
            } else {
                anchors[kind] = CGPoint(x: x, y: y) // SwiftUI coordinates: origin top left
            }
        }
        CATransaction.begin()
        CATransaction.setDisableActions(true)
        for (i, l) in gizmoLayers.enumerated() {
            l.frame = view.bounds
            l.path = paths[i]
        }
        CATransaction.commit()
        if anchors != pad.anchors { pad.anchors = anchors }
    }

    /// prepare takes the look of the frame from the Go core: exposure, emission and haze
    /// normalization and the black holes on screen (engine/look.go).
    private func prepare(_ fx: Int) {
        let p = bodies[current].contents().bindMemory(to: Float.self, capacity: capacity * Engine.floatsPerBody)
        slot = (slot + 1) % 3
        let a = attn[slot].contents().bindMemory(to: Float.self, capacity: capacity * 4)
        var raw = [GalaxyHole](repeating: GalaxyHole(), count: 64)
        worldHoles.withUnsafeMutableBytes { w in
            galaxy_prepare(p, a, Int32(capacity), Double(size.width), Double(size.height), scale, Int32(fx), &look, &raw, 64,
                           w.baseAddress!.assumingMemoryBound(to: Float.self), 16)
        }
        holes.removeAll(keepingCapacity: true)
        lensHoles.removeAll(keepingCapacity: true)
        for h in raw.prefix(Int(look.holes)) {
            holes.append(SIMD4(h.x, h.y, h.r, h.rings))
            let e = 5 * h.physR // Einstein radius on screen: 5 horizon radii
            lensHoles.append(SIMD4(h.x, h.y, e * e, 0))
        }
    }

    // ---- image export and screenshot mode ----

    private func shotDue(_ now: Double) -> URL? {
        guard let s = shot, now - s.start >= s.after else { return nil }
        return s.url
    }

    private func export(_ cmd: MTLCommandBuffer, url: URL, fx: Int) {
        let w = Int(size.width), h = Int(size.height)
        let d = MTLTextureDescriptor.texture2DDescriptor(pixelFormat: .bgra8Unorm, width: w, height: h, mipmapped: false)
        d.usage = [.renderTarget, .shaderRead]
        d.storageMode = .managed
        guard let t = device.makeTexture(descriptor: d) else { return }
        finalPass(cmd, target: t, scene: (fx & Engine.lens != 0 && !lensHoles.isEmpty) ? lensed! : scene!,
                  strength: fx & Engine.glow != 0 ? 0.9 : 0, uniforms: uniforms(fx))
        let blit = cmd.makeBlitCommandEncoder()!
        blit.synchronize(resource: t)
        blit.endEncoding()
        let isShot = shot?.url == url
        let line = engine.shotLine(url, fps: fps)
        cmd.addCompletedHandler { _ in
            var pix = [UInt8](repeating: 0, count: w * h * 4)
            t.getBytes(&pix, bytesPerRow: w * 4, from: MTLRegionMake2D(0, 0, w, h), mipmapLevel: 0)
            writePNG(pix, w, h, url)
            if isShot {
                print(line)
                exit(0)
            }
        }
    }

}

/// writePNG saves BGRA pixels.
func writePNG(_ bgra: [UInt8], _ w: Int, _ h: Int, _ url: URL) {
    var rgba = bgra
    for i in stride(from: 0, to: rgba.count, by: 4) {
        rgba.swapAt(i, i + 2)
        rgba[i + 3] = 255
    }
    guard let rep = NSBitmapImageRep(bitmapDataPlanes: nil, pixelsWide: w, pixelsHigh: h, bitsPerSample: 8,
                                     samplesPerPixel: 4, hasAlpha: true, isPlanar: false, colorSpaceName: .deviceRGB,
                                     bytesPerRow: w * 4, bitsPerPixel: 32) else { return }
    rgba.withUnsafeBytes { src in rep.bitmapData!.update(from: src.baseAddress!.assumingMemoryBound(to: UInt8.self), count: rgba.count) }
    try? rep.representation(using: .png, properties: [:])?.write(to: url)
}

/// snapshotWindow saves the window's controls (the Metal layer stays empty in this kind of
/// snapshot), for checks of the SwiftUI layout without screen access.
@MainActor func snapshotWindow(_ w: NSWindow, _ url: URL) {
    guard let v = w.contentView, let rep = v.bitmapImageRepForCachingDisplay(in: v.bounds) else { return }
    v.cacheDisplay(in: v.bounds, to: rep)
    try? rep.representation(using: .png, properties: [:])?.write(to: url)
}

/// PadAnchors are the positions (points, origin top left) of the launch pad's labels and
/// speedometer, by engine marker kind.
@MainActor
@Observable
final class PadAnchors {
    var anchors: [Int: CGPoint] = [:]

    static let labelPrograde = 9, labelOut = 10, labelCenter = 11, gauge = 12
}

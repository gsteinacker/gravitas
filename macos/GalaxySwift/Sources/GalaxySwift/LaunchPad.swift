import CGalaxy
import SwiftUI

/// LaunchPadOverlay shows the playful part of the launch pad next to the planned object (T-066):
/// compass labels, a speedometer with the circular and escape speed, and a small toolbar.
struct LaunchPadOverlay: View {
    var engine: Engine
    var pad: PadAnchors

    var body: some View {
        if engine.launch.active != 0, let g = pad.anchors[PadAnchors.gauge] {
            ZStack(alignment: .topLeading) {
                label(L("mit der Drehung", "prograde"), at: pad.anchors[PadAnchors.labelPrograde], color: .yellow)
                label(L("außen", "outward"), at: pad.anchors[PadAnchors.labelOut], color: .cyan)
                label(L("Zentrum", "center"), at: pad.anchors[PadAnchors.labelCenter], color: .cyan)
                VStack(alignment: .leading, spacing: 6) {
                    Speedometer(engine: engine)
                    PadToolbar(engine: engine)
                }
                .padding(8)
                .background(.ultraThinMaterial, in: RoundedRectangle(cornerRadius: 10))
                .fixedSize()
                .offset(x: g.x + 70, y: g.y + 40)
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        }
    }

    @ViewBuilder private func label(_ text: String, at p: CGPoint?, color: Color) -> some View {
        if let p {
            Text(text)
                .font(.caption2.weight(.semibold))
                .foregroundStyle(color)
                .shadow(color: .black, radius: 2)
                .fixedSize()
                .position(p)
                .allowsHitTesting(false)
        }
    }
}

/// Speedometer: an arc from 0 to beyond the escape speed with marks for the circular (green) and
/// escape (red) speed at the position, and the planned speed as needle.
struct Speedometer: View {
    var engine: Engine

    var body: some View {
        let l = engine.launch
        let top = max(1.3 * l.escape, 1.1 * l.speed, 1e-9)
        VStack(spacing: 2) {
            Canvas { ctx, size in
                let c = CGPoint(x: size.width / 2, y: size.height - 4)
                let r = min(size.width / 2, size.height) - 6
                func angle(_ v: Double) -> Angle { .degrees(180 + 180 * min(1, max(0, v / top))) }
                var arc = Path()
                arc.addArc(center: c, radius: r, startAngle: .degrees(180), endAngle: .degrees(360), clockwise: false)
                ctx.stroke(arc, with: .color(.white.opacity(0.35)), lineWidth: 5)
                func tick(_ v: Double, _ color: Color) {
                    let a = angle(v).radians
                    var p = Path()
                    p.move(to: CGPoint(x: c.x + (r - 8) * cos(a), y: c.y + (r - 8) * sin(a)))
                    p.addLine(to: CGPoint(x: c.x + (r + 4) * cos(a), y: c.y + (r + 4) * sin(a)))
                    ctx.stroke(p, with: .color(color), lineWidth: 3)
                }
                if l.circular > 0 { tick(l.circular, .green) }
                if l.escape > 0 { tick(l.escape, .red) }
                let a = angle(l.speed).radians
                var needle = Path()
                needle.move(to: c)
                needle.addLine(to: CGPoint(x: c.x + (r - 4) * cos(a), y: c.y + (r - 4) * sin(a)))
                ctx.stroke(needle, with: .color(.orange), style: StrokeStyle(lineWidth: 2.5, lineCap: .round))
                ctx.fill(Path(ellipseIn: CGRect(x: c.x - 4, y: c.y - 4, width: 8, height: 8)), with: .color(.orange))
            }
            .frame(width: 150, height: 62)
            Text(verbatim: engine.speedText(l.speed)).font(.headline.monospacedDigit())
            Text(verbatim: ratioText(l)).font(.caption.monospacedDigit()).foregroundStyle(.secondary)
            if l.binary != 0, l.holeKick > 0 {
                Text(verbatim: String(format: L("Das Loch erhält %@", "The hole gets %@"), engine.speedText(l.holeKick)))
                    .font(.caption.monospacedDigit()).foregroundStyle(l.holeKick > 0.3 * max(l.circular, 1e-9) ? .orange : .secondary)
            }
            if engine.state.tool == 2 { forecast }
        }
    }

    /// forecast tells whether the planned gas cloud will collapse (virial parameter, Go core).
    @ViewBuilder private var forecast: some View {
        let s = engine.state
        let (text, color): (String, Color) = s.cloudAlpha < 1
            ? (L("Wolke kollabiert, bildet Sterne", "cloud collapses, forms stars"), .green)
            : s.cloudAlpha < 2 ? (L("Wolke grenzwertig gebunden", "cloud marginally bound"), .yellow)
            : (L("Wolke zerstreut sich", "cloud disperses"), .orange)
        Text(verbatim: text + String(format: " (α = %.1f)", s.cloudAlpha)).font(.caption).foregroundStyle(color)
        if s.cloudFloorK > s.cloudK + 1 {
            Text(verbatim: String(format: L("Szene hält das Gas auf mind. %.0f K", "the scene keeps gas at ≥ %.0f K"), s.cloudFloorK))
                .font(.caption2).foregroundStyle(.secondary)
        }
    }

    private func ratioText(_ l: GalaxyLaunch) -> String {
        guard l.circular > 0 else { return "" }
        let r = l.speed / l.circular
        var s = String(format: L("%.2f × Kreisbahn", "%.2f × circular"), r)
        if l.speed >= l.escape, l.escape > 0 { s += L(" · entkommt", " · escapes") }
        if l.binary != 0 { s += L(" · Doppelsystem", " · binary") }
        return s
    }
}

/// PadToolbar: presets after Universe Sandbox's add modes, launch and cancel.
struct PadToolbar: View {
    var engine: Engine

    var body: some View {
        HStack(spacing: 4) {
            tool("pause.circle", L("Ruhe (relativ zur Galaxie)", "At rest (relative to the galaxy)")) { engine.launchPreset(0) }
            tool("circle.dashed", L("Kreisbahn", "Circular orbit")) { engine.launchPreset(1) }
            tool("arrow.triangle.2.circlepath", L("Doppelsystem mit dem Loch (Impuls erhalten)", "Binary with the hole (momentum conserved)")) {
                engine.launchPreset(4)
            }
            .disabled(engine.launch.holeAttractor == 0)
            tool("arrow.up.right.circle", L("Fluchtgeschwindigkeit nach außen", "Escape speed outward")) { engine.launchPreset(2) }
            tool("scope", L("Auf das Zentrum zu", "Towards the center")) { engine.launchPreset(3) }
            Divider().frame(height: 18)
            Button { engine.launchGo() } label: { Label(L("Los!", "Go!"), systemImage: "paperplane.fill") }
                .buttonStyle(.borderedProminent)
                .help(L("Starten (↩)", "Launch (↩)"))
            tool("xmark", L("Verwerfen (⎋)", "Discard (⎋)")) { engine.launchCancel() }
        }
        .controlSize(.small)
    }

    private func tool(_ symbol: String, _ help: String, _ action: @escaping () -> Void) -> some View {
        Button(action: action) { Image(systemName: symbol) }
            .buttonStyle(.bordered)
            .help(help)
    }
}

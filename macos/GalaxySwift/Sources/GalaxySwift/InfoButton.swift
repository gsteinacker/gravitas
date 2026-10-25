import CGalaxy
import Charts
import SwiftUI

/// SettingInfo is the (i) explanation of a control computed by the Go core (engine.SettingInfo, T-061).
struct SettingInfo: Decodable {
    struct Series: Decodable { let name: String; let color: String?; let points: [[Double]] }
    struct Bar: Decodable { let label: String; let value: Double; let current: Bool }
    struct Chart: Decodable {
        let kind: String
        let title: String
        let x: String
        let y: String
        let logX: Bool
        let logY: Bool
        let series: [Series]?
        let marker: [Double]?
        let bars: [Bar]?
    }

    let title: String
    let text: String
    let topic: String?
    let topicTitle: String?
    let chart: Chart?
}

extension Engine {
    /// settingInfo fetches the explanation of a control for the current audience and state.
    func settingInfo(_ id: String) -> SettingInfo? {
        guard let p = id.withCString({ galaxy_setting(UnsafeMutablePointer(mutating: $0)) }) else { return nil }
        defer { galaxy_free(p) }
        let json = String(cString: p)
        guard !json.isEmpty else { return nil }
        return try? JSONDecoder().decode(SettingInfo.self, from: Data(json.utf8))
    }

    /// showTopic opens the learning panel at a topic.
    func showTopic(_ id: String) {
        id.withCString { galaxy_show_topic(UnsafeMutablePointer(mutating: $0)) }
        refresh()
    }
}

/// InfoButton is the (i) next to a control: a popover with the explanation, a chart and a link
/// into the learning mode.
struct InfoButton: View {
    let engine: Engine
    let id: String
    @State private var info: SettingInfo?
    @State private var shown = false

    init(_ engine: Engine, _ id: String) {
        self.engine = engine
        self.id = id
    }

    var body: some View {
        Button {
            info = engine.settingInfo(id)
            shown = info != nil
        } label: {
            Image(systemName: "info.circle")
        }
        .buttonStyle(.borderless)
        .help(L("Erklärung", "Explanation"))
        .popover(isPresented: $shown, arrowEdge: .trailing) {
            if let info { InfoPopover(engine: engine, info: info, close: { shown = false }) }
        }
    }
}

struct InfoPopover: View {
    let engine: Engine
    let info: SettingInfo
    let close: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text(info.title).font(.headline)
            Text(info.text).textSelection(.enabled).fixedSize(horizontal: false, vertical: true)
                .frame(maxWidth: .infinity, alignment: .leading)
            if let chart = info.chart {
                Text(chart.title).font(.subheadline.bold())
                ChartView(chart: chart).frame(height: 170)
            }
            if let topic = info.topic, !topic.isEmpty, let title = info.topicTitle {
                Button {
                    engine.showTopic(topic)
                    close()
                } label: {
                    Label(L("Im Lernmodus: ", "In the learning mode: ") + title, systemImage: "book")
                }
                .buttonStyle(.link)
            }
        }
        .padding(14)
        .frame(width: 380)
    }
}

/// ChartView draws the chart of an explanation: lines on (log) axes with the current setting as a
/// dot, bars, or a sketch.
struct ChartView: View {
    let chart: SettingInfo.Chart

    private func color(_ name: String?) -> Color {
        switch name {
        case "red": return .red
        case "green": return .green
        case "blue": return .blue
        case "orange": return .orange
        default: return .accentColor
        }
    }

    var body: some View {
        switch chart.kind {
        case "line": lines
        case "bar": bars
        case "retarded": RetardedSketch()
        default: EmptyView()
        }
    }

    private var lines: some View {
        let series = chart.series ?? []
        return Chart {
            ForEach(series, id: \.name) { s in
                ForEach(Array(s.points.enumerated()), id: \.offset) { _, p in
                    LineMark(x: .value(chart.x, p[0]), y: .value(chart.y, p[1]), series: .value("", s.name))
                        .foregroundStyle(color(s.color))
                }
                .foregroundStyle(by: .value("", s.name))
            }
            if let m = chart.marker {
                PointMark(x: .value(chart.x, m[0]), y: .value(chart.y, m[1]))
                    .symbol { Circle().strokeBorder(Color.primary, lineWidth: 2).frame(width: 11, height: 11) }
            }
        }
        .chartForegroundStyleScale(domain: series.map(\.name), range: series.map { color($0.color) })
        .chartXScale(domain: xDomain, type: chart.logX ? .log : .linear)
        .chartYScale(type: chart.logY ? .log : .linear)
        .chartXAxis { axis }
        .chartYAxis { axis }
        .chartXAxisLabel(chart.x)
        .chartYAxisLabel(chart.y)
        .chartLegend(position: .bottom)
    }

    /// xDomain is the range of the data, so a linear axis does not start at 0.
    private var xDomain: ClosedRange<Double> {
        let xs = (chart.series ?? []).flatMap { $0.points.map { $0[0] } } + (chart.marker.map { [$0[0]] } ?? [])
        guard let lo = xs.min(), let hi = xs.max(), hi > lo else { return 0...1 }
        return lo...hi
    }

    /// axis labels numbers briefly (1e-06, 0.5, 100), also on log scales.
    private var axis: some AxisContent {
        AxisMarks { v in
            AxisGridLine()
            AxisValueLabel {
                if let d = v.as(Double.self) { Text(verbatim: String(format: "%g", d)) }
            }
        }
    }

    private var bars: some View {
        Chart(Array((chart.bars ?? []).enumerated()), id: \.offset) { _, b in
            BarMark(x: .value("", b.label), y: .value(chart.y, b.value))
                .foregroundStyle(b.current ? Color.accentColor : Color.secondary.opacity(0.5))
                .annotation(position: .top) { Text(verbatim: groupDigits(Int(b.value))).font(.caption2) }
        }
        .chartYAxisLabel(chart.y)
    }
}

/// RetardedSketch shows where a moving hole pulls a star from: its current position (instant), the
/// position when the gravity left it (retarded), and that position extrapolated with its velocity.
struct RetardedSketch: View {
    var body: some View {
        Canvas { ctx, size in
            let w = size.width, h = size.height
            let star = CGPoint(x: w * 0.5, y: h * 0.85)
            let past = CGPoint(x: w * 0.22, y: h * 0.2), now = CGPoint(x: w * 0.72, y: h * 0.2)
            let ahead = CGPoint(x: w * 0.69, y: h * 0.2)
            var path = Path()
            path.move(to: CGPoint(x: w * 0.08, y: h * 0.2))
            path.addLine(to: CGPoint(x: w * 0.92, y: h * 0.2))
            ctx.stroke(path, with: .color(.secondary), style: StrokeStyle(lineWidth: 1, dash: [4, 3]))
            func arrow(to p: CGPoint, _ color: Color, _ label: String, dy: CGFloat) {
                var a = Path()
                a.move(to: star)
                a.addLine(to: p)
                ctx.stroke(a, with: .color(color), lineWidth: 2)
                ctx.draw(Text(label).font(.caption).foregroundColor(color), at: CGPoint(x: p.x, y: p.y + dy))
            }
            arrow(to: past, .red, L("retardiert", "retarded"), dy: -12)
            arrow(to: now, .green, L("sofort", "instant"), dy: -12)
            arrow(to: ahead, .blue, "", dy: 0)
            ctx.draw(Text(L("ret. + v", "ret. + v")).font(.caption).foregroundColor(.blue),
                     at: CGPoint(x: (star.x + ahead.x) / 2 - 34, y: (star.y + ahead.y) / 2))
            for (p, c) in [(past, Color.gray), (now, Color.primary)] {
                ctx.fill(Path(ellipseIn: CGRect(x: p.x - 7, y: p.y - 7, width: 14, height: 14)), with: .color(c))
            }
            ctx.fill(Path(ellipseIn: CGRect(x: star.x - 4, y: star.y - 4, width: 8, height: 8)), with: .color(.yellow))
            ctx.draw(Text(L("Stern", "star")).font(.caption), at: CGPoint(x: star.x + 26, y: star.y))
            ctx.draw(Text(L("Loch: Bewegung →", "hole: motion →")).font(.caption).foregroundColor(.secondary),
                     at: CGPoint(x: w * 0.5, y: h * 0.06))
        }
    }
}

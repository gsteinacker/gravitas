import AppKit
import SwiftUI

/// IntroView is the opening crawl (T-060): a short line, then the title and a text in yellow that
/// runs tilted into the distance over a starry sky. A click or any key ends it; the simulation
/// already runs behind it.
struct IntroView: View {
    var engine: Engine
    @State private var phase = 0 // 0 lead-in, 1 crawl
    @State private var crawl: CGFloat = 0
    @State private var leadIn = 0.0
    @State private var fade = 1.0
    @State private var monitor: Any?

    /// crawlSeconds is how long the text needs to run out of sight.
    private let crawlSeconds = 70.0

    var body: some View {
        GeometryReader { geo in
            ZStack {
                Color.black.opacity(0.9)
                StarField()
                if phase == 0 {
                    Text(L("Vor nicht allzu langer Zeit, in genau dieser Galaxie …",
                           "Not so long ago, in this very galaxy …"))
                        .font(.system(size: 30, weight: .regular))
                        .foregroundStyle(Color(red: 0.3, green: 0.75, blue: 0.95))
                        .opacity(leadIn)
                        .padding(.horizontal, 80)
                } else {
                    Crawl(size: geo.size, offset: crawl)
                }
                VStack {
                    Spacer()
                    Text(L("Klick oder Taste: zur Simulation", "Click or any key: to the simulation"))
                        .font(.callout).foregroundStyle(.secondary)
                        .padding(.bottom, 18)
                }
            }
            .opacity(fade)
            .contentShape(Rectangle())
            .onTapGesture { finish() }
            .task { await play(height: geo.size.height) }
        }
        .ignoresSafeArea()
        .onAppear {
            monitor = NSEvent.addLocalMonitorForEvents(matching: .keyDown) { _ in
                finish()
                return nil // the key only ends the intro
            }
        }
        .onDisappear {
            if let monitor { NSEvent.removeMonitor(monitor) }
        }
    }

    /// paragraphs is the crawl. Each claim holds for the simulation (docs/physics.md).
    static var paragraphs: [String] {
        [
            L("Es ist eine Zeit großer Zusammenstöße. Ganze Galaxien stürzen ineinander, jede aus Milliarden Sonnen. Im Weltall dauert das Hunderte Millionen Jahre.",
              "It is a time of great collisions. Whole galaxies crash into each other, each made of billions of suns. In space this takes hundreds of millions of years."),
            L("Hier dauert es Sekunden. Zehntausend bis eine Million Teilchen ziehen einander an, jedes eine ganze Sterngruppe, berechnet nach Newton mit dem Baumverfahren von Barnes und Hut.",
              "Here it takes seconds. Ten thousand to a million particles attract each other, each a whole group of stars, computed with Newton's law and the tree method of Barnes and Hut."),
            L("In ihren Zentren lauern Schwarze Löcher. Mach das Licht langsamer, und ihre Horizonte wachsen, Sterne leuchten im Doppler-Effekt blau und rot, und selbst die Schwerkraft braucht Zeit, um anzukommen.",
              "Black holes lurk at their centers. Slow down light, and their horizons grow, stars glow blue and red in the Doppler effect, and even gravity needs time to arrive."),
            L("Gas verdichtet sich in den Spiralarmen und bildet neue, blaue Sterne. Staub verdunkelt sie. Unsichtbare Dunkle Materie hält alles zusammen.",
              "Gas piles up in the spiral arms and forms new, blue stars. Dust darkens them. Invisible dark matter holds it all together."),
            L("Echte Vorbilder warten: die Lokale Gruppe mit Milchstraße und Andromeda, die Antennen-Galaxien, die S-Sterne um das Loch im Zentrum der Milchstraße und der Pferdekopfnebel.",
              "Real examples are waiting: the Local Group with the Milky Way and Andromeda, the Antennae galaxies, the S-stars around the hole at the center of the Milky Way, and the Horsehead Nebula."),
            L("Klick hinein, setz ein Schwarzes Loch oder eine ganze Galaxie und sieh, was geschieht. Der Lernmodus (e) erklärt es, für Schüler, Interessierte und Physiker.",
              "Click into it, place a black hole or a whole galaxy and see what happens. The learning mode (e) explains it, for pupils, the curious and physicists."),
        ]
    }

    private func play(height: CGFloat) async {
        withAnimation(.easeIn(duration: 1.2)) { leadIn = 1 }
        try? await Task.sleep(for: .seconds(engine.shot != nil ? 0.5 : 4))
        withAnimation(.easeOut(duration: 1)) { leadIn = 0 }
        try? await Task.sleep(for: .seconds(engine.shot != nil ? 0.2 : 1.2))
        phase = 1
        if let shot = engine.shot {
            crawl = height * 1.0 // a still frame for the screenshot check (-intro)
            // The window snapshot drops 3D transforms and masks; render the crawl itself as well.
            let size = CGSize(width: 1200, height: 800)
            let still = ZStack { Color.black; StarField(); Crawl(size: size, offset: size.height * 1.0) }.frame(width: size.width, height: size.height)
            let r = ImageRenderer(content: still)
            if let img = r.nsImage, let tiff = img.tiffRepresentation, let rep = NSBitmapImageRep(data: tiff) {
                try? rep.representation(using: .png, properties: [:])?
                    .write(to: shot.url.deletingPathExtension().appendingPathExtension("intro.png"))
            }
            return
        }
        withAnimation(.linear(duration: crawlSeconds)) { crawl = height * 3.2 }
        try? await Task.sleep(for: .seconds(crawlSeconds))
        finish()
    }

    private func finish() {
        guard engine.showIntro else { return }
        withAnimation(.easeOut(duration: 0.6)) { fade = 0 }
        Task {
            try? await Task.sleep(for: .seconds(0.6))
            engine.showIntro = false
        }
    }
}

/// Crawl is the title and text, tilted back so that it runs into the distance as offset grows.
struct Crawl: View {
    let size: CGSize
    let offset: CGFloat

    var body: some View {
        text
            .offset(y: size.height * 0.6 - offset)
            .rotation3DEffect(.degrees(50), axis: (x: 1, y: 0, z: 0), anchor: .bottom, perspective: 1.4)
            .mask(LinearGradient(colors: [.clear, .black, .black], startPoint: .top, endPoint: UnitPoint(x: 0.5, y: 0.45)))
            .frame(width: size.width, height: size.height)
            .clipped()
    }

    private var text: some View {
        let yellow = Color(red: 1.0, green: 0.85, blue: 0.2)
        return VStack(spacing: 28) {
            Text("GRAVITAS").font(.system(size: 96, weight: .black)).kerning(8)
            Text(L("Galaxy Sandbox – eine Simulation der Schwerkraft", "Galaxy Sandbox – a simulation of gravity"))
                .font(.system(size: 34, weight: .bold)).multilineTextAlignment(.center)
            ForEach(Array(IntroView.paragraphs.enumerated()), id: \.offset) { _, p in
                Text(p).font(.system(size: 30, weight: .semibold)).multilineTextAlignment(.center).lineSpacing(6)
                    .fixedSize(horizontal: false, vertical: true)
            }
        }
        .foregroundStyle(yellow)
        .frame(width: min(760, size.width * 0.8))
        .fixedSize(horizontal: false, vertical: true)
        .frame(maxWidth: .infinity)
    }

}

/// StarField is a fixed random sky.
struct StarField: View {
    var body: some View {
        Canvas { ctx, size in
            var rng = SplitMix(seed: 7)
            for _ in 0..<Int(size.width * size.height / 2500) {
                let x = rng.next() * size.width, y = rng.next() * size.height
                let r = 0.4 + 1.2 * rng.next() * rng.next()
                ctx.fill(Path(ellipseIn: CGRect(x: x, y: y, width: r * 2, height: r * 2)),
                         with: .color(.white.opacity(0.3 + 0.7 * rng.next())))
            }
        }
    }
}

/// SplitMix is a small deterministic random generator, so the sky does not change between frames.
struct SplitMix {
    var state: UInt64
    init(seed: UInt64) { state = seed }
    mutating func next() -> Double {
        state &+= 0x9E37_79B9_7F4A_7C15
        var z = state
        z = (z ^ (z >> 30)) &* 0xBF58_476D_1CE4_E5B9
        z = (z ^ (z >> 27)) &* 0x94D0_49BB_1331_11EB
        return Double((z ^ (z >> 31)) >> 11) / Double(1 << 53)
    }
}

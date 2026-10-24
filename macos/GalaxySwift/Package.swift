// swift-tools-version:5.9
// SwiftUI variant of Galaxy (T-050). Build with `make app-swift` from the repository root: it
// builds the Go core as bin/lib/libgalaxy.a first.
import PackageDescription

let package = Package(
    name: "GalaxySwift",
    platforms: [.macOS(.v14)],
    targets: [
        .target(
            name: "CGalaxy",
            path: "Sources/CGalaxy",
            linkerSettings: [
                .unsafeFlags(["-L../../bin/lib"]),
                .linkedLibrary("galaxy"),
                .linkedLibrary("resolv"),
                .linkedFramework("Metal"),
                .linkedFramework("Foundation"),
                .linkedFramework("CoreFoundation"),
            ]
        ),
        .executableTarget(
            name: "GalaxySwift",
            dependencies: ["CGalaxy"],
            path: "Sources/GalaxySwift"
        ),
    ]
)

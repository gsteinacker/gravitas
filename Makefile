GO ?= go

.PHONY: check fmt vet test race bench run build shot plan next app dmg notarize icon dmg-swift notarize-swift package-dmg notarize-dmg lib lib-test metallib swift app-swift

check: fmt vet test
	$(GO) run ./tools/plan check

fmt:
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

vet:
	$(GO) vet ./...

test:
	$(GO) test ./...

race:
	$(GO) test -race ./...

bench:
	$(GO) test -run '^$$' -bench . -benchmem ./internal/...

build:
	$(GO) build -o bin/galaxy ./cmd/galaxy

run:
	$(GO) run ./cmd/galaxy

shot: swift
	./bin/swift/GalaxySwift -preset $(or $(PRESET),1) -shot bin/shot.png -shot-after $(or $(AFTER),6)
	@echo bin/shot.png

plan:
	@$(GO) run ./tools/plan board

next:
	@$(GO) run ./tools/plan next

# ---- macOS app (T-048) -------------------------------------------------------------------------
# make app      bin/Galaxy.app, universal (arm64 + x86_64), signed with SIGN_ID ("-" = ad hoc)
# make dmg      bin/Galaxy-<version>.dmg with a link to /Applications
# make notarize submit the DMG to Apple (needs SIGN_ID = "Developer ID Application: …" and a
#               notarytool keychain profile NOTARY_PROFILE)
# make icon     regenerate packaging/macos/Galaxy.icns from a screenshot

VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null | sed 's/^v//')
BUILD     ?= $(shell git rev-list --count HEAD 2>/dev/null || echo 1)
BUNDLE_ID ?= de.steinacker.galaxy
SIGN_ID   ?= -
NOTARY_PROFILE ?= galaxy-notary
APP       := bin/Galaxy.app
LDFLAGS   := -s -w -X galaxy/internal/mac.Version=$(VERSION)

app:
	rm -rf $(APP) bin/app-arm64 bin/app-amd64
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 CC="clang -arch arm64" CGO_CFLAGS="-mmacosx-version-min=12.0" CGO_LDFLAGS="-mmacosx-version-min=12.0" \
		$(GO) build -trimpath -ldflags '$(LDFLAGS)' -o bin/app-arm64 ./cmd/galaxy
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 CC="clang -arch x86_64" CGO_CFLAGS="-mmacosx-version-min=12.0" CGO_LDFLAGS="-mmacosx-version-min=12.0" \
		$(GO) build -trimpath -ldflags '$(LDFLAGS)' -o bin/app-amd64 ./cmd/galaxy
	mkdir -p $(APP)/Contents/MacOS $(APP)/Contents/Resources/de.lproj $(APP)/Contents/Resources/en.lproj
	lipo -create -output $(APP)/Contents/MacOS/Galaxy bin/app-arm64 bin/app-amd64
	sed -e 's/@VERSION@/$(VERSION)/g' -e 's/@BUILD@/$(BUILD)/g' -e 's/@BUNDLE_ID@/$(BUNDLE_ID)/g' \
		packaging/macos/Info.plist.tmpl > $(APP)/Contents/Info.plist
	plutil -lint $(APP)/Contents/Info.plist
	cp packaging/macos/Galaxy.icns $(APP)/Contents/Resources/
	touch $(APP)/Contents/Resources/de.lproj/InfoPlist.strings $(APP)/Contents/Resources/en.lproj/InfoPlist.strings
	codesign --force --sign "$(SIGN_ID)" $(if $(filter -,$(SIGN_ID)),--timestamp=none,--options runtime --timestamp) $(APP)
	codesign --verify --strict $(APP)
	@echo "$(APP) $(VERSION) ($$(lipo -archs $(APP)/Contents/MacOS/Galaxy))"

# DMG_APP and DMG_NAME choose the app: make dmg (Ebitengine) or make dmg-swift (SwiftUI).
DMG_APP  ?= $(APP)
DMG_NAME ?= Galaxy

dmg: app
	$(MAKE) package-dmg DMG_APP="$(APP)" DMG_NAME=Galaxy

dmg-swift: app-swift
	$(MAKE) package-dmg DMG_APP="$(SWIFT_APP)" DMG_NAME=Galaxy-SwiftUI

package-dmg:
	rm -rf bin/dmg "bin/$(DMG_NAME)-$(VERSION).dmg"
	mkdir -p bin/dmg
	cp -R "$(DMG_APP)" bin/dmg/
	ln -s /Applications bin/dmg/Applications
	hdiutil create -volname "$(DMG_NAME) $(VERSION)" -srcfolder bin/dmg -fs HFS+ -format UDZO -ov "bin/$(DMG_NAME)-$(VERSION).dmg"
	$(if $(filter -,$(SIGN_ID)),,codesign --sign "$(SIGN_ID)" --timestamp "bin/$(DMG_NAME)-$(VERSION).dmg")
	rm -rf bin/dmg
	@echo "bin/$(DMG_NAME)-$(VERSION).dmg"

notarize: dmg
	$(MAKE) notarize-dmg DMG_NAME=Galaxy

notarize-swift: dmg-swift
	$(MAKE) notarize-dmg DMG_NAME=Galaxy-SwiftUI

notarize-dmg:
	@if [ "$(SIGN_ID)" = "-" ]; then echo "notarize needs SIGN_ID=\"Developer ID Application: …\""; exit 1; fi
	xcrun notarytool submit "bin/$(DMG_NAME)-$(VERSION).dmg" --keychain-profile "$(NOTARY_PROFILE)" --wait
	xcrun stapler staple "bin/$(DMG_NAME)-$(VERSION).dmg"

icon: build
	./bin/galaxy -preset 1 -n 120000 -hud=false -scale 2 -shot bin/icon-src.png -shot-after 8
	rm -rf bin/Galaxy.iconset
	$(GO) run ./tools/icon -in bin/icon-src.png -out bin/Galaxy.iconset -size 0.82
	iconutil -c icns bin/Galaxy.iconset -o packaging/macos/Galaxy.icns

# ---- SwiftUI app (T-049 ff.) -------------------------------------------------------------------
# lib        bin/lib/libgalaxy.a, universal (arm64 + x86_64), and its header
# app-swift  "bin/Galaxy SwiftUI.app", universal, shaders precompiled (needs Xcode)
LIBFLAGS := CGO_ENABLED=1 GOOS=darwin CGO_CFLAGS="-mmacosx-version-min=14.0" CGO_LDFLAGS="-mmacosx-version-min=14.0"
SWIFT_PKG := macos/GalaxySwift
SWIFT_BIN := bin/swift/GalaxySwift

lib:
	mkdir -p bin/lib
	$(LIBFLAGS) GOARCH=arm64 CC="clang -arch arm64" $(GO) build -buildmode=c-archive -trimpath -o bin/lib/arm64/libgalaxy.a ./cmd/libgalaxy
	$(LIBFLAGS) GOARCH=amd64 CC="clang -arch x86_64" $(GO) build -buildmode=c-archive -trimpath -o bin/lib/amd64/libgalaxy.a ./cmd/libgalaxy
	lipo -create -output bin/lib/libgalaxy.a bin/lib/arm64/libgalaxy.a bin/lib/amd64/libgalaxy.a
	cp bin/lib/arm64/libgalaxy.h bin/lib/libgalaxy.h

lib-test: lib
	cc -o bin/lib/smoke cmd/libgalaxy/testdata/smoke.c -Ibin/lib bin/lib/libgalaxy.a \
		-framework Metal -framework Foundation -framework CoreFoundation -lresolv
	HOME=$$(mktemp -d) ./bin/lib/smoke

metallib:
	mkdir -p bin/metal
	xcrun -sdk macosx metal -std=metal3.0 -mmacosx-version-min=14.0 -c $(SWIFT_PKG)/Shaders/Galaxy.metal -o bin/metal/Galaxy.air
	xcrun -sdk macosx metallib bin/metal/Galaxy.air -o bin/metal/Galaxy.metallib

swift: lib metallib
	cd $(SWIFT_PKG) && swift build -c release --arch arm64 --arch x86_64
	mkdir -p bin/swift
	cp "$$(cd $(SWIFT_PKG) && swift build -c release --arch arm64 --arch x86_64 --show-bin-path)/GalaxySwift" $(SWIFT_BIN)
	cp bin/metal/Galaxy.metallib bin/swift/

SWIFT_APP := bin/Galaxy SwiftUI.app

app-swift: swift
	rm -rf "$(SWIFT_APP)"
	mkdir -p "$(SWIFT_APP)/Contents/MacOS" "$(SWIFT_APP)/Contents/Resources/de.lproj" "$(SWIFT_APP)/Contents/Resources/en.lproj"
	cp $(SWIFT_BIN) "$(SWIFT_APP)/Contents/MacOS/Galaxy"
	cp bin/metal/Galaxy.metallib "$(SWIFT_APP)/Contents/Resources/"
	sed -e 's/@VERSION@/$(VERSION)/g' -e 's/@BUILD@/$(BUILD)/g' -e 's/@BUNDLE_ID@/$(BUNDLE_ID)/g' \
		packaging/macos/Info.plist.tmpl > "$(SWIFT_APP)/Contents/Info.plist"
	/usr/libexec/PlistBuddy -c "Set :CFBundleIdentifier $(BUNDLE_ID).swiftui" -c "Set :CFBundleName 'Galaxy SwiftUI'" \
		-c "Set :CFBundleDisplayName 'Galaxy SwiftUI'" -c "Set :LSMinimumSystemVersion 14.0" \
		-c "Set :CFBundleDocumentTypes:0:LSHandlerRank Alternate" "$(SWIFT_APP)/Contents/Info.plist"
	plutil -lint "$(SWIFT_APP)/Contents/Info.plist"
	cp packaging/macos/Galaxy.icns "$(SWIFT_APP)/Contents/Resources/"
	touch "$(SWIFT_APP)/Contents/Resources/de.lproj/InfoPlist.strings" "$(SWIFT_APP)/Contents/Resources/en.lproj/InfoPlist.strings"
	codesign --force --sign "$(SIGN_ID)" $(if $(filter -,$(SIGN_ID)),--timestamp=none,--options runtime --timestamp) "$(SWIFT_APP)"
	@echo "$(SWIFT_APP) $(VERSION) ($$(lipo -archs "$(SWIFT_APP)/Contents/MacOS/Galaxy"))"

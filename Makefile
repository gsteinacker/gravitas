GO ?= go

.PHONY: check fmt vet test race bench lib lib-test metallib build run shot app dmg notarize package-dmg icon plan next

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

plan:
	@$(GO) run ./tools/plan board

next:
	@$(GO) run ./tools/plan next

# ---- the app: Go core as a library, SwiftUI front end (ADR-0011) -------------------------------
# lib       bin/lib/libgalaxy.a + .h, universal (arm64 + x86_64)
# build     bin/swift/GalaxySwift (with Galaxy.metallib next to it), for development and screenshots
# run       build and start it (flags as in CLAUDE.md, e.g. make run ARGS="-preset 2")
# shot      screenshot bin/shot.png (PRESET, AFTER)
# app       bin/Galaxy.app, universal, signed with SIGN_ID ("-" = ad hoc)
# dmg       bin/Galaxy-<version>.dmg with a link to /Applications
# notarize  submit the DMG to Apple (SIGN_ID = "Developer ID Application: …", NOTARY_PROFILE)
# icon      regenerate packaging/macos/Galaxy.icns from a screenshot

VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null | sed 's/^v//')
BUILD     ?= $(shell git rev-list --count HEAD 2>/dev/null || echo 1)
BUNDLE_ID ?= de.steinacker.galaxy
SIGN_ID   ?= -
NOTARY_PROFILE ?= galaxy-notary
APP       := bin/Galaxy.app
SWIFT_PKG := macos/GalaxySwift
BIN       := bin/swift/GalaxySwift
LIBFLAGS  := CGO_ENABLED=1 GOOS=darwin CGO_CFLAGS="-mmacosx-version-min=14.0" CGO_LDFLAGS="-mmacosx-version-min=14.0"

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

build: lib metallib
	cd $(SWIFT_PKG) && swift build -c release --arch arm64 --arch x86_64
	mkdir -p bin/swift
	cp "$$(cd $(SWIFT_PKG) && swift build -c release --arch arm64 --arch x86_64 --show-bin-path)/GalaxySwift" $(BIN)
	cp bin/metal/Galaxy.metallib bin/swift/

run: build
	./$(BIN) $(ARGS)

shot: build
	./$(BIN) -preset $(or $(PRESET),1) -shot bin/shot.png -shot-after $(or $(AFTER),6)
	@echo bin/shot.png

app: build
	rm -rf "$(APP)"
	mkdir -p "$(APP)/Contents/MacOS" "$(APP)/Contents/Resources/de.lproj" "$(APP)/Contents/Resources/en.lproj"
	cp $(BIN) "$(APP)/Contents/MacOS/Galaxy"
	cp bin/metal/Galaxy.metallib packaging/macos/Galaxy.icns "$(APP)/Contents/Resources/"
	sed -e 's/@VERSION@/$(VERSION)/g' -e 's/@BUILD@/$(BUILD)/g' -e 's/@BUNDLE_ID@/$(BUNDLE_ID)/g' \
		packaging/macos/Info.plist.tmpl > "$(APP)/Contents/Info.plist"
	plutil -lint "$(APP)/Contents/Info.plist"
	touch "$(APP)/Contents/Resources/de.lproj/InfoPlist.strings" "$(APP)/Contents/Resources/en.lproj/InfoPlist.strings"
	codesign --force --sign "$(SIGN_ID)" $(if $(filter -,$(SIGN_ID)),--timestamp=none,--options runtime --timestamp) "$(APP)"
	codesign --verify --strict "$(APP)"
	@echo "$(APP) $(VERSION) ($$(lipo -archs "$(APP)/Contents/MacOS/Galaxy"))"

dmg: app
	rm -rf bin/dmg "bin/Galaxy-$(VERSION).dmg"
	mkdir -p bin/dmg
	cp -R "$(APP)" bin/dmg/
	ln -s /Applications bin/dmg/Applications
	hdiutil create -volname "Galaxy $(VERSION)" -srcfolder bin/dmg -fs HFS+ -format UDZO -ov "bin/Galaxy-$(VERSION).dmg"
	$(if $(filter -,$(SIGN_ID)),,codesign --sign "$(SIGN_ID)" --timestamp "bin/Galaxy-$(VERSION).dmg")
	rm -rf bin/dmg
	@echo "bin/Galaxy-$(VERSION).dmg"

notarize: dmg
	@if [ "$(SIGN_ID)" = "-" ]; then echo "notarize needs SIGN_ID=\"Developer ID Application: …\""; exit 1; fi
	xcrun notarytool submit "bin/Galaxy-$(VERSION).dmg" --keychain-profile "$(NOTARY_PROFILE)" --wait
	xcrun stapler staple "bin/Galaxy-$(VERSION).dmg"

icon: build
	./$(BIN) -preset 1 -n 120000 -scale 2 -shot bin/icon-src.png -shot-after 8
	rm -rf bin/Galaxy.iconset
	$(GO) run ./tools/icon -in bin/icon-src.png -out bin/Galaxy.iconset -size 0.82
	iconutil -c icns bin/Galaxy.iconset -o packaging/macos/Galaxy.icns

GO ?= go

.PHONY: check fmt vet test race bench run build shot plan next

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

shot: build
	./bin/galaxy -preset $(or $(PRESET),1) -shot bin/shot.png -shot-after $(or $(AFTER),6)
	@echo bin/shot.png

plan:
	@$(GO) run ./tools/plan board

next:
	@$(GO) run ./tools/plan next

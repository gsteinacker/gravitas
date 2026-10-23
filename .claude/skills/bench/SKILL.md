---
name: bench
description: Benchmark simulation performance before/after a change and log results in docs/perf-log.md. Use for any change to solvers, integrators, trees or render batching.
---

# Benchmarken

1. Baseline vor der Änderung: `go test -run '^$' -bench <Pattern> -benchmem -count 6 ./internal/... > /tmp/old.txt`
   (bzw. im Scratchpad).
2. Änderung umsetzen und dasselbe nach `new.txt` schreiben.
3. Vergleich: `go run golang.org/x/perf/cmd/benchstat@latest old.txt new.txt`.
4. Bei Regressionen > 5 % oder Allokationen im Hot Loop profilen:
   `go test -run '^$' -bench <B> -cpuprofile cpu.prof ./internal/sim/... && go tool pprof -top cpu.prof`.
5. Ergebnis gegen das Budget in `docs/architecture.md` prüfen und in `docs/perf-log.md` eintragen (neueste oben).

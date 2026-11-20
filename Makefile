.PHONY: run check assets bench

# The race detector needs cgo. Where there is no C compiler, run `make check RACE=`.
# With it the game's tests take minutes, and more on a CI runner: so a package gets 20
# minutes, not go test's 10, before it is taken for hung.
RACE ?= -race

run:
	go run . $(ARGS)

check:
	@test -z "$$(gofmt -l .)" || (echo "gofmt needed:"; gofmt -l .; exit 1)
	go vet ./...
	go test -timeout 20m $(RACE) ./...

# Makes the pictures and the sounds again. The output is the same bytes every run.
assets:
	go run ./cmd/genassets

bench:
	go test -run '^$$' -bench . -benchmem ./internal/render ./internal/game

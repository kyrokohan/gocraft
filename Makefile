.PHONY: run check assets bench

# The race detector needs cgo. Where there is no C compiler, run `make check RACE=`.
RACE ?= -race

run:
	go run . $(ARGS)

check:
	@test -z "$$(gofmt -l .)" || (echo "gofmt needed:"; gofmt -l .; exit 1)
	go vet ./...
	go test $(RACE) ./...

# Draws the PNG assets again. The output is the same bytes every run.
assets:
	go run ./cmd/genassets

bench:
	go test -run '^$$' -bench . -benchmem ./internal/render ./internal/game

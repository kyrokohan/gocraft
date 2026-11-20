.PHONY: run check assets bench

# The race detector needs cgo. Where there is no C compiler, run `make check RACE=`.
# With it the game's tests take minutes, and more on a CI runner: so a package gets 20
# minutes, not go test's 10, before it is taken for hung.
RACE ?= -race

# RUN and SKIP narrow the tests to those whose names match, or do not: CI shares the
# slow run of the race detector out between two machines with them. Empty, all run.
RUN ?=
SKIP ?=

run:
	go run . $(ARGS)

check:
	@test -z "$$(gofmt -l .)" || (echo "gofmt needed:"; gofmt -l .; exit 1)
	go vet ./...
	go test -timeout 20m $(RACE) -run '$(RUN)' -skip '$(SKIP)' ./...

# Makes the pictures and the sounds again. The output is the same bytes every run.
assets:
	go run ./cmd/genassets

bench:
	go test -run '^$$' -bench . -benchmem ./internal/render ./internal/game

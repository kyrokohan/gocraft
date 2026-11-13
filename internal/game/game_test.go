package game

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"glonk.dev/engine"
)

func newGame(t *testing.T) *Game {
	t.Helper()
	g, err := New(1)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

// quitAfter runs a game for a number of frames and then quits, so that a whole
// engine.Run can be tested without a terminal or a clock.
type quitAfter struct {
	engine.Game
	frames int
}

func (q *quitAfter) Update(dt float64, in *engine.Input) error {
	if q.frames == 0 {
		return engine.ErrQuit
	}
	q.frames--
	return q.Game.Update(dt, in)
}

// runHeadless runs g with the given key bytes and returns what was written.
func runHeadless(t *testing.T, g engine.Game, keys string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var out bytes.Buffer
	err := engine.Run(ctx, g, engine.Config{
		Width: 60, Height: 20, FPS: 200,
		Input:  strings.NewReader(keys),
		Output: &out,
	})
	if err != nil {
		t.Fatalf("wanted a clean end, got %v", err)
	}
	return out.String()
}

func TestHeadlessRunDrawsFrames(t *testing.T) {
	out := runHeadless(t, &quitAfter{Game: newGame(t), frames: 3}, "")

	if out == "" {
		t.Error("no frame was written")
	}
}

func TestHeadlessRunEndsOnCtrlC(t *testing.T) {
	runHeadless(t, newGame(t), "\x03")
}

package game

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"glonk.dev/engine"
)

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

// runHeadless runs g for a few frames with the given key bytes and returns what was
// written.
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
	out := runHeadless(t, &quitAfter{Game: New(), frames: 3}, "")

	if !strings.Contains(out, "gocraft") {
		t.Error("the frames written do not show the game")
	}
}

func TestHeadlessRunEndsOnCtrlC(t *testing.T) {
	runHeadless(t, New(), "\x03")
}

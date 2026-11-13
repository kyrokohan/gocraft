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

// onFloor is a game whose player stands on the flat test floor, facing +z.
func onFloor(t *testing.T) *Game {
	t.Helper()
	g := newGame(t)
	g.world, g.player = floor(), *standing()
	return g
}

// hold runs the game for a number of frames with keys held down.
func hold(t *testing.T, g *Game, frames int, keys ...engine.Key) {
	t.Helper()
	kb := engine.NewKeyboard()
	kb.Press(keys...)
	for range frames {
		if err := g.Update(frame, kb.Input()); err != nil {
			t.Fatalf("Update: %v", err)
		}
		kb.EndFrame()
	}
}

func TestKeysMoveAndTurnThePlayer(t *testing.T) {
	for _, tc := range []struct {
		name  string
		key   engine.Key
		moved func(p Player) bool
	}{
		{"w walks ahead", 'w', func(p Player) bool { return p.Pos.Z > 21 }},
		{"a capital works too", 'W', func(p Player) bool { return p.Pos.Z > 21 }},
		{"s walks back", 's', func(p Player) bool { return p.Pos.Z < 20 }},
		{"a goes left", 'a', func(p Player) bool { return p.Pos.X < 20 }},
		{"d goes right", 'd', func(p Player) bool { return p.Pos.X > 21 }},
		{"right arrow turns right", engine.KeyRight, func(p Player) bool { return p.Yaw > 0.1 }},
		{"left arrow turns left", engine.KeyLeft, func(p Player) bool { return p.Yaw < -0.1 }},
		{"up arrow looks up", engine.KeyUp, func(p Player) bool { return p.Pitch > 0.1 }},
		{"down arrow looks down", engine.KeyDown, func(p Player) bool { return p.Pitch < -0.1 }},
		{"space jumps", ' ', func(p Player) bool { return p.Pos.Y > 6.5 }},
	} {
		g := onFloor(t)

		hold(t, g, 6, tc.key)

		if !tc.moved(g.player) {
			t.Errorf("%s: the player is at %+v", tc.name, g.player)
		}
	}
}

func TestFTurnsFlyingOnAndOff(t *testing.T) {
	g := onFloor(t)

	hold(t, g, 1, 'f')
	if !g.player.Flying {
		t.Fatal("f did not turn flying on")
	}

	hold(t, g, 10, ' ')
	hold(t, g, 5, 'x')
	if want := 6 + flySpeed*frame*5; !near(g.player.Pos.Y, want) {
		t.Errorf("after 10 frames up and 5 down the player is at %v, wanted %v", g.player.Pos.Y, want)
	}

	hold(t, g, 1, 'F')
	if g.player.Flying {
		t.Error("a second f did not turn flying off")
	}
}

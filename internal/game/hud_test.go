package game

import (
	"errors"
	"strings"
	"testing"

	"glonk.dev/engine"
)

// drawn returns the text of a frame of the game, 100 by 30 cells.
func drawn(t *testing.T, g *Game) (string, *engine.FrameBuffer) {
	t.Helper()
	fb, err := engine.NewFrameBuffer(100, 30)
	if err != nil {
		t.Fatal(err)
	}
	g.Draw(fb)
	return fb.String(), fb
}

// press runs one frame of the game with keys pressed and returns what Update returned.
func press(g *Game, keys ...engine.Key) error {
	kb := engine.NewKeyboard()
	kb.Press(keys...)
	return g.Update(frame, kb.Input())
}

func TestTheGameOpensOnItsTitle(t *testing.T) {
	g := newGame(t)

	text, _ := drawn(t, g)

	for _, want := range []string{itemNew, itemQuit, "Enter"} {
		if !strings.Contains(text, want) {
			t.Errorf("the title screen does not show %q:\n%s", want, text)
		}
	}
	// Nothing moves the player until the game is started.
	before := g.player
	hold(t, g, 5, 'w', ' ')
	if g.player != before {
		t.Error("the player moved while the title was showing")
	}
}

func TestTheTitleIsDrawnInPixelsOverTheWorld(t *testing.T) {
	g := newGame(t)
	_, with := drawn(t, g)

	// The same view with no title: the world alone.
	without, _ := engine.NewFrameBuffer(100, 30)
	cam := g.player.Camera()
	cam.Position.Y += 4
	cam.Yaw, cam.Pitch = g.spin, 0.15
	g.rend.Draw(without.Canvas(), g.world, cam, nil)

	changed := 0
	for y := range with.Canvas().Height() {
		for x := range with.Canvas().Width() {
			if with.Canvas().At(x, y) != without.Canvas().At(x, y) {
				changed++
			}
		}
	}
	if changed < 200 {
		t.Errorf("the title changed %d pixels, wanted a name across the view", changed)
	}
}

func TestEnterOnNewWorldStartsPlay(t *testing.T) {
	g := newGame(t)

	if err := press(g, engine.KeyEnter); err != nil {
		t.Fatal(err)
	}

	if g.screen != playing {
		t.Errorf("screen %d, wanted play", g.screen)
	}
}

func TestQuitAndEscapeLeaveFromTheTitle(t *testing.T) {
	g := newGame(t)
	if err := press(g, engine.KeyDown, engine.KeyEnter); !errors.Is(err, engine.ErrQuit) {
		t.Errorf("Quit returned %v, wanted ErrQuit", err)
	}

	g = newGame(t)
	if err := press(g, engine.KeyEscape); !errors.Is(err, engine.ErrQuit) {
		t.Errorf("Escape returned %v, wanted ErrQuit", err)
	}
}

func TestEscapePausesAndTheWorldStandsStill(t *testing.T) {
	g := onFloor(t)

	if err := press(g, engine.KeyEscape); err != nil {
		t.Fatal(err)
	}
	before := g.player
	hold(t, g, 5, 'w', 'q')

	if g.screen != paused || g.player != before || g.world.At(20, 5, 21) == 0 {
		t.Errorf("screen %d, player moved %v: wanted a pause with nothing changed", g.screen, g.player != before)
	}
	text, _ := drawn(t, g)
	for _, want := range []string{"paused", itemResume, itemSave, "W A S D", "break"} {
		if !strings.Contains(text, want) {
			t.Errorf("the pause menu does not show %q:\n%s", want, text)
		}
	}
}

func TestThePauseMenuResumesAndQuits(t *testing.T) {
	for _, tc := range []struct {
		name string
		keys []engine.Key
		quit bool
	}{
		{"Escape resumes", []engine.Key{engine.KeyEscape}, false},
		{"Resume resumes", []engine.Key{engine.KeyEnter}, false},
		{"Save and quit quits", []engine.Key{engine.KeyDown, engine.KeyEnter}, true},
	} {
		g := onFloor(t)
		press(g, engine.KeyEscape)

		err := press(g, tc.keys...)

		if quit := errors.Is(err, engine.ErrQuit); quit != tc.quit {
			t.Errorf("%s: got %v", tc.name, err)
		}
		if !tc.quit && g.screen != playing {
			t.Errorf("%s: screen %d, wanted play", tc.name, g.screen)
		}
	}
}

func TestPlayShowsACrosshairTheBlockInUseAndWhereThePlayerIs(t *testing.T) {
	g := onFloor(t)
	hold(t, g, 1, '8')

	text, fb := drawn(t, g)

	if got := fb.At(50, 15).Character; got != '+' {
		t.Errorf("the middle of the view is %q, wanted a crosshair", got)
	}
	for _, want := range []string{"fps", "20 6 20", "brick", "1", "9"} {
		if !strings.Contains(text, want) {
			t.Errorf("the view does not show %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "flying") {
		t.Error("the view says flying, and the player is not")
	}

	hold(t, g, 1, 'f')
	if text, _ = drawn(t, g); !strings.Contains(text, "flying") {
		t.Error("the view does not say flying")
	}
}

func TestTheHotbarShowsAPictureOfEachBlock(t *testing.T) {
	g := onFloor(t)
	_, fb := drawn(t, g)
	c := fb.Canvas()
	x0, y0 := barOrigin(fb)

	// The middle of a slot's picture, in pixels.
	at := func(slot int) (r, g, b uint8) {
		return c.At(x0+1+slot*slotPitch+slotSize/2, 2*(y0+1)+slotSize/2).RGB()
	}
	if r, g, b := at(2); max(r, g, b)-min(r, g, b) > 24 {
		t.Errorf("the stone slot is %d %d %d, wanted gray", r, g, b)
	}
	if r, g, _ := at(7); int(r) < int(g)+25 {
		t.Errorf("the brick slot is red %d green %d, wanted red", r, g)
	}
	if r, g, b := at(0); g <= r || g <= b {
		t.Errorf("the grass slot is %d %d %d, wanted green", r, g, b)
	}
}

// Every screen draws on a frame of any size, down to nothing, without failing.
func TestEveryScreenDrawsAtAnySize(t *testing.T) {
	for _, size := range [][2]int{{1, 1}, {3, 2}, {10, 4}, {30, 9}, {80, 24}, {250, 70}} {
		for _, s := range []screen{onTitle, confirming, playing, paused} {
			g := onFloor(t)
			g.show(s)
			fb, err := engine.NewFrameBuffer(size[0], size[1])
			if err != nil {
				t.Fatal(err)
			}

			g.Draw(fb)
		}
	}
}

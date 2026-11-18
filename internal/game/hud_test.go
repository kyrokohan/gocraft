package game

import (
	"errors"
	"strings"
	"testing"
	"time"

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
	cam := g.camera()
	cam.Position.Y += 4
	cam.Yaw, cam.Pitch = g.spin, 0.15
	g.rend.Draw(without.Canvas(), g.world, cam, g.sky)

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

// startNew picks New world on the title, which has no saved world on it, and then
// Survival or Creative.
func startNew(t *testing.T, g *Game, survival bool) {
	t.Helper()
	keys := [][]engine.Key{{engine.KeyEnter}, {engine.KeyEnter}}
	if !survival {
		keys[1] = []engine.Key{engine.KeyDown, engine.KeyEnter}
	}
	for _, k := range keys {
		if err := press(g, k...); err != nil {
			t.Fatal(err)
		}
	}
}

func TestANewWorldAsksHowItIsPlayedAndThenStartsPlay(t *testing.T) {
	g := newGame(t)
	if err := press(g, engine.KeyEnter); err != nil {
		t.Fatal(err)
	}
	text, _ := drawn(t, g)
	if g.screen != choosing || !strings.Contains(text, itemSurvival) || !strings.Contains(text, itemCreative) {
		t.Fatalf("screen %d, wanted the question of how to play:\n%s", g.screen, text)
	}
	if press(g, engine.KeyEscape); g.screen != onTitle {
		t.Errorf("Escape on the question went to screen %d, wanted the title", g.screen)
	}

	for _, survival := range []bool{true, false} {
		g := newGame(t)
		startNew(t, g, survival)
		if g.screen != playing || g.survival != survival {
			t.Errorf("screen %d in survival %v, wanted play in survival %v", g.screen, g.survival, survival)
		}
	}
}

func TestQuitAndEscapeLeaveFromTheTitle(t *testing.T) {
	g := newGame(t)
	if err := press(g, engine.KeyUp, engine.KeyEnter); !errors.Is(err, engine.ErrQuit) {
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
		{"Save and quit quits", []engine.Key{engine.KeyUp, engine.KeyEnter}, true},
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
	x0, y0 := barOrigin(fb.Width(), fb.Height())

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

func TestTheStatusLineShowsTheFrameRateGlonkReports(t *testing.T) {
	g := onFloor(t)

	g.Stats(engine.Stats{FPS: 23.6})

	if text, _ := drawn(t, g); !strings.Contains(text, "24 fps") {
		t.Errorf("the status line does not show 24 fps:\n%s", text)
	}
}

func TestF3ShowsAndHidesTheFrameNumbers(t *testing.T) {
	g := onFloor(t)
	g.Stats(engine.Stats{
		FPS:    29.6,
		Update: 180 * time.Microsecond,
		Draw:   2300 * time.Microsecond,
		Write:  800 * time.Microsecond,
		Bytes:  7100,
	})
	numbers := []string{"update", "0.2 ms", "draw", "2.3 ms", "write", "0.8 ms", "7.1 KB", "seed"}

	if text, _ := drawn(t, g); strings.Contains(text, "update") {
		t.Fatalf("the frame numbers show before F3 is pressed:\n%s", text)
	}

	if err := press(g, engine.KeyF3); err != nil {
		t.Fatal(err)
	}
	text, _ := drawn(t, g)
	for _, want := range numbers {
		if !strings.Contains(text, want) {
			t.Errorf("after F3 the view does not show %q:\n%s", want, text)
		}
	}

	if err := press(g, engine.KeyF3); err != nil {
		t.Fatal(err)
	}
	if text, _ = drawn(t, g); strings.Contains(text, "update") {
		t.Errorf("a second F3 did not hide the frame numbers:\n%s", text)
	}
}

func TestAHintSaysWhereOnlyOneKeyCanBeHeld(t *testing.T) {
	const hint = "one key at a time"

	g := onFloor(t)
	if err := g.Update(frame, engine.NewKeyboard().Input()); err != nil {
		t.Fatal(err)
	}
	if text, _ := drawn(t, g); strings.Contains(text, hint) {
		t.Errorf("the hint shows on a terminal that reports key releases:\n%s", text)
	}

	g = onFloor(t)
	if err := g.Update(frame, engine.NewLegacyKeyboard().Input()); err != nil {
		t.Fatal(err)
	}
	if text, _ := drawn(t, g); !strings.Contains(text, hint) {
		t.Errorf("no hint on a terminal that reports no key releases:\n%s", text)
	}
}

func TestThePauseMenuDimsTheWorldBehindIt(t *testing.T) {
	g := onFloor(t)
	g.player.Pitch = -0.5 // at the floor, so the corner of the view is stone

	// The cell at the left edge, half way down: clear of the menu and the hotbar.
	light := func() int {
		_, fb := drawn(t, g)
		r, g, b := fb.At(0, 15).Background.RGB()
		return int(r) + int(g) + int(b)
	}
	playing := light()
	press(g, engine.KeyEscape)
	paused := light()

	if playing < 60 || paused > playing*2/3 {
		t.Errorf("the world is %d bright in play and %d behind the pause menu, wanted it dimmed", playing, paused)
	}
}

func TestAHeadlessRunTellsTheGameItsFrameNumbers(t *testing.T) {
	g := newGame(t)

	runHeadless(t, &quitAfter{Game: g, frames: 4}, "")

	if g.stats.Bytes == 0 {
		t.Error("the game was not told how many bytes its frames took")
	}
}

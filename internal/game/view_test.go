package game

import (
	"math"
	"math/rand/v2"
	"path/filepath"
	"strings"
	"testing"

	"glonk.dev/engine"
	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/world"
)

// fromBehind switches a game to the view from behind the player.
func fromBehind(t *testing.T, g *Game) *Game {
	t.Helper()
	hold(t, g, 1, 'v')
	if g.settings.View != viewBehind {
		t.Fatal("V did not switch to the view from behind")
	}
	return g
}

// wallAt stands a wall of stone across the floor world at a z.
func wallAt(g *Game, z int) {
	for y := 6; y < 14; y++ {
		for x := 12; x < 30; x++ {
			g.world.Set(x, y, z, world.Stone)
		}
	}
}

func TestVSwitchesTheViewAndTheGameRemembersIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	start := func() *Game {
		g, err := New(Options{Seed: 1, world: floor(), Settings: path})
		if err != nil {
			t.Fatal(err)
		}
		g.Play()
		g.player = *standing()
		return g
	}

	g := start()
	if _, behind := g.viewed(); behind {
		t.Fatal("a new player sees from behind, wanted through the eyes")
	}
	hold(t, g, 3, 'v') // held for three frames: only its going down counts
	if _, behind := g.viewed(); !behind {
		t.Fatal("V did not switch to the view from behind")
	}

	again := start()
	if _, behind := again.viewed(); !behind {
		t.Error("the view from behind was not remembered")
	}
	hold(t, again, 1, 'V')
	if _, behind := again.viewed(); behind {
		t.Error("V again did not switch back to the eyes")
	}
}

func TestTheCameraFromBehindLooksOverTheRightShoulder(t *testing.T) {
	g := fromBehind(t, onFloor(t))
	g.player.Pitch = -0.3
	eyes := g.eyes()
	cam, behind := g.viewed()

	off := cam.Position.Sub(eyes.Position)
	back, right := -off.Dot(eyes.Look()), off.Dot(eyes.Right())
	// The camera is behindBack behind the shoulder, which is a little above the eyes.
	if !behind || math.Abs(back-behindBack) > 0.15 || right < 0.4 || right > 0.8 {
		t.Errorf("behind %v: the camera is %.2f back and %.2f to the right of the eyes, wanted %.1f back and over the right shoulder", behind, back, right, behindBack)
	}
	if cam.Yaw != eyes.Yaw || cam.Pitch != eyes.Pitch || cam.FOV != eyes.FOV {
		t.Error("the camera from behind does not look the way the eyes do")
	}
}

func TestAWallBehindThePlayerPullsTheCameraInAndThenHidesTheBody(t *testing.T) {
	g := fromBehind(t, onFloor(t))
	wallAt(g, 17) // its face is 2.5 behind the player
	cam, behind := g.viewed()
	if !behind || cam.Position.Z < 18.2 || cam.Position.Z > 18.6 {
		t.Errorf("behind %v: with a wall at 18 behind it the camera is at z=%.2f, wanted just short of the wall", behind, cam.Position.Z)
	}

	wallAt(g, 19) // and now half a block behind
	cam, behind = g.viewed()
	if behind || cam.Position != g.eyes().Position {
		t.Errorf("behind %v at %v: with no room behind, wanted the view from the eyes", behind, cam.Position)
	}
	drawn(t, g)
	if len(g.cast) != 0 {
		t.Error("the body is drawn with the camera at the eyes")
	}
}

func TestTheCameraFromBehindIsNeverInsideABlock(t *testing.T) {
	g, err := New(Options{Seed: 7})
	if err != nil {
		t.Fatal(err)
	}
	g.Play()
	g.settings.View = viewBehind

	rng := rand.New(rand.NewPCG(1, 2))
	behinds := 0
	for range 200 {
		x, z := rng.IntN(120)-60, rng.IntN(120)-60
		g.player = Player{
			Pos: geometry.Vector3{X: float64(x) + 0.5, Y: float64(g.world.Ground(x, z) + 1), Z: float64(z) + 0.5},
			Yaw: rng.Float64() * 2 * math.Pi, Pitch: rng.Float64()*2 - 1,
		}
		cam, behind := g.viewed()
		if behind {
			behinds++
		}
		at := cam.Position
		if b := g.world.At(int(math.Floor(at.X)), int(math.Floor(at.Y)), int(math.Floor(at.Z))); b.Info().Solid {
			t.Errorf("from %v the camera is at %v, inside %s", g.player.Pos, at, b.Info().Name)
		}
	}
	if behinds < 60 {
		t.Errorf("only %d of 200 places had room for the view from behind", behinds)
	}
}

func TestFromBehindTheMiddleOfTheViewAimsBeyondThePlayer(t *testing.T) {
	g := fromBehind(t, lookingDown(t))
	hold(t, g, 1)
	cam, behind := g.viewed()

	// The first block along the middle of the view, from where it passes the player.
	want, ok := g.world.Target(cam.Position.Add(cam.Look().Scale(behindBack)), cam.Look(), reach)
	if !behind || !ok || !g.aimed || g.target.X != want.X || g.target.Y != want.Y || g.target.Z != want.Z {
		t.Errorf("behind %v: aimed %v at %+v, wanted the block in the middle of the view, %+v", behind, g.aimed, g.target, want)
	}
}

func TestABlockBetweenTheCameraAndThePlayerIsNotAimedAt(t *testing.T) {
	g := fromBehind(t, onFloor(t))
	cam, _ := g.viewed()
	// A plant does not stop the camera, and anything can be aimed at through the eyes.
	mid := cam.Position.Add(cam.Look().Scale(behindBack / 2))
	g.world.Set(int(math.Floor(mid.X)), int(math.Floor(mid.Y)), int(math.Floor(mid.Z)), world.TallFern)
	hold(t, g, 1)

	if _, behind := g.viewed(); !behind {
		t.Fatal("a plant behind the player took the view from behind away")
	}
	if g.aimed && g.target.Block == world.TallFern {
		t.Errorf("the player aims at a plant behind its back, at %+v", g.target)
	}
}

func TestReachIsMeasuredFromThePlayerFromBehindToo(t *testing.T) {
	near := fromBehind(t, onFloor(t))
	wallAt(near, 24) // its face is 3.5 ahead of the player
	hold(t, near, 1)
	if !near.aimed || near.target.Z != 24 {
		t.Errorf("a wall 3.5 ahead: aimed %v at %+v, wanted the wall", near.aimed, near.target)
	}

	far := fromBehind(t, onFloor(t))
	wallAt(far, 27) // 6.5 ahead: in reach of the camera's ray were it longer, not of the player
	hold(t, far, 1)
	if far.aimed {
		t.Errorf("a wall 6.5 ahead is aimed at from behind: %+v", far.target)
	}
}

func TestTheBodyShowsWhatThePlayerDoes(t *testing.T) {
	g := fromBehind(t, onFloor(t))
	g.player.Pitch = 0.4
	hold(t, g, 10, 'w')
	drawn(t, g)
	if len(g.cast) != 1 {
		t.Fatalf("%d bodies are drawn from behind, wanted the player's", len(g.cast))
	}
	one := g.cast[0]
	if one.Pos != g.player.Pos || one.Yaw != g.player.Yaw || one.Tilt != 0.4 || one.Held != world.Turf.Item() {
		t.Errorf("the body is %+v, wanted it where the player is, looking up, with mossy earth in hand", one)
	}
	if math.Sin(one.Stride) == 0 || one.Crouched || one.Hurt || one.Swing != 0 {
		t.Errorf("the body of a walking player is %+v, wanted it in mid stride and no more", one)
	}

	hold(t, g, 2) // standing still
	hold(t, g, 1, 'c')
	g.flash = 0.2
	drawn(t, g)
	if one = g.cast[0]; one.Stride != 0 || !one.Crouched || !one.Hurt {
		t.Errorf("the body of a still, crouched, hurt player is %+v", one)
	}

	digging := fromBehind(t, lookingDown(t))
	hold(t, digging, 1, 'q')
	hold(t, digging, 1) // a swing begins at nothing, and shows a frame on
	drawn(t, digging)
	if len(digging.cast) != 1 || digging.cast[0].Swing <= 0 {
		t.Errorf("the body of a player who broke a block does not swing its arm: %+v", digging.cast)
	}
}

func TestTheBodyIsDrawnFromBehindAndTheHandThroughTheEyes(t *testing.T) {
	// The picture with something in hand and with nothing, in each view.
	frames := func(g *Game) (full, empty *engine.Canvas) {
		_, fb := drawn(t, g)
		full = fb.Canvas()
		g.items[g.selected] = world.Stack{}
		_, fb = drawn(t, g)
		return full, fb.Canvas()
	}
	// changed counts the pixels that differ in the lower right of the view, where the
	// hand is drawn through the eyes, and everywhere else.
	changed := func(a, b *engine.Canvas) (corner, rest int) {
		for y := range a.Height() {
			for x := range a.Width() {
				if a.At(x, y) == b.At(x, y) {
					continue
				}
				if x > a.Width()*6/10 && y > a.Height()*6/10 {
					corner++
				} else {
					rest++
				}
			}
		}
		return corner, rest
	}

	g := onFloor(t)
	drawn(t, g)
	if len(g.cast) != 0 {
		t.Error("the body is drawn through its own eyes")
	}
	if corner, _ := changed(frames(g)); corner == 0 {
		t.Error("through the eyes, what is held is not drawn in the corner of the view")
	}

	g = fromBehind(t, onFloor(t))
	if corner, rest := changed(frames(g)); corner != 0 || rest == 0 {
		t.Errorf("from behind, what is held changed %d pixels in the corner and %d on the body, wanted none and some", corner, rest)
	}
}

func TestThePauseMenuNamesTheKeysOfTheBody(t *testing.T) {
	g := onFloor(t)
	if err := press(g, engine.KeyEscape); err != nil {
		t.Fatal(err)
	}
	text, _ := drawn(t, g)
	for _, want := range []string{"V       view from behind", "C         crouch", "L       next look"} {
		if !strings.Contains(text, want) {
			t.Errorf("the pause menu does not show %q:\n%s", want, text)
		}
	}
}

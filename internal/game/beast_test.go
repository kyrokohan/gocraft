package game

import (
	"math"
	"math/rand/v2"
	"testing"

	"glonk.dev/engine"
	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/world"
)

// walker is a creature on the test floor that walks the way it faces and does not
// change its mind.
func walker(x, z, yaw float64) *beast {
	return &beast{
		Pos:     geometry.Vector3{X: x, Y: 6, Z: z},
		Yaw:     yaw,
		walking: true,
		timer:   math.Inf(1),
		rng:     rand.New(rand.NewPCG(1, 2)),
	}
}

// mossbackWidth is how wide the box is that blocks stop a mossback in.
func mossbackWidth() float64 {
	width, _ := render.Size(render.Mossback, false)
	return width
}

// only returns the beasts of a kind.
func only(all []beast, kind render.Kind) (of []beast) {
	for _, c := range all {
		if c.kind == kind {
			of = append(of, c)
		}
	}
	return of
}

// live steps a creature for a time, and calls each (if it is not nil) after every step.
func live(c *beast, w *world.World, seconds float64, each func()) {
	for range int(math.Round(seconds / frame)) {
		c.Step(w, frame)
		if each != nil {
			each()
		}
	}
}

func TestACreatureLandsOnTheGround(t *testing.T) {
	w, c := floor(), walker(20.5, 20.5, 0)
	c.walking = false
	c.Pos.Y = 12

	live(c, w, 3, nil)

	if !near(c.Pos.Y, 6) || !c.OnGround {
		t.Errorf("after falling the creature is at height %v, on ground %v; wanted 6, true", c.Pos.Y, c.OnGround)
	}
}

func TestACreatureWalksTheWayItFaces(t *testing.T) {
	w := floor()
	for _, tc := range []struct{ yaw, dx, dz float64 }{
		{0, 0, 1}, {math.Pi / 2, 1, 0}, {math.Pi, 0, -1},
	} {
		c := walker(20.5, 20.5, tc.yaw)

		live(c, w, 2, nil)

		wantX, wantZ := 20.5+tc.dx*2*creatureSpeed, 20.5+tc.dz*2*creatureSpeed
		if math.Abs(c.Pos.X-wantX) > 0.05 || math.Abs(c.Pos.Z-wantZ) > 0.05 {
			t.Errorf("facing %v the creature walked to %v, %v; wanted %v, %v", tc.yaw, c.Pos.X, c.Pos.Z, wantX, wantZ)
		}
	}
}

func TestTheLegsMoveOnlyWhileItWalks(t *testing.T) {
	w, c := floor(), walker(20.5, 20.5, 0)

	live(c, w, 0.5, nil)
	if c.Stride == 0 {
		t.Error("the legs did not move while the creature walked")
	}

	c.walking = false
	before := c.Pos
	live(c, w, 0.5, nil)
	if c.Stride != 0 || c.Pos != before {
		t.Errorf("standing, the creature moved to %v with its legs at %v", c.Pos, c.Stride)
	}
}

func TestACreatureHopsUpOneBlock(t *testing.T) {
	w := floor()
	for x := range 40 {
		for z := 24; z < 40; z++ {
			w.Set(x, 6, z, world.Stone)
		}
	}
	c := walker(20.5, 20.5, 0)

	live(c, w, 6, nil)

	if c.Pos.Z < 25 || !near(c.Pos.Y, 7) {
		t.Errorf("the creature is at z=%v height %v, wanted it up on the step", c.Pos.Z, c.Pos.Y)
	}
}

func TestACreatureTurnsAwayFromAWall(t *testing.T) {
	w := floor()
	for x := range 40 {
		w.Set(x, 6, 24, world.Brick)
		w.Set(x, 7, 24, world.Brick)
	}
	c := walker(20.5, 20.5, 0)

	farthest, turned := 0.0, false
	live(c, w, 20, func() {
		farthest = max(farthest, c.Pos.Z)
		turned = turned || c.Yaw != 0
	})

	if farthest > 24-mossbackWidth()/2 {
		t.Errorf("the creature got to z=%v, through a wall at 24", farthest)
	}
	if !turned {
		t.Error("the creature never turned away from the wall")
	}
}

func TestACreatureKeepsOutOfWater(t *testing.T) {
	w := floor()
	for x := range 40 {
		for z := 24; z < 40; z++ {
			w.Set(x, 5, z, world.Water) // a pool level with the ground
		}
	}
	c := walker(20.5, 20.5, 0)

	live(c, w, 40, func() {
		if w.At(int(math.Floor(c.Pos.X)), 5, int(math.Floor(c.Pos.Z))) == world.Water {
			t.Fatalf("the creature walked into the water at %v", c.Pos)
		}
	})
}

func TestACreatureKeepsBackFromALongDrop(t *testing.T) {
	w := floor() // which ends at 40 on x and z, with nothing beyond it
	c := walker(36.5, 36.5, 0)

	live(c, w, 40, func() {
		if c.Pos.Y < 5.9 {
			t.Fatalf("the creature fell off the edge, to %v", c.Pos)
		}
	})
}

func TestACreatureWalksAndStandsByTurns(t *testing.T) {
	w := floor()
	c := walker(20.5, 20.5, 0)
	c.walking, c.timer = false, 0 // free to change its mind

	walked, stood, changes, was := 0, 0, 0, false
	live(c, w, 90, func() {
		if c.walking {
			walked++
		} else {
			stood++
		}
		if c.walking != was {
			changes++
		}
		was = c.walking
	})

	if walked == 0 || stood == 0 || changes < 6 {
		t.Errorf("in 90 seconds the creature walked %d frames, stood %d and changed %d times", walked, stood, changes)
	}
	if !near(c.Pos.Y, 6) {
		t.Errorf("the creature ended at height %v, wanted it still on the floor", c.Pos.Y)
	}
}

func TestAHerdStartsOnGrassRoundThePlayer(t *testing.T) {
	for seed := int64(1); seed <= 3; seed++ {
		w := world.Generate(seed)
		spawn := w.Spawn()

		all := only(newFauna(w, spawn).all, render.Mossback)
		again := only(newFauna(w, spawn).all, render.Mossback)

		if len(all) != creatureCount {
			t.Fatalf("seed %d: %d creatures, wanted %d", seed, len(all), creatureCount)
		}
		for i, c := range all {
			x, y, z := int(math.Floor(c.Pos.X)), int(c.Pos.Y), int(math.Floor(c.Pos.Z))
			if at := w.At(x, y, z); w.At(x, y-1, z) != world.Turf || (at != world.Air && !at.Info().Plant) || w.At(x, y+1, z) != world.Air {
				t.Errorf("seed %d: creature %d at %v is not on grass with room above", seed, i, c.Pos)
			}
			if d := c.Pos.Distance(spawn); d < 3 || d > 80 {
				t.Errorf("seed %d: creature %d is %v from the player", seed, i, d)
			}
			if again[i].Pos != c.Pos || again[i].Yaw != c.Yaw {
				t.Errorf("seed %d: creature %d is not where it was the first time", seed, i)
			}
		}
	}
}

// The world has no end, so the herd goes with the player: creatures left far behind
// are let go, and new ones are found a place near where the player now is.
func TestTheHerdGoesWithThePlayer(t *testing.T) {
	w := world.Generate(2)
	at := w.Spawn()
	h := newFauna(w, at)

	// The player goes a long way, in one step, and stays there a while.
	at.X += 600
	w.Follow(at.X, at.Z)
	w.LoadAll()
	for range 40 * 30 {
		h.step(w, at, frame)
	}

	if len(h.all) < creatureCount/2 {
		t.Fatalf("%d creatures round the player after 40 seconds in a new place, wanted most of %d", len(h.all), creatureCount)
	}
	for i, c := range h.all {
		if d := c.Pos.Distance(at); d > creatureGone {
			t.Errorf("creature %d is %v from the player, wanted those left behind let go", i, d)
		}
		if c.Pos.Y < 1 {
			t.Errorf("creature %d fell out of the world, to %v", i, c.Pos)
		}
	}
}

func TestCreaturesFarFromThePlayerAreNotMoved(t *testing.T) {
	w := floor()
	far := walker(20.5, 20.5, 0)
	h := &fauna{all: []beast{*far}, rng: far.rng, timer: math.Inf(1)}

	h.step(w, geometry.Vector3{X: 20.5 + creatureAwake + 5, Y: 6, Z: 20.5}, 1)

	if h.all[0].Pos != far.Pos {
		t.Error("a creature beyond where the ground is loaded was moved")
	}
}

func TestCreaturesMoveInPlayAndStandStillInAPause(t *testing.T) {
	g := onFloor(t)
	g.fauna.all = []beast{*walker(22.5, 22.5, 0)}

	hold(t, g, 10)
	moved := g.fauna.all[0].Pos
	if moved.Z <= 22.5 {
		t.Fatal("the creature did not move in play")
	}

	press(g, engine.KeyEscape)
	hold(t, g, 10)
	if g.fauna.all[0].Pos != moved {
		t.Error("the creature moved while the game was paused")
	}
}

func TestACreatureInViewIsDrawn(t *testing.T) {
	g := onFloor(t) // the player faces +z from 20.5, 20.5
	g.fauna.all = nil
	_, without := drawn(t, g)

	g.fauna.all = []beast{*walker(20.5, 23.5, math.Pi)}
	_, with := drawn(t, g)

	n := 0
	for y := range with.Canvas().Height() {
		for x := range with.Canvas().Width() {
			if with.Canvas().At(x, y) != without.Canvas().At(x, y) {
				n++
			}
		}
	}
	if n < 60 {
		t.Errorf("a creature three blocks ahead changed %d pixels of the frame", n)
	}
}

// A whole frame of play: the blocks, the clouds, the creatures and the HUD, at 240 by
// 67 cells, by day and by night, when the stars are out.
func BenchmarkFrame(b *testing.B) {
	g, err := New(Options{Seed: 1})
	if err != nil {
		b.Fatal(err)
	}
	g.Play()
	fb, err := engine.NewFrameBuffer(240, 67)
	if err != nil {
		b.Fatal(err)
	}

	ahead := g.player.Camera().Look()
	ahead.Y = 0
	ahead = ahead.Normalize()
	for _, c := range []struct {
		name   string
		days   float64
		gloams int
		view   string
	}{{"day", 0, 0, viewEyes}, {"day from behind", 0, 0, viewBehind}, {"night", 0.7, 0, viewEyes}, {"night with gloams", 0.7, gloamMost, viewEyes}} {
		b.Run(c.name, func(b *testing.B) {
			g.days, g.gloams, g.settings.View = c.days, nil, c.view
			for i := range c.gloams {
				at := g.player.Pos.Add(ahead.Scale(4 + 2*float64(i)))
				at.Y = float64(g.world.Ground(int(math.Floor(at.X)), int(math.Floor(at.Z))) + 1)
				g.gloams = append(g.gloams, gloam{pos: at, yaw: math.Pi, health: gloamHealth})
			}
			for b.Loop() {
				fb.Clear()
				g.Draw(fb)
			}
		})
	}
}

package game

import (
	"math"
	"testing"

	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/world"
)

const frame = 1.0 / 30

// floor is a world with ground of stone up to layer 5, so its top is at height 6, over
// a square of 40 by 40 blocks from the origin of the world. Beyond the square there is
// nothing, down to the bottom of the world.
func floor() *world.World {
	w := world.New()
	for z := range 40 {
		for x := range 40 {
			for y := range 6 {
				w.Set(x, y, z, world.Stone)
			}
		}
	}
	return w
}

// standing is a player on that floor, well away from the edge, facing +z.
func standing() *Player {
	return &Player{Pos: geometry.Vector3{X: 20.5, Y: 6, Z: 20.5}}
}

// run steps a player for a time with the same controls held.
func run(p *Player, w *world.World, c controls, seconds float64) {
	for range int(math.Round(seconds / frame)) {
		p.Step(w, c, frame)
	}
}

func near(a, b float64) bool {
	return math.Abs(a-b) < 1e-4
}

func TestGravityLandsThePlayerOnTheGround(t *testing.T) {
	w, p := floor(), standing()
	p.Pos.Y = 14

	run(p, w, controls{}, 3)

	if !near(p.Pos.Y, 6) || !p.OnGround {
		t.Errorf("after falling the player is at height %v, on ground %v; wanted 6, true", p.Pos.Y, p.OnGround)
	}
}

func TestWalkingGoesTheWayThePlayerFaces(t *testing.T) {
	w := floor()
	for _, tc := range []struct {
		name   string
		yaw    float64
		c      controls
		dx, dz float64
	}{
		{"ahead", 0, controls{forward: 1}, 0, walkSpeed},
		{"back", 0, controls{forward: -1}, 0, -walkSpeed},
		{"right", 0, controls{strafe: 1}, walkSpeed, 0},
		{"ahead, turned right", math.Pi / 2, controls{forward: 1}, walkSpeed, 0},
		{"both at once is no faster", 0, controls{forward: 1, strafe: 1}, walkSpeed / math.Sqrt2, walkSpeed / math.Sqrt2},
	} {
		p := standing()
		p.Yaw = tc.yaw

		run(p, w, tc.c, 1)

		if !near(p.Pos.X, 20.5+tc.dx) || !near(p.Pos.Z, 20.5+tc.dz) {
			t.Errorf("%s: moved to %v, %v; wanted %v, %v", tc.name, p.Pos.X, p.Pos.Z, 20.5+tc.dx, 20.5+tc.dz)
		}
	}
}

func TestArrowsTurnAndTiltTheView(t *testing.T) {
	w, p := floor(), standing()

	run(p, w, controls{turn: 1, tilt: 1}, 0.5)

	if !near(p.Yaw, turnSpeed*0.5) || !near(p.Pitch, tiltSpeed*0.5) {
		t.Errorf("yaw %v pitch %v, wanted %v and %v", p.Yaw, p.Pitch, turnSpeed*0.5, tiltSpeed*0.5)
	}

	run(p, w, controls{tilt: 1}, 5)

	if p.Pitch != maxPitch {
		t.Errorf("pitch %v, wanted it held at %v", p.Pitch, maxPitch)
	}
}

func TestAWallStopsThePlayerAndLetsItSlide(t *testing.T) {
	w := floor()
	for x := range 40 {
		w.Set(x, 6, 24, world.Brick)
		w.Set(x, 7, 24, world.Brick)
	}
	p := standing()

	run(p, w, controls{forward: 1, strafe: 1}, 3)

	if !near(p.Pos.Z, 24-playerWidth/2) {
		t.Errorf("the player is at z=%v, wanted it against the wall at %v", p.Pos.Z, 24-playerWidth/2)
	}
	if p.Pos.X < 25 {
		t.Errorf("the player is at x=%v: it should slide along the wall", p.Pos.X)
	}
}

func TestAJumpClimbsOneBlockAndNotTwo(t *testing.T) {
	for _, tc := range []struct {
		high   int
		climbs bool
	}{{1, true}, {2, false}} {
		w := floor()
		for x := range 40 {
			for z := 24; z < 40; z++ {
				for y := range tc.high {
					w.Set(x, 6+y, z, world.Stone)
				}
			}
		}
		p := standing()

		run(p, w, controls{forward: 1, rise: true}, 3)

		if climbed := p.Pos.Z > 25; climbed != tc.climbs {
			t.Errorf("a step %d high: the player is at z=%v height %v, climbed %v, wanted %v",
				tc.high, p.Pos.Z, p.Pos.Y, climbed, tc.climbs)
		}
	}
}

func TestACeilingStopsAJump(t *testing.T) {
	w, p := floor(), standing()
	w.Set(20, 8, 20, world.Stone) // its underside is at 8: 0.2 above the player's head

	top := 0.0
	for range 30 {
		p.Step(w, controls{rise: true}, frame)
		top = max(top, p.Pos.Y)
	}

	if top > 8-playerHeight+1e-4 {
		t.Errorf("the player rose to %v, through a ceiling that allows %v", top, 8-playerHeight)
	}
}

func TestFlyingIgnoresGravityAndRisesAndSinks(t *testing.T) {
	w, p := floor(), standing()
	p.Pos.Y = 20

	p.Step(w, controls{toggleFly: true}, frame)
	run(p, w, controls{}, 1)
	if !p.Flying || !near(p.Pos.Y, 20) {
		t.Fatalf("flying %v at height %v, wanted true and 20", p.Flying, p.Pos.Y)
	}

	run(p, w, controls{rise: true}, 1)
	if !near(p.Pos.Y, 20+flySpeed) {
		t.Errorf("after rising the player is at %v, wanted %v", p.Pos.Y, 20+flySpeed)
	}

	run(p, w, controls{sink: true}, 0.5)
	if !near(p.Pos.Y, 20+flySpeed/2) {
		t.Errorf("after sinking the player is at %v, wanted %v", p.Pos.Y, 20+flySpeed/2)
	}

	p.Step(w, controls{toggleFly: true}, frame)
	run(p, w, controls{}, 4)
	if p.Flying || !near(p.Pos.Y, 6) {
		t.Errorf("after flying is turned off: flying %v at height %v, wanted false and 6", p.Flying, p.Pos.Y)
	}
}

func TestALongFrameDoesNotPassThroughTheFloor(t *testing.T) {
	w := world.New()
	for z := range 40 {
		for x := range 40 {
			w.Set(x, 5, z, world.Glass) // one block thick, with nothing under it
		}
	}
	p := standing()
	p.Pos.Y = 30

	for range 40 {
		p.Step(w, controls{}, 0.25) // the longest frame glonk hands out
	}

	if !near(p.Pos.Y, 6) {
		t.Errorf("the player is at height %v, wanted it on the floor at 6", p.Pos.Y)
	}
}

// The world has no edge: the ground goes on below 0 on x and z, and so does the player.
func TestThePlayerWalksOnBelowZero(t *testing.T) {
	w := floor()
	for z := -12; z < 0; z++ {
		for x := -12; x < 40; x++ {
			w.Set(x, 5, z, world.Stone)
			w.Set(z, 5, x, world.Stone)
		}
	}
	p := standing()
	p.Pos = geometry.Vector3{X: 2.5, Y: 6, Z: 2.5}

	run(p, w, controls{forward: -1, strafe: -1}, 3)

	want := 2.5 - 3*walkSpeed/math.Sqrt2
	if !near(p.Pos.X, want) || !near(p.Pos.Z, want) || !near(p.Pos.Y, 6) {
		t.Errorf("the player is at %v, wanted it at %v, 6, %v: past 0 and still on the ground", p.Pos, want, want)
	}
}

// Under the world there is a floor that nothing falls through.
func TestTheBottomOfTheWorldIsAFloor(t *testing.T) {
	w, p := world.New(), standing()

	run(p, w, controls{}, 4)

	if !near(p.Pos.Y, 0) || !p.OnGround {
		t.Errorf("with no ground the player is at height %v, on ground %v; wanted 0, true", p.Pos.Y, p.OnGround)
	}
}

func TestWaterSlowsAFallAndSpaceSwimsUp(t *testing.T) {
	w := floor()
	for z := range 40 {
		for x := range 40 {
			for y := 6; y < 16; y++ {
				w.Set(x, y, z, world.Water)
			}
		}
	}
	p := standing()
	p.Pos.Y = 15

	run(p, w, controls{}, 1)
	if sunk := 15 - p.Pos.Y; sunk <= 0 || sunk > sinkSpeed {
		t.Errorf("in a second the player sank %v, wanted more than 0 and at most %v", sunk, sinkSpeed)
	}

	run(p, w, controls{rise: true}, 6)
	if p.Pos.Y < 15.5 {
		t.Errorf("swimming up, the player is at %v, wanted it at the surface near 16", p.Pos.Y)
	}
}

func TestTheSwimmerCanClimbOutOntoAShore(t *testing.T) {
	w := floor()
	for z := range 40 {
		for x := range 40 {
			if z < 24 {
				w.Set(x, 6, z, world.Water) // a pool one deep, its surface at 7
			} else {
				w.Set(x, 6, z, world.Sand) // and a shore level with the surface
			}
		}
	}
	p := standing()

	run(p, w, controls{forward: 1, rise: true}, 4)

	if p.Pos.Z < 25 || p.Pos.Y < 7-1e-4 {
		t.Errorf("the player is at z=%v height %v, wanted it up on the shore", p.Pos.Z, p.Pos.Y)
	}
}

func TestTheCameraIsAtThePlayersEyes(t *testing.T) {
	p := standing()
	p.Yaw, p.Pitch = 0.5, -0.25

	cam := p.Camera()

	if cam.Position != (geometry.Vector3{X: 20.5, Y: 6 + eyeHeight, Z: 20.5}) || cam.Yaw != 0.5 || cam.Pitch != -0.25 || cam.FOV != defaultSettings.fov() {
		t.Errorf("got camera %+v", cam)
	}
}

package game

import (
	"math"
	"math/rand/v2"
	"slices"

	"glonk.dev/geometry"
	"glonk.dev/mesh"

	"github.com/kyrokohan/gocraft/internal/render"
	"github.com/kyrokohan/gocraft/internal/world"
)

// How a mossback lives. Distances are in blocks, times in seconds.
const (
	creatureCount = 12    // in a world
	creatureRange = 48.0  // how far from the player they start, at the most
	creatureAwake = 88.0  // how far from the player they still move: the ground is loaded that far
	creatureGone  = 104.0 // how far from the player they are let go
	creatureSpeed = 1.3   // walking
	creatureHop   = 8.0   // the speed it leaves the ground at: enough to clear one block
	strideRate    = 8.0   // radians of leg swing a second
	lookAhead     = 0.35  // how far past its own front it looks before it steps
	longDrop      = 3     // blocks: it does not walk off anything deeper
	rethink       = 0.4   // how long it stands after something turns it back
)

// Creature is one mossback: an animal that walks about, stands a while, and walks on.
// It is stopped by blocks as the player is, hops up a single block, and turns back from
// walls, water and long drops.
type Creature struct {
	// Pos is the middle of the bottom of its body: the point between its feet.
	Pos geometry.Vector3

	// Yaw is the way it faces, as a camera's Yaw is.
	Yaw float64

	// Stride is how far through a step its legs are, in radians, and 0 when it stands.
	Stride float64

	// OnGround is true while it stands on a block.
	OnGround bool

	velY    float64    // speed upward
	walking bool       // or standing
	timer   float64    // until it changes its mind
	rng     *rand.Rand // its own, so that what it does can be told from a seed
}

// herd is the mossbacks round the player. The world has no end, so they cannot all be
// placed when it is made: they are found a place on grass near the player as it moves,
// up to creatureCount of them, and those left far behind are let go.
type herd struct {
	all   []Creature
	rng   *rand.Rand // where the next one goes: from the world's seed
	born  uint64     // how many there have been, to give each a generator of its own
	timer float64    // until the herd is next looked over
}

// newHerd returns the creatures of a world round a place, as many as there is grass
// for, up to creatureCount. The same world and place always give the same herd.
func newHerd(w *world.World, round geometry.Vector3) *herd {
	h := &herd{rng: rand.New(rand.NewPCG(uint64(w.Seed), 0x6d6f7373))}
	for try := 0; len(h.all) < creatureCount && try < 60*creatureCount; try++ {
		h.add(w, round, 4, creatureRange)
	}
	return h
}

// add tries one place for a new creature, between near and far from a point: on grass,
// with room above, and not on top of another. It reports whether the place would do.
func (h *herd) add(w *world.World, round geometry.Vector3, near, far float64) bool {
	angle, away := h.rng.Float64()*2*math.Pi, near+h.rng.Float64()*(far-near)
	x := int(math.Floor(round.X + math.Cos(angle)*away))
	z := int(math.Floor(round.Z + math.Sin(angle)*away))
	y := w.Ground(x, z) // under any plant
	if w.At(x, y, z) != world.Grass || y+2 >= world.Height || w.At(x, y+2, z) != world.Air {
		return false // not grass, or under a tree, or a place that is not loaded
	}
	pos := geometry.Vector3{X: float64(x) + 0.5, Y: float64(y + 1), Z: float64(z) + 0.5}
	for _, other := range h.all {
		if other.Pos.Distance(pos) < 3 {
			return false
		}
	}
	h.born++
	own := rand.New(rand.NewPCG(uint64(w.Seed), h.born))
	h.all = append(h.all, Creature{
		Pos:   pos,
		Yaw:   own.Float64()*2*math.Pi - math.Pi,
		timer: own.Float64() * 2,
		rng:   own,
	})
	return true
}

// step moves the creatures for dt seconds, and now and then looks the herd over:
// creatures far behind the player are let go, and a new one is found a place ahead if
// there are too few. Those too far off to stand on loaded ground are not moved.
func (h *herd) step(w *world.World, player geometry.Vector3, dt float64) {
	for i := range h.all {
		if c := &h.all[i]; c.Pos.Distance(player) < creatureAwake {
			c.Step(w, dt)
		}
	}

	if h.timer -= dt; h.timer > 0 {
		return
	}
	h.timer = 1
	h.all = slices.DeleteFunc(h.all, func(c Creature) bool {
		return c.Pos.Distance(player) > creatureGone
	})
	if len(h.all) < creatureCount {
		h.add(w, player, creatureRange/2, creatureAwake-8) // out of the way, where it will be met
	}
}

// Pose returns where the creature is and how it stands, for drawing.
func (c *Creature) Pose() render.Mossback {
	return render.Mossback{Pos: c.Pos, Yaw: c.Yaw, Stride: c.Stride}
}

// decide changes the creature's mind when its time is up: it stands for one to three
// seconds, or turns some new way and walks for two to five.
func (c *Creature) decide() {
	if c.walking {
		c.walking, c.timer = false, 1+c.rng.Float64()*2
		return
	}
	c.walking, c.timer = true, 2+c.rng.Float64()*3
	c.Yaw = c.rng.Float64()*2*math.Pi - math.Pi
}

// turnBack stops the creature where something is in its way. It stands a moment and
// then picks a new way to go, which is what makes it look as if it thought about it.
func (c *Creature) turnBack() {
	c.walking, c.timer = false, rethink
}

// wayAhead looks at the cell just past the creature's front and says what to do about
// it: walk on, hop up onto it, or turn back.
func (c *Creature) wayAhead(w *world.World, ahead geometry.Vector3, afloat bool) (hop, blocked bool) {
	front := c.Pos.Add(ahead.Scale(render.MossbackWidth/2 + lookAhead))
	x, y, z := int(math.Floor(front.X)), int(math.Floor(c.Pos.Y+0.1)), int(math.Floor(front.Z))

	if solid(w, x, y, z) {
		// A block at the height of its feet. One block is a step it can hop, if it has
		// its feet on something and there is room on top.
		return (c.OnGround || afloat) && !solid(w, x, y+1, z), true
	}
	if w.At(x, y, z) == world.Water || w.At(x, y-1, z) == world.Water {
		return false, true
	}
	for down := 1; down <= longDrop; down++ {
		if solid(w, x, y-down, z) {
			return false, false // ground to step onto or down to
		}
	}
	return false, true // nothing under it for a long way
}

// Step moves the creature for dt seconds.
func (c *Creature) Step(w *world.World, dt float64) {
	if c.timer -= dt; c.timer <= 0 {
		c.decide()
	}

	afloat := inWater(w, c.Pos)
	if afloat {
		c.velY = 1.5 // it floats up, should it ever fall in
	} else {
		c.velY = max(c.velY-gravity*dt, -fallSpeed)
	}

	var move geometry.Vector3
	if c.walking {
		ahead := mesh.Camera{Yaw: c.Yaw}.Forward()
		switch hop, blocked := c.wayAhead(w, ahead, afloat); {
		case hop:
			c.velY = creatureHop
		case blocked && (c.OnGround || afloat):
			c.turnBack() // in the air it is in the middle of a hop, and goes on
		}
		if c.walking {
			move = ahead.Scale(creatureSpeed * dt)
			c.Stride += strideRate * dt
		}
	}
	if !c.walking {
		c.Stride = 0
	}
	move.Y = c.velY * dt

	stopped := travel(w, &c.Pos, render.MossbackWidth, render.MossbackHeight, move)
	c.OnGround = stopped[1] && move.Y < 0
	if stopped[1] {
		c.velY = 0
	}
	if (stopped[0] || stopped[2]) && c.walking && c.OnGround {
		c.turnBack() // it walked into something the look ahead did not see: a corner
	}
}

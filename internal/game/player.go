package game

import (
	"math"
	"unicode"

	"glonk.dev/engine"
	"glonk.dev/geometry"
	"glonk.dev/mesh"

	"github.com/kyrokohan/gocraft/internal/world"
)

// The player's body and how it moves. Distances are in blocks, times in seconds.
const (
	playerWidth  = 0.6
	playerHeight = 1.8

	walkSpeed = 4.3
	swimSpeed = 2.6
	flySpeed  = 10.0
	turnSpeed = 2.0 // radians a second
	tiltSpeed = 1.5
	maxPitch  = 1.5 // just short of straight up or down, where the view would flip

	gravity   = 28.0
	jumpSpeed = 8.6  // enough to clear one block and not two
	fallSpeed = 50.0 // the fastest fall
	sinkSpeed = 2.5  // the fastest fall in water
	swimUp    = 4.0  // enough to hop out onto a shore level with the water

	// maxStep is the farthest the player moves before blocks are looked at again. It
	// is under half a block, so no frame is long enough to carry it through a wall.
	maxStep = 0.4

	// gap is how far from a block the player is left when stopped by it. Without it,
	// rounding could leave the player a hair inside the block.
	gap = 1e-7

	// ceiling is how high the player can fly: far enough to look down on the world.
	ceiling = world.SizeY + 16
)

// Player is the player's body: a box that stands on blocks and is stopped by them.
type Player struct {
	// Pos is the middle of the bottom of the box: the point between the feet.
	Pos geometry.Vector3

	// Yaw and Pitch are where the player looks, as they are for a glonk Camera.
	Yaw, Pitch float64

	// Flying turns gravity off: the player rises and sinks only when asked.
	Flying bool

	// OnGround is true while the player stands on a block.
	OnGround bool

	velY float64 // speed upward
}

// controls is what the player asks for in one frame.
type controls struct {
	forward, strafe float64 // 1 is ahead and to the right, -1 back and to the left
	turn, tilt      float64 // 1 is to the right and up
	rise, sink      bool    // jump, swim up or fly up; fly down
	toggleFly       bool
}

// keyDown reports whether a letter key counts as down or pressed, in either case:
// terminals that cannot report keys exactly send a capital for Shift or Caps Lock.
// held is Input.IsDown or Input.Pressed.
func keyDown(held func(engine.Key) bool, letter rune) bool {
	return held(engine.Key(letter)) || held(engine.Key(unicode.ToUpper(letter)))
}

// readControls turns the keys into what the player asks for. F is a switch, so it is
// read with Started: Pressed is true for every repeat of a held key, and would flip
// flying many times a second.
func readControls(in *engine.Input) controls {
	axis := func(plus, minus bool) float64 {
		switch {
		case plus && !minus:
			return 1
		case minus && !plus:
			return -1
		}
		return 0
	}
	return controls{
		forward:   axis(keyDown(in.IsDown, 'w'), keyDown(in.IsDown, 's')),
		strafe:    axis(keyDown(in.IsDown, 'd'), keyDown(in.IsDown, 'a')),
		turn:      axis(in.IsDown(engine.KeyRight), in.IsDown(engine.KeyLeft)),
		tilt:      axis(in.IsDown(engine.KeyUp), in.IsDown(engine.KeyDown)),
		rise:      in.IsDown(engine.Key(' ')),
		sink:      keyDown(in.IsDown, 'x'),
		toggleFly: keyDown(in.Started, 'f'),
	}
}

// Camera returns the view from the player's eyes.
func (p *Player) Camera() mesh.Camera {
	return mesh.Camera{
		Position: geometry.Vector3{X: p.Pos.X, Y: p.Pos.Y + eyeHeight, Z: p.Pos.Z},
		Yaw:      p.Yaw,
		Pitch:    p.Pitch,
		FOV:      fieldOfView,
	}
}

// solid reports whether a cell stops the player. The sides and the bottom of the world
// are walls; above it there is open air.
func solid(w *world.World, x, y, z int) bool {
	if x < 0 || x >= world.SizeX || z < 0 || z >= world.SizeZ || y < 0 {
		return true
	}
	return w.At(x, y, z).Info().Solid
}

// box returns the low and high corner of the player's box.
func (p *Player) box() (lo, hi [3]float64) {
	half := playerWidth / 2.0
	return [3]float64{p.Pos.X - half, p.Pos.Y, p.Pos.Z - half},
		[3]float64{p.Pos.X + half, p.Pos.Y + playerHeight, p.Pos.Z + half}
}

// overlaps reports whether the player's box is in any part of a cell.
func (p *Player) overlaps(x, y, z int) bool {
	lo, hi := p.box()
	cell := [3]float64{float64(x), float64(y), float64(z)}
	for a := range 3 {
		if hi[a] <= cell[a] || lo[a] >= cell[a]+1 {
			return false
		}
	}
	return true
}

// swimming reports whether the player's feet are in water.
func (p *Player) swimming(w *world.World) bool {
	return w.At(int(math.Floor(p.Pos.X)), int(math.Floor(p.Pos.Y+0.1)), int(math.Floor(p.Pos.Z))) == world.Water
}

// Step moves the player for dt seconds: turns the view, then walks, falls, swims or
// flies, stopping at solid blocks.
func (p *Player) Step(w *world.World, c controls, dt float64) {
	p.Yaw = math.Remainder(p.Yaw+c.turn*turnSpeed*dt, 2*math.Pi)
	p.Pitch = min(max(p.Pitch+c.tilt*tiltSpeed*dt, -maxPitch), maxPitch)
	if c.toggleFly {
		p.Flying = !p.Flying
		p.velY = 0
	}

	speed := walkSpeed
	switch {
	case p.Flying:
		speed = flySpeed
		p.velY = 0
		if c.rise {
			p.velY += flySpeed
		}
		if c.sink {
			p.velY -= flySpeed
		}
	case p.swimming(w):
		speed = swimSpeed
		p.velY = max(p.velY-gravity/4*dt, -sinkSpeed)
		if c.rise {
			p.velY = swimUp
		}
	default:
		p.velY = max(p.velY-gravity*dt, -fallSpeed)
		if c.rise && p.OnGround {
			p.velY = jumpSpeed
		}
	}

	// Normalized so that walking forward and sideways at once is no faster.
	facing := mesh.Camera{Yaw: p.Yaw}
	move := facing.Forward().Scale(c.forward).Add(facing.Right().Scale(c.strafe)).Normalize().Scale(speed * dt)
	move.Y = p.velY * dt

	p.OnGround = false
	steps := max(1, int(math.Ceil(max(math.Abs(move.X), math.Abs(move.Y), math.Abs(move.Z))/maxStep)))
	part := move.Scale(1 / float64(steps))
	for range steps {
		p.slide(w, 0, part.X)
		p.slide(w, 2, part.Z)
		if p.slide(w, 1, part.Y) {
			p.OnGround = part.Y < 0
			p.velY = 0
		}
	}
	p.Pos.Y = min(p.Pos.Y, ceiling)
}

// slide moves the player along one axis (0 is x, 1 is y, 2 is z) and, if that takes its
// box into a solid block, puts it back against the block's face. It reports whether the
// player was stopped. Moving one axis at a time is what lets the player slide along a
// wall it walks into at an angle.
func (p *Player) slide(w *world.World, axis int, delta float64) bool {
	if delta == 0 {
		return false
	}
	coord := [3]*float64{&p.Pos.X, &p.Pos.Y, &p.Pos.Z}[axis]
	*coord += delta

	lo, hi := p.box()
	var from, to [3]int
	for a := range 3 {
		from[a], to[a] = int(math.Floor(lo[a])), int(math.Ceil(hi[a]))-1
	}

	// The face that stops the player is the nearest one it has gone past.
	stopped := false
	face := math.Inf(1)
	if delta < 0 {
		face = math.Inf(-1)
	}
	for y := from[1]; y <= to[1]; y++ {
		for z := from[2]; z <= to[2]; z++ {
			for x := from[0]; x <= to[0]; x++ {
				if !solid(w, x, y, z) {
					continue
				}
				stopped = true
				cell := float64([3]int{x, y, z}[axis])
				if delta > 0 {
					face = min(face, cell)
				} else {
					face = max(face, cell+1)
				}
			}
		}
	}
	if !stopped {
		return false
	}
	if delta > 0 {
		*coord = face - (hi[axis] - *coord) - gap
	} else {
		*coord = face + (*coord - lo[axis]) + gap
	}
	return true
}

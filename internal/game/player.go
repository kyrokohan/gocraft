package game

import (
	"math"
	"unicode"

	"glonk.dev/engine"
	"glonk.dev/geometry"
	"glonk.dev/mesh"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/world"
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

	// ceiling is how high the player can fly: far enough to look down on the world.
	ceiling = world.Height + 16
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

// Camera returns the view from the player's eyes, as deep as the world is drawn, so that
// what glonk draws through it ends where the blocks do. It has no fog: that is the sky's.
func (p *Player) Camera() mesh.Camera {
	return mesh.Camera{
		Position: geometry.Vector3{X: p.Pos.X, Y: p.Pos.Y + eyeHeight, Z: p.Pos.Z},
		Yaw:      p.Yaw,
		Pitch:    p.Pitch,
		FOV:      defaultSettings.fov(),
		Far:      render.ViewDistance,
	}
}

// overlaps reports whether the player's box is in any part of a cell.
func (p *Player) overlaps(x, y, z int) bool {
	lo, hi := boxOf(p.Pos, playerWidth, playerHeight)
	cell := [3]float64{float64(x), float64(y), float64(z)}
	for a := range 3 {
		if hi[a] <= cell[a] || lo[a] >= cell[a]+1 {
			return false
		}
	}
	return true
}

// Turn turns the view to the right by yaw and up by pitch, in radians. It cannot be
// tipped past straight up or straight down.
func (p *Player) Turn(yaw, pitch float64) {
	p.Yaw = math.Remainder(p.Yaw+yaw, 2*math.Pi)
	p.Pitch = min(max(p.Pitch+pitch, -maxPitch), maxPitch)
}

// Step moves the player for dt seconds: turns the view, then walks, falls, swims or
// flies, stopping at solid blocks.
func (p *Player) Step(w *world.World, c controls, dt float64) {
	p.Turn(c.turn*turnSpeed*dt, c.tilt*tiltSpeed*dt)
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
	case inWater(w, p.Pos):
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

	stopped := travel(w, &p.Pos, playerWidth, playerHeight, move)
	p.OnGround = stopped[1] && move.Y < 0
	if stopped[1] {
		p.velY = 0
	}
	p.Pos.Y = min(p.Pos.Y, ceiling)
}

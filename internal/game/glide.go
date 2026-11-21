package game

import (
	"math"

	"glonk.dev/geometry"
	"glonk.dev/mesh"

	"github.com/kyrokohan/skyspear/internal/world"
)

// A galewing of the player's, grown, takes the player when it steps off a height, and
// carries it down in a glide.
const (
	glideHigh  = 5.5  // how far over what is under it a falling player must be: six blocks, less the hair it has fallen when it is seen
	glideReach = 32.0 // how near the galewing must be, along the ground
	glideSpeed = 9.0  // blocks a second ahead
	glideSink  = 0.2  // blocks down for each block ahead, and diveSink with W held
	diveSink   = 0.5
	keepOver   = 20.0 // how far from the player a galewing of its own comes straight back
)

// glider returns the beast that carries the player, or nil.
func (f *fauna) glider() *beast {
	for i := range f.all {
		if f.all[i].carrying {
			return &f.all[i]
		}
	}
	return nil
}

// takeOff has a grown galewing of the player's that is in the air near it take a player
// that falls from a height, under open sky.
func (g *Game) takeOff() {
	p := &g.player
	if p.OnGround || p.Flying || inWater(g.world, p.Pos) {
		g.dropped = false
	}
	if g.dropped || p.OnGround || p.Flying || p.velY >= 0 || inWater(g.world, p.Pos) {
		return
	}
	x, z := int(math.Floor(p.Pos.X)), int(math.Floor(p.Pos.Z))
	if g.world.Surface(x, z) >= int(math.Floor(p.Pos.Y)) {
		return // under a roof
	}
	for y := int(math.Floor(p.Pos.Y)); y >= 0; y-- {
		if solid(g.world, x, y, z) || g.world.At(x, y, z) == world.Water {
			if p.Pos.Y-float64(y+1) < glideHigh {
				return
			}
			break
		}
	}
	for i := range g.fauna.all {
		c := &g.fauna.all[i]
		if breeds[c.kind].glides && c.tame && !c.young && !c.limping && !c.dead && c.aloft &&
			math.Hypot(c.Pos.X-p.Pos.X, c.Pos.Z-p.Pos.Z) < glideReach {
			c.carrying = true
			g.hang()
			g.tell("the galewing has you: A and D turn, W dives, C lets go")
			return
		}
	}
}

// glide moves a player that a galewing carries for dt seconds: ahead the way it faces,
// and down a block for every five, or two with W held. It cannot climb. On the ground or
// in water the galewing lets it go, unhurt, and C lets go of it where it is.
func (g *Game) glide(m *beast, c controls, dt float64) {
	p := &g.player
	p.Turn((c.turn+c.strafe)*turnSpeed*dt, c.tilt*tiltSpeed*dt)
	if c.toggleCrouch {
		m.carrying, g.dropped = false, true // it falls from here, and is not taken again until it is down
		return
	}
	sink := glideSink
	if c.forward > 0 {
		sink = diveSink
	}
	move := mesh.Camera{Yaw: p.Yaw}.Forward().Scale(glideSpeed * dt)
	move.Y = -sink * glideSpeed * dt
	stopped := travel(g.world, &p.Pos, playerWidth, playerHeight, move)
	p.velY, p.OnGround, p.Crouched = move.Y/dt, stopped[1], false
	g.peak = p.Pos.Y // a glide is no fall
	if stopped[1] || inWater(g.world, p.Pos) {
		m.carrying, p.velY = false, 0
	}
}

// hang puts the galewing that carries the player over its head, its feet on it.
func (g *Game) hang() {
	if m := g.fauna.glider(); m != nil {
		m.Pos = g.player.Pos.Add(geometry.Vector3{Y: playerHeight})
		m.Yaw, m.aloft, m.walking, m.OnGround = g.player.Yaw, true, true, false
	}
}

package game

import (
	"math"

	"glonk.dev/geometry"
	"glonk.dev/mesh"

	"github.com/kyrokohan/skyspear/internal/render"
)

// About riding.
const (
	gather   = 1.5  // how many seconds a beast ridden with W held takes to go from a walk to its run
	backPace = 0.5  // how much of a walk it backs at
	setGap   = 0.05 // how far from its mount a rider is set down

	ramTime  = 2.0  // how long the charge of a beast that rams lasts, at the most
	ramSpeed = 8.0  // how fast it charges
	ramRest  = 4.0  // from the end of a charge to the next
	ramKeep  = 10.0 // how far off its rider a hunter its charge would kill keeps
)

// mount returns the beast the player rides, or nil.
func (f *fauna) mount() *beast {
	for i := range f.all {
		if f.all[i].ridden {
			return &f.all[i]
		}
	}
	return nil
}

// getOn has the player get on the beast aimed at, if it is of a kind that can be
// ridden and is the player's. It reports whether E was taken up by that: a young one is
// too small, and one that limps is going home.
func (g *Game) getOn(p prey) bool {
	c := &g.fauna.all[p.i]
	switch {
	case breeds[c.kind].ride == 0 || !c.tame || c.dead:
		return false
	case c.young:
		g.tell("the young " + c.kind.String() + " is too small to ride")
		return true
	case c.limping:
		g.tell("the " + c.kind.String() + " is hurt: it goes home to heal")
		return true
	}
	c.ridden, g.urge = true, 0
	g.seat()
	g.tell("you ride the " + c.kind.String() + ": C gets you down")
	return true
}

// ride turns the player by the keys, and has the beast it rides go as they say, the way
// the player faces: W on at a walk, the player's or its own if that is slower, and on to
// its run while it is held; S back; A and D turn, as the arrows do. Space jumps, or sets
// off a charge. C gets the player down.
func (g *Game) ride(m *beast, c controls, dt float64) {
	p := &g.player
	p.Turn((c.turn+c.strafe)*turnSpeed*dt, c.tilt*tiltSpeed*dt)
	if c.toggleCrouch {
		g.getDown(m)
		return
	}
	b := &breeds[m.kind]
	walk := min(walkSpeed, b.ride)
	switch {
	case c.forward > 0:
		g.urge = min(g.urge+dt/gather, 1)
		m.reined = walk + (b.ride-walk)*g.urge
	case c.forward < 0:
		g.urge, m.reined = 0, -backPace*walk
	default:
		g.urge, m.reined = 0, 0
	}
	m.Yaw, m.walking, m.jumping = p.Yaw, m.reined != 0, c.rise && b.clears > 0
	if c.rise && b.rams > 0 && m.ire == 0 && m.rest == 0 {
		m.ire = ramTime
	}
}

// seat puts the player on the back of the beast it rides. One that has lost all its
// health throws the player off.
func (g *Game) seat() {
	m := g.fauna.mount()
	switch {
	case m == nil:
		return
	case m.limping || m.dead:
		if at, ok := g.besideMount(m); ok {
			g.setDown(m, at)
		} else {
			g.setDown(m, g.player.Pos) // nowhere to stand: the player falls where it was
		}
		return
	}
	b := &breeds[m.kind]
	p := &g.player
	p.Pos = m.Pos.Add(mesh.Camera{Yaw: m.Yaw}.Forward().Scale(b.seat))
	p.Pos.Y = m.Pos.Y + b.back - render.SeatHeight
	p.velY, p.OnGround, p.Crouched, p.Flying = 0, false, false, false
	g.peak = p.Pos.Y // a fall is the beast's, and not the rider's
}

// getDown sets the player down off the beast it rides, where there is room beside it. With
// none anywhere the player stays on, and is told.
func (g *Game) getDown(m *beast) {
	at, ok := g.besideMount(m)
	if !ok {
		g.tell("no room to get down here")
		return
	}
	g.setDown(m, at)
}

// setDown leaves the player standing at a place, and the beast it rode its own again.
func (g *Game) setDown(m *beast, at geometry.Vector3) {
	m.ridden, m.reined, m.jumping, m.walking = false, 0, false, false
	g.player.Pos, g.player.velY = at, 0
	g.peak = at.Y
}

// besideMount finds where the player can stand by the beast it rides: to its left, to its
// right, behind it or ahead of it, at the height of its feet or a block up. It reports
// whether there is such a place.
func (g *Game) besideMount(m *beast) (geometry.Vector3, bool) {
	width, _ := render.Size(m.kind, m.young)
	back, front := render.Span(m.kind, m.young)
	turned := mesh.Camera{Yaw: m.Yaw}
	ahead, right := turned.Forward(), turned.Right()
	side, end := width/2+playerWidth/2+setGap, playerWidth/2+setGap
	for _, way := range [4]geometry.Vector3{right.Scale(-side), right.Scale(side), ahead.Scale(back - end), ahead.Scale(front + end)} {
		for up := range 2 {
			at := m.Pos.Add(way).Add(geometry.Vector3{Y: float64(up)})
			if g.standsIn(at) {
				return at, true
			}
		}
	}
	return geometry.Vector3{}, false
}

// standsIn reports whether the player's box at a place is clear of solid blocks.
func (g *Game) standsIn(at geometry.Vector3) bool {
	lo, hi := boxOf(at, playerWidth, playerHeight)
	for x := int(math.Floor(lo[0])); x < int(math.Ceil(hi[0])); x++ {
		for y := int(math.Floor(lo[1])); y < int(math.Ceil(hi[1])); y++ {
			for z := int(math.Floor(lo[2])); z < int(math.Ceil(hi[2])); z++ {
				if solid(g.world, x, y, z) {
					return false
				}
			}
		}
	}
	return true
}

package game

import (
	"math"

	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/sound"
	"github.com/kyrokohan/skyspear/internal/world"
)

// Things that fly. R throws the spear in hand, and with a sling in hand a pebble of
// those carried. What is thrown goes the way the player aims, falls as it goes, and
// stops at the first block or beast it meets: a beast is hurt by it, and a herd it
// lands near takes alarm. Then it lies there, or sticks there if it is a spear, to be
// picked up again. A missile is one kind of moving thing, with a row of the table for
// each kind of flight: what a fire mountain throws out and what falls from the sky are
// to be more rows of it.
const (
	thrownMore  = 2.0  // points a thrown spear takes from a beast over what a stab with it takes
	slingWait   = 0.45 // seconds from one shot of a sling to the next
	missileStep = 0.25 // the farthest a thing in flight goes before it is asked again what it met, in blocks
	flightLife  = 12.0 // seconds a thing can fly before it is lost: it left the world that is loaded
	spearLong   = 1.1  // how far across the picture of a spear in flight is, in blocks
	pebbleSize  = 0.14 // how far across a pebble in flight is
)

// flier is a kind of thing that flies.
type flier uint8

const (
	spearThrown flier = iota
	pebbleSlung
)

// flight is how a kind of thing flies, and what it does where it comes down.
type flight struct {
	speed  float64 // blocks a second, as it leaves the hand
	weight float64 // how much faster it falls each second, in blocks a second
	harm   float64 // points it takes from a beast it meets, over what a hit with the thing takes
	scare  float64 // how near a herd it lands for the herd to take alarm, in blocks
	sticks bool    // it is long, flies point first, and stays in what it meets
}

var fliers = [...]flight{
	spearThrown: {speed: 24, weight: 16, harm: thrownMore, scare: 4, sticks: true},
	pebbleSlung: {speed: 38, weight: 14, scare: 8},
}

// missile is a thing in flight, or a spear that sticks where it came down.
type missile struct {
	kind     flier
	stack    world.Stack      // the thing it is
	pos, vel geometry.Vector3 // its middle, and how it moves: kept, once it has landed, for the way it points
	leaves   bool             // it is left where it comes down, to be picked up: not in creative, where nothing is used up
	landed   bool             // it sticks in a block
	stuck    world.Cell       // the block it sticks in
	age      float64          // seconds it has flown, or since it landed
}

// throw throws what is in hand, if that is thrown, the way the player aims: a spear
// goes itself, and a sling sends a pebble of those carried. It reports whether anything
// flew.
func (g *Game) throw() bool {
	held := &g.items[g.selected]
	m := missile{leaves: g.survival}
	switch held.Item {
	case world.Spear, world.HardSpear:
		m.kind, m.stack = spearThrown, world.Stack{Item: held.Item, Count: 1, Wear: held.Wear}
		g.hitWait = hitWait
		if g.survival {
			m.stack.Wear++ // a throw is a use
			*held = world.Stack{}
		}
	case world.Sling:
		pebbles := []world.Item{world.Pebble}
		g.hitWait = slingWait
		if g.survival && g.items.count(pebbles) == 0 {
			g.tell("no pebbles to sling: gravel makes them")
			return false
		}
		m.kind, m.stack = pebbleSlung, world.Stack{Item: world.Pebble, Count: 1}
		if g.survival {
			g.items.use(pebbles, 1)
			g.wearOut()
		}
	default:
		return false
	}
	m.pos = g.from.Add(g.along.Scale(0.4))
	m.vel = g.along.Scale(fliers[m.kind].speed)
	g.missiles = append(g.missiles, m)
	g.swing = swingTime
	g.ears.PlayNear(sound.Dig(sound.Plant), 0.6)
	return true
}

// stepMissiles moves what was thrown on by dt seconds: what is in flight flies, and a
// spear that sticks waits to be picked up as a thing on the ground is, falls when what
// it sticks in is gone, and is gone itself when old.
func (g *Game) stepMissiles(dt float64) {
	middle := g.player.Pos.Add(geometry.Vector3{Y: playerHeight / 2})
	kept := g.missiles[:0]
	for _, m := range g.missiles {
		switch m.age += dt; {
		case !m.landed:
			if !g.fly(&m, dt) {
				continue
			}
		case m.age > dropLife:
			continue
		case !g.world.At(m.stuck.X, m.stuck.Y, m.stuck.Z).Info().Solid:
			g.lay(m) // what it stuck in was broken
			continue
		case m.age >= pickWait && m.pos.Sub(middle).Length() <= pickReach && g.items.add(m.stack) == 0:
			g.ears.PlayNear(sound.Pickup, 0.6)
			continue
		}
		kept = append(kept, m)
	}
	g.missiles = kept
}

// fly moves a thing in flight on by dt seconds, a short way at a time, and reports
// whether it is still to be kept: in flight, or stuck where it landed.
func (g *Game) fly(m *missile, dt float64) bool {
	f := &fliers[m.kind]
	if m.age > flightLife || m.pos.Y < -missileStep {
		return false // it is lost: it left the world that is loaded, or fell out of the bottom of it
	}
	for left := dt; left > 0; {
		step := min(left, missileStep/max(m.vel.Length(), 1))
		left -= step
		m.vel.Y -= f.weight * step
		next := m.pos.Add(m.vel.Scale(step))
		if i, ok := g.beastAt(next); ok {
			g.wound(i, m.stack.Item.Damage()+f.harm, 1, false)
			g.lay(*m)
			return false
		}
		if !g.solidAt(next) {
			m.pos = next
			continue
		}

		// It has met a block.
		in := world.Cell{X: int(math.Floor(next.X)), Y: int(math.Floor(next.Y)), Z: int(math.Floor(next.Z))}
		g.ears.Play(sound.Place(materialOf(g.world.At(in.X, in.Y, in.Z))), m.pos, 0.8)
		for i := range g.fauna.all {
			if c := &g.fauna.all[i]; !c.dead && c.Pos.Distance(m.pos) <= f.scare {
				g.fauna.raise(c.herd, m.pos)
			}
		}
		if !f.sticks || !m.leaves || worn(m.stack) {
			g.lay(*m)
			return false
		}
		m.landed, m.stuck, m.age, m.vel = true, in, 0, m.vel.Normalize()
		return true
	}
	return true
}

// worn reports whether a thing that wears has no use left in it.
func worn(s world.Stack) bool {
	return s.Item.Uses() > 0 && s.Wear >= s.Item.Uses()
}

// lay puts a thing that was thrown down where it is, to fall and be picked up, if it is
// one that is left and it has not broken.
func (g *Game) lay(m missile) {
	switch {
	case !m.leaves:
	case worn(m.stack):
		g.ears.Play(sound.Break(sound.Wood), m.pos, 0.8)
	default:
		g.drops = append(g.drops, drop{Drop: world.Drop{Stack: m.stack, Pos: m.pos}})
	}
}

// solidAt reports whether the block at a point stops a thing in flight.
func (g *Game) solidAt(p geometry.Vector3) bool {
	return g.world.At(int(math.Floor(p.X)), int(math.Floor(p.Y)), int(math.Floor(p.Z))).Info().Solid
}

// beastAt returns the living beast whose body a point is in, if it is in one.
func (g *Game) beastAt(p geometry.Vector3) (int, bool) {
	for i := range g.fauna.all {
		c := &g.fauna.all[i]
		if c.dead {
			continue
		}
		width, height := render.Size(c.kind, c.young)
		lo, hi := boxOf(c.Pos, width*aimWider, height)
		if p.X >= lo[0] && p.X <= hi[0] && p.Y >= lo[1] && p.Y <= hi[1] && p.Z >= lo[2] && p.Z <= hi[2] {
			return i, true
		}
	}
	return 0, false
}

// flying returns what was thrown as it is drawn: lit by the cell it is in.
func (g *Game) flying() []render.Flying {
	g.thrown = g.thrown[:0]
	for _, m := range g.missiles {
		size := pebbleSize
		if fliers[m.kind].sticks {
			size = spearLong
		}
		g.thrown = append(g.thrown, render.Flying{
			Item: m.stack.Item, Pos: m.pos, Along: m.vel, Long: fliers[m.kind].sticks, Size: size,
			Light: g.sky.Brightness(g.lightAt(m.pos)),
		})
	}
	return g.thrown
}

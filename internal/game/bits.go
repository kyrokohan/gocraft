package game

import (
	"math"

	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/world"
)

// The bits that fly from a block as it breaks.
const (
	bitsABreak = 10   // how many fly from one block
	maxBits    = 192  // the most there are at once: the oldest go first past that
	bitSize    = 0.14 // how far across one is, in blocks
	bitFall    = 20.0 // how fast they fall faster, in blocks a second, each second
	bitLife    = 0.7  // seconds the longest lived one lasts
)

// Smoke is bits too: grey ones that rise from every fire near the player, and grow as
// they go.
const (
	smokeEvery = 0.4  // seconds from one puff of a fire to its next
	smokeReach = 96.0 // how far from the player a fire still smokes, in blocks
	smokeRise  = 1.5  // how fast a puff rises, in blocks a second
	smokeLife  = 6.0  // seconds a puff of a great fire lasts: so its smoke stands nine blocks high

	craterPuffs = 2   // how many puffs a fire mountain lets go at a time
	craterSmoke = 5.0 // how far from the middle of its crater they rise, in blocks
)

// bit is a piece of a broken block, flying and then lying where it fell, till it is gone,
// or a puff of smoke.
type bit struct {
	pos, vel  geometry.Vector3
	item      world.Item
	age, life float64
	spin      float64 // how fast it tumbles, in radians a second
	smoke     bool    // it is a puff of smoke, which rises and does not fall
	size      float64 // how far across a puff is when it is new: it grows to twice that
	glows     bool    // it is fire, and is its own light
}

// shatter throws bits of a block out from its cell: up and away from its middle. A plant,
// a torch or a mat, which is mostly holes, leaves none.
func (g *Game) shatter(x, y, z int, b world.Block) {
	if b.Info().Stands() {
		return
	}
	r := g.scatter
	mid := cellMid(x, y, z)
	for range bitsABreak {
		off := geometry.Vector3{X: r.Float64() - 0.5, Y: r.Float64() - 0.5, Z: r.Float64() - 0.5}.Scale(0.7)
		g.bits = append(g.bits, bit{
			pos:  mid.Add(off),
			vel:  geometry.Vector3{X: off.X * 6, Y: 2 + 3*r.Float64(), Z: off.Z * 6},
			item: b.Item(),
			life: bitLife * (0.6 + 0.4*r.Float64()),
			spin: 6 * (2*r.Float64() - 1),
		})
	}
	if over := len(g.bits) - maxBits; over > 0 {
		g.bits = g.bits[over:]
	}
}

// puff lets a puff of smoke go from a place: size across to begin with, to last about
// life seconds.
func (g *Game) puff(at geometry.Vector3, size, life float64) {
	g.puffOf(smokeItem, at, size, life, smokeRise)
}

// flame lets a puff of fire go from a place, as puff does one of smoke: it rises at a
// speed, in blocks a second, and is its own light.
func (g *Game) flame(at geometry.Vector3, size, life, rise float64) {
	g.puffOf(world.Lava.Item(), at, size, life, rise)
	g.bits[len(g.bits)-1].glows = true
}

// puffOf lets a puff of something that rises go from a place, drawn as a piece of an
// item: smoke, or the steam of a geyser. It rises at a speed, in blocks a second.
func (g *Game) puffOf(item world.Item, at geometry.Vector3, size, life, rise float64) {
	r := g.scatter
	g.bits = append(g.bits, bit{
		pos:   at.Add(geometry.Vector3{X: 0.3 * (r.Float64() - 0.5), Z: 0.3 * (r.Float64() - 0.5)}),
		vel:   geometry.Vector3{X: 0.3 * (r.Float64() - 0.5), Y: rise * (0.8 + 0.4*r.Float64()), Z: 0.3 * (r.Float64() - 0.5)},
		item:  item,
		life:  life * (0.7 + 0.3*r.Float64()),
		spin:  r.Float64() - 0.5,
		smoke: true,
		size:  size,
	})
	if over := len(g.bits) - maxBits; over > 0 {
		g.bits = g.bits[over:]
	}
}

// smokeItem is what a puff of smoke is drawn as a piece of: grey, and of no one thing.
var smokeItem = world.Gravel.Item()

// smoke lets the fires near the player smoke, dt seconds on: the trees that lightning
// struck, whose smoke stands high and is seen from far off, and the hearths that burn.
func (g *Game) smoke(dt float64) {
	if g.smoking -= dt; g.smoking > 0 {
		return
	}
	g.smoking = smokeEvery
	p := g.player.Pos
	for _, c := range g.world.Struck(p.X, p.Z, smokeReach) {
		if g.world.At(c.X, c.Y, c.Z) == world.Fire {
			g.puff(cellMid(c.X, c.Y+1, c.Z), 0.5, smokeLife)
		}
	}
	for c, k := range g.hearths {
		if mid := cellMid(c.X, c.Y+1, c.Z); k.Burn > 0 && mid.Distance(p) < smokeReach {
			g.puff(mid, 0.2, smokeLife/2)
		}
	}
	// A fire mountain smokes from its crater: great puffs, from all over its lake.
	if crater, far, ok := g.world.Mountain(p.X, p.Z); ok && far < smokeReach {
		puffs := craterPuffs
		if _, fire := g.ash(); fire > 0 {
			puffs *= 2 // it erupts
		}
		for range puffs {
			at := cellMid(crater.X, crater.Y, crater.Z).Add(geometry.Vector3{
				X: craterSmoke * (2*g.scatter.Float64() - 1), Z: craterSmoke * (2*g.scatter.Float64() - 1),
			})
			g.puff(at, 1.1, 1.5*smokeLife)
		}
	}
}

// stepBits moves the bits on by dt seconds. One that meets a solid block stops there,
// resting on it, and each is gone when its time is up.
func (g *Game) stepBits(dt float64) {
	kept := g.bits[:0]
	for _, b := range g.bits {
		if b.age += dt; b.age >= b.life {
			continue
		}
		if !b.smoke {
			b.vel.Y -= bitFall * dt
		}
		next := b.pos.Add(b.vel.Scale(dt))
		switch solid := g.world.At(int(math.Floor(next.X)), int(math.Floor(next.Y-bitSize/2)), int(math.Floor(next.Z))).Info().Solid; {
		case solid && b.smoke:
			continue // smoke that meets a roof is gone
		case solid:
			b.vel, b.spin = geometry.Vector3{}, 0
		default:
			b.pos = next
		}
		kept = append(kept, b)
	}
	g.bits = kept
}

// drawnBits returns the bits as they are drawn: each shrinks away in the last part of
// its time, and is lit by the cell it is in.
func (g *Game) drawnBits() []render.Bit {
	g.pieces = g.pieces[:0]
	for _, b := range g.bits {
		left := (b.life - b.age) / b.life
		size := bitSize
		if b.smoke {
			size = b.size * (2 - left)
		}
		light := 1.0
		if !b.glows {
			light = g.sky.Brightness(g.lightAt(b.pos))
		}
		g.pieces = append(g.pieces, render.Bit{
			Item: b.item, Pos: b.pos, Size: size * min(1, left/0.4), Spin: b.spin * b.age, Light: light,
		})
	}
	return g.pieces
}

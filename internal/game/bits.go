package game

import (
	"math"

	"glonk.dev/geometry"

	"github.com/kyrokohan/gocraft/internal/render"
	"github.com/kyrokohan/gocraft/internal/world"
)

// The bits that fly from a block as it breaks.
const (
	bitsABreak = 10   // how many fly from one block
	maxBits    = 96   // the most there are at once: the oldest go first past that
	bitSize    = 0.14 // how far across one is, in blocks
	bitFall    = 20.0 // how fast they fall faster, in blocks a second, each second
	bitLife    = 0.7  // seconds the longest lived one lasts
)

// bit is a piece of a broken block, flying and then lying where it fell, till it is gone.
type bit struct {
	pos, vel  geometry.Vector3
	item      world.Item
	age, life float64
	spin      float64 // how fast it tumbles, in radians a second
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

// stepBits moves the bits on by dt seconds. One that meets a solid block stops there,
// resting on it, and each is gone when its time is up.
func (g *Game) stepBits(dt float64) {
	kept := g.bits[:0]
	for _, b := range g.bits {
		if b.age += dt; b.age >= b.life {
			continue
		}
		b.vel.Y -= bitFall * dt
		next := b.pos.Add(b.vel.Scale(dt))
		if g.world.At(int(math.Floor(next.X)), int(math.Floor(next.Y-bitSize/2)), int(math.Floor(next.Z))).Info().Solid {
			b.vel, b.spin = geometry.Vector3{}, 0
		} else {
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
		g.pieces = append(g.pieces, render.Bit{
			Item:  b.item,
			Pos:   b.pos,
			Size:  bitSize * min(1, left/0.4),
			Spin:  b.spin * b.age,
			Light: g.sky.Brightness(g.lightAt(b.pos)),
		})
	}
	return g.pieces
}

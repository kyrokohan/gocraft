package game

import (
	"math"

	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/sound"
	"github.com/kyrokohan/skyspear/internal/world"
)

// The player hits a creature it aims at with what breaks a block, Q or a click, and
// shears a mossback's moss with what places one, E or a right click. A creature in front
// of a block is aimed at before the block.

// hitWait is the least time between two hits, however fast the key repeats.
const hitWait = 0.35

// aimWider is how much wider than the box blocks stop is the box a ray through the view
// meets a beast in, so that the head and the legs are aimed at too.
const aimWider = 1.375

// mossShorn is how much moss a mossback gives.
const mossShorn = 2

// prey is a beast the player aims at, by its place in the list of them. It holds for
// the frame it was found in.
type prey struct {
	i int
}

// aimCreature finds the nearest creature a ray from origin along dir, one long, meets
// before limit.
func (g *Game) aimCreature(origin, dir geometry.Vector3, limit float64) (p prey, ok bool) {
	best := limit
	for i := range g.fauna.all {
		c := &g.fauna.all[i]
		if c.dead {
			continue // it cannot be hit again as it falls
		}
		width, height := render.Size(c.kind, c.young)
		boxes, n := c.body()
		for _, at := range boxes[:n] {
			if t, hit := rayBox(origin, dir, at, width*aimWider, height); hit && t < best {
				best, p, ok = t, prey{i: i}, true
			}
		}
	}
	return p, ok
}

// rayBox returns how far along a ray from origin along dir it meets a box standing on
// the point pos, if it does in front of the origin.
func rayBox(origin, dir, pos geometry.Vector3, width, height float64) (float64, bool) {
	lo, hi := boxOf(pos, width, height)
	o, d := [3]float64{origin.X, origin.Y, origin.Z}, [3]float64{dir.X, dir.Y, dir.Z}
	near, far := 0.0, math.Inf(1)
	for a := range 3 {
		if d[a] == 0 {
			if o[a] < lo[a] || o[a] > hi[a] {
				return 0, false
			}
			continue
		}
		t0, t1 := (lo[a]-o[a])/d[a], (hi[a]-o[a])/d[a]
		if t0 > t1 {
			t0, t1 = t1, t0
		}
		near, far = max(near, t0), min(far, t1)
		if near > far {
			return 0, false
		}
	}
	return near, true
}

// strike hits the creature aimed at with what is in hand, which wears a tool.
func (g *Game) strike(p prey) {
	g.hitWait, g.swing = hitWait, swingTime
	held := g.items[g.selected].Item
	g.wound(p.i, held.Damage(), held.Knock(), held.Carves())
	if g.survival {
		g.wearOut()
	}
}

// wound is a beast hit by the player, with what is in hand or with a thing thrown: it
// takes points of harm, is knocked back, as many times as far as a hand knocks it, and
// cries out. A herd runs with it, and a hunter only staggers, unless it is badly hurt
// and alone. A kill that is carved leaves more.
func (g *Game) wound(i int, points, knock float64, carves bool) {
	from := g.player.Pos
	c := &g.fauna.all[i]
	if c.hit(from, points) {
		c.carved = carves
	}
	c.push = c.push.Scale(knock) // a club sends it farther
	g.voice(c, breeds[c.kind].cry, 1)
	if b := &breeds[c.kind]; b.temper == hunter {
		// A hunter does not run from a hit, unless it is badly hurt and alone.
		if c.flee = staggerFor; c.health < hurtPart*b.health && !g.fauna.packNear(i, aloneReach) {
			c.flee = routTime
		}
	} else {
		g.fauna.raise(c.herd, from) // its herd runs with it
	}
}

// feed gives the beast aimed at what is in the player's hand, if it will take it. It
// reports whether it did.
func (g *Game) feed(p prey) bool {
	c := &g.fauna.all[p.i]
	ate, tamed := c.feed(g.items[g.selected].Item)
	if !ate {
		return false
	}
	g.ears.Play(sound.Eat, c.voice(), 0.8)
	g.swing = swingTime
	if g.survival {
		g.items.takeOne(g.selected)
	}
	if tamed {
		g.tell("the " + c.kind.String() + " trusts you: it will follow you")
	} else {
		g.tell("the " + c.kind.String() + " eats from your hand")
	}
	return true
}

// shear takes the moss of the mossback aimed at, in survival, if it has grown, and puts
// it down on the mossback's back to be picked up. It reports whether it did. A hand
// that holds food shears nothing: that is for eating, or for feeding.
func (g *Game) shear(p prey) bool {
	if !g.survival || g.items[g.selected].Item.Food() > 0 {
		return false
	}
	c := &g.fauna.all[p.i]
	if !c.shear() {
		return false
	}
	_, height := render.Size(c.kind, c.young)
	at := c.Pos.Add(geometry.Vector3{Y: height})
	g.ears.Play(sound.Dig(sound.Plant), at, 0.8)
	g.drops = append(g.drops, drop{Drop: world.Drop{Stack: world.Stack{Item: world.Moss, Count: mossShorn}, Pos: at}})
	return true
}

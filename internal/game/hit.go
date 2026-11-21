package game

import (
	"fmt"
	"math"
	"math/rand/v2"
	"slices"

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
		if c.dead || c.ridden {
			continue // it cannot be hit again as it falls, and the player sits on the one it rides
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

// aimEgg finds the nearest egg a ray from origin along dir, one long, meets before limit.
func (g *Game) aimEgg(origin, dir geometry.Vector3, limit float64) (i int, ok bool) {
	best := limit
	for j, e := range g.fauna.eggs {
		if t, hit := rayBox(origin, dir, e.Pos, eggSize*aimWider, eggSize*1.5); hit && t < best {
			best, i, ok = t, j, true
		}
	}
	return i, ok
}

// takeEgg takes the egg at i out of the world and gives it to the player. The herd whose
// nest it lay in turns on the thief: guards charge, and the grown ones of a herd that
// runs run at the thief for a while, and not from it.
func (g *Game) takeEgg(i int) {
	f := g.fauna
	e := f.eggs[i]
	f.eggs = slices.Delete(f.eggs, i, i+1)
	if e.ring {
		for j := range f.eggs {
			if o := &f.eggs[j]; o.wild && o.herd == e.herd {
				o.ring = true // the nest is drawn round the egg that is left
				break
			}
		}
	}
	// Into the bag, or, with no room there, onto the ground where it lay.
	if stack := (world.Stack{Item: breeds[e.kind].egg, Count: 1}); g.items.add(stack) != 0 {
		g.drops = append(g.drops, drop{Drop: world.Drop{Stack: stack, Pos: e.Pos}})
	}
	g.ears.PlayNear(sound.Pickup, 0.8)
	g.swing = swingTime
	if !e.wild || e.herd == 0 {
		return
	}
	if f.thief == nil {
		f.thief = map[int]float64{}
	}
	delete(f.alarms, e.herd)
	f.thief[e.herd] = thiefTime
	if f.raise(e.herd, g.player.Pos) {
		for j := range f.all {
			if c := &f.all[j]; c.herd == e.herd && !c.dead {
				g.voice(c, breeds[c.kind].cry, 1)
				break
			}
		}
	}
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
	g.meet(beastPicture(c.kind))
	if c.hit(from, points) {
		c.carved = carves
		g.helped(c)
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
	food, day := g.items[g.selected].Item, firstMorning+g.days
	if c.sated(day) && slices.Contains(breeds[c.kind].eats, food) {
		g.tell("the young " + c.kind.String() + " has eaten all it will today")
		return true
	}
	ate, tamed := c.feed(food, day)
	if !ate {
		return false
	}
	g.ears.Play(sound.Eat, c.voice(), 0.8)
	g.swing = swingTime
	if g.survival {
		g.items.takeOne(g.selected)
	}
	switch {
	case tamed:
		g.tell("the " + c.kind.String() + " trusts you: it will follow you")
	case c.tame:
		g.tell("the young " + c.kind.String() + " eats: it will grow for a day")
	case c.young:
		g.tell(fmt.Sprintf("the young %s eats from your hand: %d meals of %d", c.kind, c.fed, breeds[c.kind].meals))
	default:
		g.tell("the " + c.kind.String() + " eats from your hand")
	}
	return true
}

// eggKind returns the kind of beast a thing is the egg of, if it is one.
func eggKind(item world.Item) (render.Kind, bool) {
	for k := range render.Kinds {
		if breeds[k].egg == item && item != 0 {
			return k, true
		}
	}
	return 0, false
}

// warmAt reports whether a fire warms a place: one that burns near the player, or a
// hearth that burns, wherever the player is.
func (g *Game) warmAt(at geometry.Vector3) bool {
	for _, fire := range g.burning {
		if fire.Distance(at) <= warmReach {
			return true
		}
	}
	for c, k := range g.hearths {
		if k.Burn > 0 && cellMid(c.X, c.Y, c.Z).Distance(at) <= warmReach {
			return true
		}
	}
	return false
}

// hatch warms the eggs the player put down for dt seconds, each that lies by a fire, and
// hatches one that has had a day of it: what comes out is young, and the player's.
func (g *Game) hatch(dt float64) {
	f := g.fauna
	for i := 0; i < len(f.eggs); i++ {
		e := &f.eggs[i]
		if e.wild || !g.warmAt(e.Pos) {
			continue
		}
		if e.warm += dt; e.warm < hatchTime {
			continue
		}
		f.born++
		own := rand.New(rand.NewPCG(uint64(g.world.Seed), f.born))
		f.all = append(f.all, beast{
			kind: e.kind, Pos: e.Pos, young: true, tame: true, health: breeds[e.kind].health,
			Yaw: own.Float64()*2*math.Pi - math.Pi, timer: own.Float64() * 2, rng: own,
		})
		g.ears.Play(sound.Pickup, e.Pos, 1)
		g.tell("the egg hatches: a young " + e.kind.String() + ", and it is yours")
		f.eggs = slices.Delete(f.eggs, i, i+1)
		i--
	}
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

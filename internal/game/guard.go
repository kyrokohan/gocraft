package game

import (
	"math"

	"glonk.dev/geometry"
	"glonk.dev/mesh"

	"github.com/kyrokohan/skyspear/internal/render"
)

// How a guard lives: a plant-eater that does not run. It watches over its young, if its
// family has any, and over itself. What comes too near is warned: the guard turns on it
// and cries out. Nearer, a shieldbrow lowers its head and charges, and a thornback or a
// boulderback strikes with its tail. The player is such a threat, and so is a hunter.
// Distances are in blocks and times in seconds, and these are first guesses.
const (
	warnAgain  = 4.0  // from one warning to the next
	chargeTime = 4.0  // how long a charge lasts, at the most
	chargeAim  = 0.5  // how far off facing what it charges it may be when it begins to run, in radians
	chargeTurn = 0.9  // how fast a charging beast can turn, in radians a second: a step aside saves the player
	chargeRest = 3.0  // from the end of a charge to the next
	chargeKeen = 3.0  // how many times as far as it charges from a charging beast still follows what it charges
	swingWait  = 1.6  // from one blow of a tail to the next
	swingShow  = 0.5  // how long a tail is seen to swing
	guardTurn  = 2.2  // how fast a guard turns on what threatens it, in radians a second
	blowShove  = 0.27 // how far a blow throws the player, for each point it takes
	blowReach  = 0.7  // how far past its own width the head of a charging beast hits
)

// turnToward returns a Yaw turned from one toward another by no more than by.
func turnToward(from, to, by float64) float64 {
	d := math.Remainder(to-from, 2*math.Pi)
	return from + min(max(d, -by), by)
}

// threat returns what a guard has most to fear for: where the player or the hunter is
// that is nearest to what the guard watches over, and how near that.
func (f *fauna) threat(i int, player geometry.Vector3) (at geometry.Vector3, near float64) {
	c := &f.all[i]
	near, at = math.Inf(1), player
	weigh := func(p geometry.Vector3) {
		for _, e := range f.eggs {
			if d := e.Pos.Distance(p); e.wild && c.herd != 0 && e.herd == c.herd && d < near {
				near, at = d, p // an egg of its nest is as its young are
			}
		}
		for j := range f.all {
			ward := &f.all[j]
			if ward.dead || j != i && (c.herd == 0 || ward.herd != c.herd || !ward.young) {
				continue // it watches over itself, and the young of its family
			}
			if d := ward.Pos.Distance(p); d < near {
				near, at = d, p
			}
		}
	}
	if !c.tame {
		weigh(player) // one that is the player's has nothing to fear from it
	}
	for j := range f.all {
		if h := &f.all[j]; breeds[h.kind].temper == hunter && !h.dead {
			weigh(h.Pos)
		}
	}
	return at, near
}

// strike has a guard's charge or tail hit what is within reach of a point: the player,
// whom it throws the way given, and any hunter. It reports whether it hit anything.
func (f *fauna) strike(i int, at geometry.Vector3, reach float64, way, player geometry.Vector3) (hit bool) {
	c := &f.all[i]
	b := &breeds[c.kind]
	_, height := render.Size(c.kind, false)
	within := func(p geometry.Vector3) bool {
		return math.Hypot(p.X-at.X, p.Z-at.Z) <= reach && p.Y > c.Pos.Y-playerHeight && p.Y < c.Pos.Y+height
	}
	if within(player) {
		f.bitten += b.blow
		f.shove = f.shove.Add(way.Scale(blowShove * b.blow))
		hit = true
	}
	for j := range f.all {
		if h := &f.all[j]; breeds[h.kind].temper == hunter && !h.dead && within(h.Pos) {
			h.hit(c.Pos, b.blow)
			f.sounds = append(f.sounds, said{breeds[h.kind].cry, h.kind, h.voice()})
			hit = true
		}
	}
	return hit
}

// guard has a guard keep watch for dt seconds. A young one only keeps with its family.
func (f *fauna) guard(i int, player geometry.Vector3, dt float64) {
	c := &f.all[i]
	b := &breeds[c.kind]
	c.rest, c.warned, c.lash = max(c.rest-dt, 0), max(c.warned-dt, 0), max(c.lash-dt, 0)
	c.lower = false
	if c.young {
		c.angry = false
		return
	}

	foe, near := f.threat(i, player)
	if c.tame {
		c.angry = false // it does not turn on the player, whatever the player did
	}
	if c.angry {
		// It was hurt, or its young were: it turns on where that came from, however far.
		c.angry, foe, near = false, c.foe, 0
	}
	width, _ := render.Size(c.kind, false)
	back, front := render.Span(c.kind, false)
	ahead := mesh.Camera{Yaw: c.Yaw}.Forward()
	stand := func(yaw float64) { // it stands and turns, and does nothing else for now
		c.Yaw = turnToward(c.Yaw, yaw, guardTurn*dt)
		c.walking, c.timer = false, max(c.timer, rethink)
	}
	cry := func() {
		if c.warned == 0 {
			c.warned = warnAgain
			f.sounds = append(f.sounds, said{b.cry, c.kind, c.voice()})
		}
	}

	switch to := (geometry.Vector3{X: foe.X - c.Pos.X, Z: foe.Z - c.Pos.Z}); {
	case c.ire > 0:
		// A charge: head down, at a run, at where what it charges is now, but it cannot
		// turn fast. It ends on what it hits, or when its time is up.
		if near < chargeKeen*b.charges {
			c.foe = foe
		}
		at := geometry.Vector3{X: c.foe.X - c.Pos.X, Z: c.foe.Z - c.Pos.Z}
		if c.lower = true; c.ire == chargeTime && math.Abs(math.Remainder(yawOf(at)-c.Yaw, 2*math.Pi)) > chargeAim {
			stand(yawOf(at)) // it turns to face what it will charge, before it begins to run
			break
		}
		c.Yaw = turnToward(c.Yaw, yawOf(at), chargeTurn*dt)
		c.walking, c.hurry, c.timer = true, true, max(c.timer, rethink)
		head := c.Pos.Add(ahead.Scale(front - width/2))
		if c.ire -= dt; f.strike(i, head, width/2+blowReach, ahead, player) || c.ire <= 0 {
			c.ire, c.rest = 0, chargeRest
			c.walking, c.timer = false, rethink
		}
	case b.charges > 0 && near < b.charges && c.rest == 0:
		c.ire, c.foe, c.warned = chargeTime, foe, 0
		cry()
	case b.swings > 0 && near < b.warns:
		// It turns its tail to the threat, and strikes when that is in reach of it.
		stand(yawOf(to.Neg()))
		cry()
		tail := c.Pos.Add(ahead.Scale(back / 2))
		if c.rest == 0 && math.Hypot(foe.X-tail.X, foe.Z-tail.Z) <= b.swings {
			c.rest, c.lash, c.lashSide = swingWait, swingShow, 1
			if ahead.X*to.Z-ahead.Z*to.X > 0 {
				c.lashSide = -1 // to the side the threat is on
			}
			f.strike(i, tail, b.swings, awayFrom(foe, c.Pos, c.Yaw), player)
		}
	case near < b.warns:
		// It turns on the threat, lowers its head and cries out.
		stand(yawOf(to))
		c.lower = true
		cry()
	case b.rears && !c.walking:
		c.lower = true // it has stopped, and stands up as it does to feed from a tree
	}
}

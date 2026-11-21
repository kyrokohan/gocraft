package game

import (
	"math"
	"slices"

	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/sound"
	"github.com/kyrokohan/skyspear/internal/world"
)

// Work. One who lives at the player's fire is given a job by being shown a thing: wood,
// or anything else that burns, to keep the fire, and berries to gather. A fire keeper
// keeps the hearth burning with fuel from the pots by it, and cooks the raw meat in
// them, putting what is cooked back. A gatherer goes out by day, picks berries and what
// lies on the ground near the fire, and brings it to a pot by it. They work through the
// hearth's store and the pots, with nobody at them, while the player is near enough
// for them to move.
const (
	workEvery    = 2.0  // how often a keeper sees to the hearth, in seconds
	potReach     = 4    // how near the hearth a pot must be to be by it, in blocks
	keepFuel     = 4    // a keeper puts fuel in the hearth when it has less than this
	keepMeat     = 8    // and at most this much raw meat in at once
	gatherReach  = 24.0 // how far from the fire a gatherer goes, in blocks
	gatherLoad   = 4    // how many things they pick before they bring them home
	gatherGiveUp = 30.0 // how long they try to reach a thing, in seconds
	gatherRest   = 5.0  // and how long they rest when there is nothing to pick
	pickNear     = 1.3  // how near they come to a thing to pick it
)

// job is the work of one who lives at the player's fire.
type job uint8

const (
	idle job = iota
	keeper
	gatherer
	jobs
)

// jobNames are the jobs as a save names them.
var jobNames = [jobs]string{"", "keeper", "gatherer"}

// jobTold is what the player is told when one is given a job.
var jobTold = [jobs]string{"", "they will keep your fire", "they will gather for your fire"}

// jobFor is the job a thing shown gives: a keeper's for what burns, a gatherer's for
// berries.
func jobFor(item world.Item) (job, bool) {
	switch {
	case burnTime(item) > 0:
		return keeper, true
	case item == world.Berries:
		return gatherer, true
	}
	return idle, false
}

// What a gatherer is on the way to do.
const (
	errandNone = iota
	errandDrop // pick up what lies at goal
	errandBush // pick the bush at goalCell
	errandHome // bring what they carry to a pot by the fire
)

// addTo puts a stack into slots, onto the stacks of its item with room first and then
// into the empty ones, and returns how many did not fit. A tool keeps its wear.
func addTo(slots []world.Stack, s world.Stack) (left int) {
	n, size := s.Count, s.Item.StackSize()
	for i := range slots {
		if in := &slots[i]; in.Item == s.Item && in.Count < size {
			put := min(n, size-in.Count)
			in.Count += put
			n -= put
		}
	}
	for i := range slots {
		if in := &slots[i]; n > 0 && in.Count == 0 {
			put := min(n, size)
			*in = world.Stack{Item: s.Item, Count: put, Wear: s.Wear}
			n -= put
		}
	}
	return n
}

// topUp fills a stack from pots with what ok lets in, up to n in it: more of its own
// item, or, when it is empty, of the first such item the pots hold.
func topUp(s world.Stack, pots []*world.PotStore, ok func(world.Item) bool, n int) world.Stack {
	for _, p := range pots {
		for i := range p.Items {
			in := &p.Items[i]
			if in.Count == 0 || !ok(in.Item) || s.Count > 0 && in.Item != s.Item || s.Count >= n {
				continue
			}
			put := min(n-s.Count, in.Count)
			s = world.Stack{Item: in.Item, Count: s.Count + put}
			takeFrom(in, put)
		}
	}
	return s
}

// potsNear returns the stores of the pots by a hearth.
func (g *Game) potsNear(c world.Cell) []*world.PotStore {
	var pots []*world.PotStore
	for y := c.Y - 2; y <= c.Y+2; y++ {
		for z := c.Z - potReach; z <= c.Z+potReach; z++ {
			for x := c.X - potReach; x <= c.X+potReach; x++ {
				if g.world.At(x, y, z) == world.ClayPot {
					pots = append(pots, g.pot(world.Cell{X: x, Y: y, Z: z}))
				}
			}
		}
	}
	return pots
}

// keep is a fire keeper seeing to the hearth at a cell: what it has cooked goes into the
// pots by it, raw meat from them goes in to be cooked, fuel from them goes in when it
// runs low, and one that has gone out is lit again.
func (g *Game) keep(c world.Cell) {
	if !isHearth(g.world.At(c.X, c.Y, c.Z)) {
		return
	}
	k, pots := g.hearth(c), g.potsNear(c)
	for _, p := range pots {
		if k.Out.Count > 0 {
			if k.Out.Count = addTo(p.Items, k.Out); k.Out.Count == 0 {
				k.Out = world.Stack{}
			}
		}
	}
	if k.In.Count == 0 || k.In.Item == world.Meat {
		k.In = topUp(k.In, pots, func(i world.Item) bool { return i == world.Meat }, keepMeat)
	}
	if k.Fuel.Count < keepFuel {
		k.Fuel = topUp(k.Fuel, pots, func(i world.Item) bool { return burnTime(i) > 0 }, 2*keepFuel)
	}
	if k.Burn == 0 && kindle(k) {
		g.ears.Play(sound.Crackle, cellMid(c.X, c.Y, c.Z), 0.9)
	}
}

// work has one who lives at the player's fire do their job, dt seconds on. It reports
// whether that is what moved them, so that they do not also go about as the others do.
func (g *Game) work(one *person, night bool, dt float64) bool {
	switch {
	case one.follows:
		return false
	case one.job == keeper:
		if one.workIn -= dt; one.workIn <= 0 {
			one.workIn = workEvery
			g.keep(one.fire)
		}
		return false
	case one.job == gatherer && !night:
		g.gather(one, dt)
		return true
	}
	return false
}

// gather has a gatherer go about their work, dt seconds on: to the nearest thing to
// pick within gatherReach of the fire, and, once they have picked gatherLoad things or
// there are no more, home with them. A thing they cannot reach in gatherGiveUp seconds
// they give up, and do not try again until they have been home.
func (g *Game) gather(one *person, dt float64) {
	one.seated = false
	if one.aside > 0 {
		one.aside -= dt
		return
	}
	one.timer -= dt
	if one.errand == errandNone {
		one.walking = false
		switch {
		case one.timer > 0:
		case one.picks < gatherLoad && g.findPicking(one):
			one.timer = gatherGiveUp
		case one.picks > 0:
			one.errand, one.goal, one.timer = errandHome, cellMid(one.fire.X, one.fire.Y, one.fire.Z), gatherGiveUp
		default:
			one.timer = gatherRest
		}
		return
	}
	to := one.goal.Sub(one.Pos)
	to.Y = 0
	near := pickNear
	if one.errand == errandHome {
		near = seatOut + 0.5
	}
	switch {
	case to.Length() < near:
		one.walking = false
		g.reach(one)
		one.errand, one.timer = errandNone, 1
	case one.timer <= 0 && one.errand == errandHome:
		g.reach(one) // they put it down where they are
		one.errand = errandNone
	case one.timer <= 0:
		one.missed = append(one.missed, one.goalCell)
		one.errand = errandNone
	default:
		one.walking, one.Yaw = true, yawOf(to)
	}
}

// findPicking sets a gatherer on the way to the nearest thing to pick: what lies on the
// ground within gatherReach of the fire, and then a bush. It reports whether there was
// one.
func (g *Game) findPicking(one *person) bool {
	mid := cellMid(one.fire.X, one.fire.Y, one.fire.Z)
	within := func(p geometry.Vector3) bool { return math.Hypot(p.X-mid.X, p.Z-mid.Z) <= gatherReach }
	best := math.Inf(1)
	for _, d := range g.drops {
		c := world.Cell{X: int(math.Floor(d.Pos.X)), Y: int(math.Floor(d.Pos.Y)), Z: int(math.Floor(d.Pos.Z))}
		if far := d.Pos.Distance(one.Pos); far < best && d.Age >= pickWait && within(d.Pos) && !slices.Contains(one.missed, c) {
			best, one.errand, one.goal, one.goalCell = far, errandDrop, d.Pos, c
		}
	}
	if one.errand != errandNone {
		return true
	}
	r := int(gatherReach)
	for y := one.fire.Y - 4; y <= one.fire.Y+4; y++ {
		for z := one.fire.Z - r; z <= one.fire.Z+r; z++ {
			for x := one.fire.X - r; x <= one.fire.X+r; x++ {
				c := world.Cell{X: x, Y: y, Z: z}
				at := cellMid(x, y, z)
				if g.world.At(x, y, z) != world.Bush || !within(at) || slices.Contains(one.missed, c) {
					continue
				}
				if far := at.Distance(one.Pos); far < best {
					best, one.errand, one.goal, one.goalCell = far, errandBush, at, c
				}
			}
		}
	}
	return one.errand != errandNone
}

// reach is a gatherer come to where they were going: they pick up what lies there, or
// pick the bush, as the player would break it, or put what they carry into the pots by
// the fire, and down by it what does not fit.
func (g *Game) reach(one *person) {
	switch one.errand {
	case errandDrop:
		for i := range g.drops {
			if d := &g.drops[i]; d.Count > 0 && math.Hypot(d.Pos.X-one.Pos.X, d.Pos.Z-one.Pos.Z) < pickNear+0.5 {
				d.Count = addTo(one.carry[:], d.Stack)
				one.picks++
				g.ears.Play(sound.Pickup, one.Pos, 0.6)
			}
		}
		g.drops = slices.DeleteFunc(g.drops, func(d drop) bool { return d.Count == 0 })
	case errandBush:
		c := one.goalCell
		if g.world.At(c.X, c.Y, c.Z) == world.Bush {
			g.world.Set(c.X, c.Y, c.Z, world.Air)
			item, n := world.Bush.Drop()
			if left := addTo(one.carry[:], world.Stack{Item: item, Count: n}); left > 0 {
				g.drops = append(g.drops, drop{Drop: world.Drop{Stack: world.Stack{Item: item, Count: left}, Pos: cellMid(c.X, c.Y, c.Z)}})
			}
			one.picks++
			g.ears.Play(sound.Break(sound.Plant), cellMid(c.X, c.Y, c.Z), 0.7)
		}
	case errandHome:
		pots := g.potsNear(one.fire)
		for i := range one.carry {
			s := &one.carry[i]
			for _, p := range pots {
				if s.Count > 0 {
					s.Count = addTo(p.Items, *s)
				}
			}
			if s.Count > 0 {
				g.drops = append(g.drops, drop{Drop: world.Drop{Stack: *s, Pos: one.Pos.Add(geometry.Vector3{Y: 0.5})}})
			}
			*s = world.Stack{}
		}
		one.picks, one.missed = 0, nil
		g.ears.Play(sound.Pickup, one.Pos, 0.6)
	}
}

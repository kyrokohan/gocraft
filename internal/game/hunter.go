package game

import (
	"math"
	"slices"

	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/world"
)

// How a hunter lives: it comes in the dark, goes for the player it notices or for a
// beast it can kill, and keeps out of the light. Distances are in blocks, times in
// seconds, light as Sky.Brightness gives it, and these are first guesses.
const (
	hunterLit  = 0.315 // the light it will not step into: a torch three blocks away, or the day
	firstHunt  = 1.5   // the days since the world's first midnight before any hunts the player: the first night is quiet
	hunterNear = 16.0  // the least distance from the player it comes out at
	hunterFar  = 36.0  // and the most
	lairSearch = 8     // blocks above and below the player's feet that a place is looked for in

	sneakSight  = 0.5 // how much of the distance it sees a player at is left for a crouched one
	hiddenSight = 3.0 // how far it sees a player crouched among tall plants
	sneakHears  = 3.0 // how far it hears a crouched player move
	meatScent   = 2.0 // how many times farther it notices a player who carries raw meat
	keenTime    = 6.0 // how long it keeps after a player it has lost sight and sound of

	biteReach = 1.2  // how near it comes to bite
	biteWait  = 1.0  // between two bites
	preySight = 16.0 // how far it sees a beast to hunt
	feedTime  = 15.0 // how long it stands over a kill

	litTime    = 2.0  // how long it bears the light before it leaves
	leaveAt    = 24.0 // how far from the player one that leaves is let go
	leaveAfter = 20.0 // and how long after it turned to go, wherever it is

	hurtPart   = 0.4  // the part of its health below which it thinks of running
	aloneReach = 10.0 // how near another of its pack must be for it to stand its ground
	routTime   = 8.0  // how long one that is hurt and alone runs
	staggerFor = 0.5  // how long a hit sets back one that stands its ground

	glowBelow = 0.5 // the light under which eyes that can shine do
)

// brightAt returns how bright the light at a place is. With no light to ask it is dark:
// a herd in a test knows no sky.
func (f *fauna) brightAt(at geometry.Vector3) float64 {
	if f.bright == nil {
		return 0
	}
	return f.bright(at)
}

// notices reports whether a hunter sees or hears a player standing at a place.
//
// It sees a player from as far as its breed sees, from half that if the player is
// crouched, and only from close by one crouched among tall plants, and never through a
// solid block. It hears a player who moves: a crouched one only from close by. Raw meat
// carried is smelled, and doubles every distance.
func (f *fauna) notices(w *world.World, c *beast, player geometry.Vector3) bool {
	b := &breeds[c.kind]
	sees, hears := b.sees, 0.0
	if f.sneaking {
		sees *= sneakSight
	}
	if f.hidden {
		sees = hiddenSight
	}
	if f.moving {
		if hears = b.hears; f.sneaking {
			hears = sneakHears
		}
	}
	if f.meaty {
		sees, hears = sees*meatScent, hears*meatScent
	}
	far := c.Pos.Distance(player)
	if far < hears {
		return true
	}
	return far < sees && clearSight(w, c.voice(), player.Add(geometry.Vector3{Y: 1.2}))
}

// clearSight reports whether no solid block lies between two points.
func clearSight(w *world.World, from, to geometry.Vector3) bool {
	way := to.Sub(from)
	far := way.Length()
	if far < 1e-6 {
		return true
	}
	r := world.NewRay(from, way.Scale(1/far))
	for {
		hit, ok := w.Next(&r, far)
		if !ok {
			return true
		}
		if hit.Block.Info().Solid {
			return false
		}
	}
}

// lair returns where a hunter could come out in the column at x, z: a floor near the
// height of the player's feet, in the dark, with room over it for its body.
func (f *fauna) lair(w *world.World, k render.Kind, x, z, feet int) (geometry.Vector3, bool) {
	if !w.Loaded(x, z) {
		return geometry.Vector3{}, false
	}
	_, height := render.Size(k, false)
	for y := feet + lairSearch; y >= feet-lairSearch; y-- {
		if !solid(w, x, y-1, z) {
			continue
		}
		room := true
		for up := range int(math.Ceil(height)) {
			if b := w.At(x, y+up, z); b == world.Water || b.Info().Solid {
				room = false
			}
		}
		if !room {
			continue
		}
		pos := geometry.Vector3{X: float64(x) + 0.5, Y: float64(y), Z: float64(z) + 0.5}
		// The floor nearest the player's height there is the one: if it is lit, none.
		return pos, f.brightAt(pos.Add(geometry.Vector3{Y: 1})) <= hunterLit
	}
	return geometry.Vector3{}, false
}

// prey returns the nearest beast a hunter could kill that it sees, or nil.
func (f *fauna) prey(c *beast) *beast {
	var nearest *beast
	best := preySight
	for i := range f.all {
		p := &f.all[i]
		if p.dead || breeds[p.kind].health == 0 || breeds[p.kind].temper == hunter {
			continue
		}
		if d := p.Pos.Distance(c.Pos); d < best {
			best, nearest = d, p
		}
	}
	return nearest
}

// packNear reports whether another living beast of the same herd is within a distance
// of the one at i.
func (f *fauna) packNear(i int, reach float64) bool {
	c := &f.all[i]
	for j := range f.all {
		if o := &f.all[j]; j != i && o.herd == c.herd && o.kind == c.kind && !o.dead && o.Pos.Distance(c.Pos) <= reach {
			return true
		}
	}
	return false
}

// hunt decides what the hunter at i does for the next dt seconds: where it turns, and
// whether it runs, stands or bites. Its walking is then the same as any beast's.
func (f *fauna) hunt(w *world.World, i int, player geometry.Vector3, dt float64) {
	c := &f.all[i]
	b := &breeds[c.kind]
	c.biteIn = max(c.biteIn-dt, 0)

	// It does not bear the light: after a little of it, by day or beside a fire, it
	// turns from the player and goes.
	if f.brightAt(c.Pos.Add(geometry.Vector3{Y: 1})) > hunterLit {
		c.lit += dt
	} else {
		c.lit = 0
	}
	if c.lit > litTime && !c.leaving {
		c.leaving, c.keen, c.feeding = true, 0, 0
		c.Yaw = yawOf(awayFrom(c.Pos, player, c.Yaw))
	}
	if c.leaving {
		c.gone += dt
		c.flee = max(c.flee, staggerFor) // it runs, and turns aside from what is in its way
		return
	}
	if c.flee > 0 {
		return
	}
	if c.feeding > 0 {
		c.feeding -= dt
		c.walking, c.timer = false, max(c.timer, rethink)
		return
	}

	// It goes for a player it notices, and keeps after one it has lost for a while.
	// With no player to go for, it goes for a beast it can kill.
	if f.hunted && f.notices(w, c, player) {
		if c.keen <= 0 {
			f.sounds = append(f.sounds, said{b.cry, c.kind, c.voice()}) // it has seen its prey
		}
		c.keen = keenTime
	} else {
		c.keen = max(c.keen-dt, 0)
	}
	goal, quarry := player, (*beast)(nil)
	if c.keen <= 0 {
		if quarry = f.prey(c); quarry == nil {
			return // nothing to hunt: it goes about with its pack
		}
		goal = quarry.Pos
	}

	to := geometry.Vector3{X: goal.X - c.Pos.X, Z: goal.Z - c.Pos.Z}
	far := to.Length()
	if far > 1e-6 {
		c.Yaw = yawOf(to)
	}
	c.timer = max(c.timer, rethink)
	if far < biteReach && math.Abs(goal.Y-c.Pos.Y) < 2 {
		c.walking = false
		if c.biteIn > 0 {
			return
		}
		c.biteIn = biteWait
		if quarry == nil {
			f.bitten += b.bite
			return
		}
		if f.raise(quarry.herd, c.Pos) {
			f.sounds = append(f.sounds, said{breeds[quarry.kind].cry, quarry.kind, quarry.voice()})
		}
		if quarry.hit(c.Pos, b.bite) {
			c.feeding = feedTime
		}
		return
	}
	// It will not step into the light: it stands at the edge of it, and waits.
	width, _ := render.Size(c.kind, c.young)
	if far > 1e-6 && f.brightAt(c.Pos.Add(to.Scale((width/2+0.8)/far)).Add(geometry.Vector3{Y: 1})) > hunterLit {
		c.walking = false
		return
	}
	c.walking, c.hurry = true, true
}

// dismiss lets go of every beast that comes only in the dark: the night is over, or
// the player is somewhere else.
func (f *fauna) dismiss() {
	f.all = slices.DeleteFunc(f.all, func(c beast) bool { return breeds[c.kind].night })
}

// hunterWithin reports whether a living hunter is within a distance of the player.
func (g *Game) hunterWithin(d float64) bool {
	for i := range g.fauna.all {
		c := &g.fauna.all[i]
		if breeds[c.kind].temper == hunter && !c.dead && !c.leaving && c.Pos.Distance(g.player.Pos) <= d {
			return true
		}
	}
	return false
}

// hidden reports whether the player is out of sight: crouched among tall ferns or
// horsetails.
func (g *Game) hidden() bool {
	p := g.player.Pos
	x, y, z := int(math.Floor(p.X)), int(math.Floor(p.Y+0.1)), int(math.Floor(p.Z))
	in := func(b world.Block) bool { return b == world.TallFern || b == world.Horsetail }
	return g.player.Crouched && (in(g.world.At(x, y, z)) || in(g.world.At(x, y+1, z)))
}

package game

import (
	"math"

	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/sound"
	"github.com/kyrokohan/skyspear/internal/world"
)

// A fire mountain erupts, on days the seed gives it. For a minute the ground trembles,
// more and more, the mountain rumbles, and the beasts near it run. Then for a minute and
// a half burning rocks arc out of the crater and come down on the ash round it: one
// hurts what it hits, and leaves a block of rock with a fire on it that burns out. Ash
// darkens the sky while it does, and for half a day after. A player farther off than
// eruptReach knows nothing of it. Times are in seconds and distances in blocks.
const (
	eruptReach = 300.0 // how far from the crater an eruption is known at all
	eruptFull  = 200.0 // and how near it is known in full
	eruptWarn  = 60.0  // how long the ground trembles before the first rock
	eruptTime  = 90.0  // how long the rocks fly
	ashRise    = 15.0  // from the first rock until the sky is full of ash
	ashAfter   = 0.5   // days from the last rock until the sky is clear again

	tremorMost = 0.4   // how hard the ground shakes at the end of the minute, and while rocks fly
	tremorBeat = 2.0   // from one rumble of the mountain to its next
	driveReach = 100.0 // how far from the crater beasts run from it
	driveTime  = 3.0   // how long a beast runs each time the mountain rumbles

	rockEvery  = 0.75  // from one rock to the next
	rockNear   = 14.0  // the nearest the middle of the crater a rock is aimed: past the rim
	rockFar    = 64.0  // and the farthest: none comes down past the ash, 70 blocks out
	rockAir    = 2.5   // how long the nearest rock is in the air
	rockPace   = 30.0  // how many blocks farther a rock is aimed for each second more of it
	rockHarm   = 10.0  // points a rock takes from what it hits
	rockHeard  = 120.0 // how far off a rock is heard to leave the crater, and to land
	rockFelt   = 24.0  // how far off its landing shakes the ground
	rockTries  = 8     // how many places are tried for one rock before none is thrown
	trailEvery = 0.1   // from one puff of the fire a rock trails to the next
	spoutEvery = 0.06  // from one puff of the fire that stands over the crater to the next
	spoutRise  = 16.0  // how fast that fire goes up, in blocks a second
	obsidianIn = 4     // one rock in so many leaves obsidian, and the others dark rock
	bedSafe    = 24.0  // how near the bed no rock comes down
	bedMargin  = 3.0   // how much farther than that from the bed a rock is aimed
	emberTime  = 60.0  // how long the fire a rock leaves burns
)

// erupting returns the crater of the fire mountain near the player, how many seconds ago
// it last began to erupt, and how much of that the player knows, from 0 to 1: all of it
// within eruptFull blocks, and none from eruptReach. With no eruption to know of, the
// seconds are without end.
func (g *Game) erupting() (crater world.Cell, since, near float64) {
	p := g.player.Pos
	crater, far, ok := g.world.Mountain(p.X, p.Z)
	if !ok || far >= eruptReach {
		return crater, math.Inf(1), 0
	}
	began, _, ok := g.world.Erupted(p.X, p.Z, g.days)
	if !ok {
		return crater, math.Inf(1), 0
	}
	return crater, (g.days - began) * dayLength, min((eruptReach-far)/(eruptReach-eruptFull), 1)
}

// ash returns how much ash the sky holds, from 0 to 1, and how much of the fire of a
// mountain that is throwing it out is seen: the ash comes with the first rocks, and
// thins for half a day after the last.
func (g *Game) ash() (ash, fire float64) {
	_, since, near := g.erupting()
	switch after := since - eruptWarn - eruptTime; {
	case since < eruptWarn:
		return 0, 0
	case after < 0:
		return near * min((since-eruptWarn)/ashRise, 1), near
	default:
		return near * max(1-after/(ashAfter*dayLength), 0), 0
	}
}

// erupt runs the fire mountain near the player for dt seconds: the tremble and the
// rumble of the minute before, the run of the beasts near it, and then the rocks. The
// fires that rocks left burn down whether it erupts or not.
func (g *Game) erupt(dt float64) {
	g.stepEmbers(dt)
	crater, since, near := g.erupting()
	if since >= eruptWarn+eruptTime {
		return
	}
	top := cellMid(crater.X, crater.Y, crater.Z)
	grown := min(since/eruptWarn, 1)
	g.quake = max(g.quake, tremorMost*near*grown)
	if math.Floor(since/tremorBeat) != math.Floor((since-dt)/tremorBeat) {
		g.ears.PlayFar(sound.Tremor, top, 0.4+0.6*grown, eruptReach)
		g.fauna.drive(top, driveReach, driveTime)
	}
	if on := since - eruptWarn; on >= 0 && math.Floor(on/rockEvery) != math.Floor((on-dt)/rockEvery) {
		g.fling(crater)
	}
	if on := since - eruptWarn; on >= 0 && top.Distance(g.player.Pos) < smokeReach && math.Floor(on/spoutEvery) != math.Floor((on-dt)/spoutEvery) {
		// A fountain of fire stands over the crater while the rocks fly.
		g.flame(top.Add(geometry.Vector3{X: 6 * (g.scatter.Float64() - 0.5), Z: 6 * (g.scatter.Float64() - 0.5)}), 1.5, 1.6, spoutRise)
	}
	for _, m := range g.missiles {
		if fliers[m.kind].burns && math.Floor(m.age/trailEvery) != math.Floor((m.age-dt)/trailEvery) {
			g.flame(m.pos, 0.7, 0.8, 0.5) // the fire it trails
		}
	}
}

// fling throws one burning rock out of the crater, at a place on the ash round it where
// the world is loaded, and which is not near the bed. It tries a few places, and throws
// no rock if none of them will do.
func (g *Game) fling(crater world.Cell) {
	from := cellMid(crater.X, crater.Y+2, crater.Z)
	for range rockTries {
		bearing, far := 2*math.Pi*g.scatter.Float64(), rockNear+(rockFar-rockNear)*math.Sqrt(g.scatter.Float64())
		x, z := int(math.Floor(from.X+far*math.Cos(bearing))), int(math.Floor(from.Z+far*math.Sin(bearing)))
		if !g.world.Loaded(x, z) || g.bed != nil && math.Hypot(float64(x-g.bed.X), float64(z-g.bed.Z)) < bedSafe+bedMargin {
			continue
		}
		g.missiles = append(g.missiles, rockTo(from, cellMid(x, g.world.Ground(x, z)+1, z)))
		g.ears.PlayFar(sound.Thud, from, 1, rockHeard)
		return
	}
}

// rockTo returns a burning rock thrown from one place so as to come down at another: it
// goes high, and is the longer in the air the farther it goes.
func rockTo(from, to geometry.Vector3) missile {
	way := to.Sub(from)
	t := rockAir + math.Hypot(way.X, way.Z)/rockPace
	return missile{
		kind: rockFlung, stack: world.Stack{Item: world.Lava.Item(), Count: 1}, pos: from,
		vel: geometry.Vector3{X: way.X / t, Y: way.Y/t + fliers[rockFlung].weight*t/2, Z: way.Z / t},
	}
}

// struck reports whether a burning rock at a point has met the player, whom it hurts:
// but not in creative, where nothing does.
func (g *Game) struck(at geometry.Vector3) bool {
	lo, hi := boxOf(g.player.Pos, playerWidth+fliers[rockFlung].size, playerHeight)
	if at.X < lo[0] || at.X > hi[0] || at.Y < lo[1] || at.Y > hi[1] || at.Z < lo[2] || at.Z > hi[2] {
		return false
	}
	if g.survival {
		g.hurt(rockHarm)
	}
	return true
}

// land has a burning rock come down at a place: it is heard and felt, and it leaves a
// block of rock there, obsidian now and then, with a fire on it that burns out. It
// leaves nothing in water or lava, and nothing within bedSafe blocks of the bed.
func (g *Game) land(at geometry.Vector3) {
	x, y, z := int(math.Floor(at.X)), int(math.Floor(at.Y)), int(math.Floor(at.Z))
	g.ears.PlayFar(sound.Thud, at, 1, rockHeard)
	if far := at.Distance(g.player.Pos); far < rockFelt {
		g.quake = max(g.quake, 1-far/rockFelt)
	}
	for range 3 {
		g.puff(at, 0.6, 2)
	}
	if g.bed != nil && math.Hypot(float64(x-g.bed.X), float64(z-g.bed.Z)) < bedSafe {
		return
	}
	if g.world.At(x, y, z) != world.Air {
		return
	}
	rock := world.Basalt
	if g.scatter.IntN(obsidianIn) == 0 {
		rock = world.ObsidianRock
	}
	g.world.Set(x, y, z, rock)
	if g.world.At(x, y+1, z) == world.Air {
		g.world.Set(x, y+1, z, world.Fire)
		g.embers = append(g.embers, world.Ember{At: world.Cell{X: x, Y: y + 1, Z: z}, Left: emberTime})
	}
}

// stepEmbers burns what burns for a time for dt seconds more, and changes each whose
// time is up into what it turns into: a fire goes out, and the hot core of a star goes
// dark. One that is out of the window is kept until it is seen again.
func (g *Game) stepEmbers(dt float64) {
	kept := g.embers[:0]
	for _, e := range g.embers {
		if e.Left -= dt; e.Left > 0 || !g.world.Loaded(e.At.X, e.At.Z) {
			kept = append(kept, e)
			continue
		}
		if b := g.world.At(e.At.X, e.At.Y, e.At.Z); b == world.Fire || b == world.HotCore {
			g.world.Set(e.At.X, e.At.Y, e.At.Z, e.Into)
		}
	}
	g.embers = kept
}

// drive has every beast within reach of a place, along the ground, run from it for a
// time: all but a tame one, which keeps by the player.
func (f *fauna) drive(from geometry.Vector3, reach, time float64) {
	for i := range f.all {
		c := &f.all[i]
		if c.dead || c.tame || math.Hypot(c.Pos.X-from.X, c.Pos.Z-from.Z) > reach {
			continue
		}
		c.flee, c.walking = max(c.flee, time), true
		c.Yaw = yawOf(awayFrom(c.Pos, from, c.Yaw))
	}
}

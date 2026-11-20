package game

import (
	"math"

	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/sound"
	"github.com/kyrokohan/skyspear/internal/world"
)

// Stars fall. Shooting stars streak the night sky, and on a night the seed names one
// comes down near the player: a fireball crosses the sky, which draws it, and strikes.
// The flash is seen at once, the boom comes when its sound has crossed the land, and
// where it struck is a crater with fires round it and a core of sky iron in it, too hot
// to touch at first. Times are in seconds and distances in blocks.
const (
	streaksAMinute = 5.0  // how many shooting stars streak the sky in a minute of the night
	streaksMost    = 30.0 // and how many on the night the comet arrives
	starSpan       = 0.3  // the part of a day, round midnight, in which a star may fall

	cometNight = 12 // the night on which the Skyspear first arrives, counted from 1
	cometEvery = 24 // how many days it is from one arrival to the next
	cometFalls = 3  // in so many nights before it arrives a star falls near the player every night

	// The Great Fall: round the midnight the Skyspear arrives stars come down thick, and
	// then the Skyspear itself strikes, far off.
	fallLasts = 180.0 // how long the stars come down: half of it before that midnight
	fallEvery = 2.0   // from one of them to the next
	fallFar   = 96.0  // how far from the player the farthest is aimed: any nearer may be
	fallScare = 40.0  // how far from where a star strikes beasts run from it
	fallRun   = 5.0   // and for how long

	whiteTime    = 3.0  // how long the sky is white when the Skyspear strikes
	skyspearBoom = 20.0 // how long its boom is on its way: it struck 800 blocks off
	shockTime    = 6.0  // how long the ground shakes when its boom comes
	ashClimb     = 60.0 // how long its ash takes to fill the sky
	winterAsh    = 0.7  // how much ash the sky holds through the winter after
	winterDays   = 4.0  // how many days that winter lasts

	starNear   = 40.0  // the nearest the player a star is aimed
	starFar    = 90.0  // and the farthest
	starFlight = 4.0   // how long a fireball is in the sky
	starHigh   = 100.0 // how high over where it is aimed it begins
	starAside  = 110.0 // and how far to the side of it
	starTail   = 0.8   // how many seconds of its way a fireball's tail is long
	starTries  = 12    // how many places are tried for one star before none falls

	flashTime = 0.6   // how long the flash of a star that struck lights the land
	boomPace  = 40.0  // how fast its boom crosses the land, in blocks a second
	boomHeard = 400.0 // how far off the boom is heard
	boomFelt  = 200.0 // and how far off it shakes the ground

	bowlWide    = 3.5       // how far from its middle a crater reaches: it is seven across
	bowlDeep    = 3         // how deep it is
	blastHigh   = 6.0       // how far over its rim what stands in it is hurt
	blastHarm   = 12.0      // points a star takes from what is in its crater when it strikes
	craterFires = 5         // how many fires it leaves round its crater
	starFire    = dayLength // how long those burn: a day
	coreHot     = 120.0     // how long the core is too hot to touch
	coreFalls   = 12        // how far under its crater a core that broke through a roof is looked for a floor to lie in
	storeSafe   = 8.0       // how near a pot or a hearth no star comes down
	mountSafe   = 75.0      // how near the middle of a fire mountain none does: on its ash
)

// boom is the sound of a star that struck, on its way to the player.
type boom struct {
	at    geometry.Vector3
	left  float64 // seconds until it is heard
	great bool    // it is the boom of the Skyspear itself, which shakes the ground hard
}

// cometAt returns how near the Skyspear is on a day of the world, from 0, when the last
// has only just gone, to 1 at the midnight it arrives, and how many days it is until
// then. It grows through the days of the first, which are fewer, as through those of
// every one after.
func cometAt(days float64) (near, left float64) {
	first := cometNight - firstMorning // the midnight of the night it first arrives
	if days <= first {
		return days / first, first - days
	}
	since := math.Mod(days-first, cometEvery)
	return since / cometEvery, cometEvery - since
}

// comet returns how near the Skyspear is now, and how many days it is until it arrives.
// Through the Great Fall it is all here, until it strikes.
func (g *Game) comet() (near, left float64) {
	near, left = cometAt(g.days)
	if since := g.greatFall(); since >= 0 && since < fallLasts {
		near = 1
	}
	return near, left
}

// greatFall returns how many seconds ago the last Great Fall began: its stars come down
// for fallLasts, round the midnight the Skyspear arrives, and then the Skyspear strikes.
// Before the first has begun it is fewer than none. It is by the clock alone, so a
// world opened half way through one goes on with it, and nothing of it is in a save.
func (g *Game) greatFall() float64 {
	first := cometNight - firstMorning - fallLasts/2/dayLength // the day the first begins
	if g.days < first {
		return (g.days - first) * dayLength
	}
	return math.Mod(g.days-first, cometEvery) * dayLength
}

// fallTonight reports whether the Great Fall is on, or begins within half a day: on such
// a night nobody sleeps.
func (g *Game) fallTonight() bool {
	since := g.greatFall()
	until := -since
	if since >= 0 {
		until = cometEvery*dayLength - since
	}
	return since >= 0 && since < fallLasts || until < dayLength/2
}

// fallAsh returns how much ash the Great Fall has put in the sky: none until the
// Skyspear strikes, then it climbs, stays through the winter, and thins for half a day
// after.
func (g *Game) fallAsh() float64 {
	after := g.greatFall() - fallLasts // seconds since the Skyspear struck
	switch winter := winterDays * dayLength; {
	case after < 0:
		return 0
	case after < winter:
		return winterAsh * min(after/ashClimb, 1)
	default:
		return winterAsh * max(1-(after-winter)/(ashAfter*dayLength), 0)
	}
}

// streaks returns how many shooting stars a minute streak the sky tonight: more the
// nearer the comet is at this night's midnight. It is the same all through one night,
// as the sky works out which streak is in it from the clock and this number.
func (g *Game) streaks() float64 {
	near, _ := cometAt(math.Floor(g.days+firstMorning+0.5) - firstMorning)
	return streaksAMinute + (streaksMost-streaksAMinute)*near*near
}

// starFall returns the day of the world on which the star of a night falls, and whether
// one falls on that night at all. Nights are counted from 1, and a star falls in the
// dark, round the midnight of its night. One falls about one night in three, and on
// every one of the last nights before the comet arrives.
func (g *Game) starFall(night int) (when float64, ok bool) {
	to := ((cometNight-night)%cometEvery + cometEvery) % cometEvery // nights until the comet arrives
	part, ok := g.world.StarFall(night, to >= 1 && to <= cometFalls)
	return float64(night) - firstMorning + (part-0.5)*starSpan, ok
}

// nextStar returns the day on which a star next falls, looking a month of nights ahead.
func (g *Game) nextStar() (when float64, ok bool) {
	tonight := int(math.Floor(g.days + firstMorning + 0.5))
	for night := tonight; night < tonight+30; night++ {
		if when, ok := g.starFall(night); ok && when > g.days {
			return when, true
		}
	}
	return 0, false
}

// stars runs the falling stars for dt seconds: the flash of one that struck fades, its
// boom comes when it has crossed the land, and a star whose moment has come falls. One
// whose moment the clock was moved past still falls if it is night, and one slept
// through does not. In the Great Fall they come one after another, and at its end the
// Skyspear strikes.
func (g *Game) stars(dt float64) {
	g.glare, g.white = max(g.glare-dt, 0), max(g.white-dt, 0)
	if g.shock > 0 {
		g.shock = max(g.shock-dt, 0)
		g.quake = max(g.quake, min(1.5*g.shock/shockTime, 1))
	}
	kept := g.booms[:0]
	for _, b := range g.booms {
		switch b.left -= dt; {
		case b.left > 0:
			kept = append(kept, b)
		case b.great:
			g.ears.PlayFar(sound.Boom, b.at, 1, 2*b.at.Distance(g.player.Pos))
			g.ears.PlayNear(sound.Tremor, 1)
			g.shock = shockTime
		default:
			g.ears.PlayFar(sound.Boom, b.at, 1, boomHeard)
			if far := b.at.Distance(g.player.Pos); far < boomFelt {
				g.quake = max(g.quake, math.Pow(1-far/boomFelt, 2))
			}
		}
	}
	g.booms = kept

	from := g.fallen
	g.fallen = g.days
	switch since := g.greatFall(); {
	case since >= 0 && since < fallLasts:
		if math.Floor(since/fallEvery) != math.Floor((since-dt)/fallEvery) {
			g.dropStar(0, fallFar)
		}
		return
	case since >= fallLasts && since-(g.days-from)*dayLength < fallLasts:
		g.skyspear()
		return
	}
	when, ok := g.starFall(int(math.Floor(g.days + firstMorning + 0.5)))
	if at := timeOfDay(g.days); !ok || when <= from || when > g.days || at > 0.2 && at < 0.8 {
		return
	}
	g.dropStar(starNear, starFar)
}

// skyspear has the Skyspear itself strike, far off, where it stood in the sky: the sky
// goes white, and its boom is on its way.
func (g *Game) skyspear() {
	g.white, g.glare = whiteTime, flashTime
	g.booms = append(g.booms, boom{
		at: g.player.Pos.Add(geometry.Vector3{Z: skyspearBoom * boomPace}), left: skyspearBoom, great: true,
	})
}

// dropStar lets a star fall at a place where one may strike, so far from the player and
// no farther. It comes across the sky, from a side of that place as the player sees it
// and never from the player's side of it: so what it meets on its way down is no nearer
// the player than where it was aimed. With no place to strike, none falls.
func (g *Game) dropStar(near, most float64) {
	p, r := g.player.Pos, g.scatter
	for range starTries {
		bearing, far := 2*math.Pi*r.Float64(), near+(most-near)*r.Float64()
		x, z := int(math.Floor(p.X+far*math.Cos(bearing))), int(math.Floor(p.Z+far*math.Sin(bearing)))
		if !g.starMayStrike(x, z) {
			continue
		}
		to := cellMid(x, g.firm(x, z)+1, z)
		out := bearing + (0.9+0.6*r.Float64())*float64(1-2*r.IntN(2)) // 50 to 86 degrees to one side
		from := to.Add(geometry.Vector3{X: starAside * math.Cos(out), Y: starHigh, Z: starAside * math.Sin(out)})
		g.missiles = append(g.missiles, missile{
			kind: starFallen, stack: world.Stack{Item: world.Lava.Item(), Count: 1},
			pos: from, vel: to.Sub(from).Scale(1 / starFlight),
		})
		return
	}
}

// soft reports whether a star goes through a block: what is no solid thing, and leaves,
// wood and glass.
func soft(b world.Block) bool {
	info := b.Info()
	return !info.Solid || info.Tool == world.Axe || info.Shade < world.MaxLight
}

// firm returns the layer of the ground in a column: its highest block that a star does
// not go through.
func (g *Game) firm(x, z int) int {
	y := g.world.Ground(x, z)
	for y > 0 && soft(g.world.At(x, y, z)) {
		y--
	}
	return y
}

// starMayStrike reports whether a star may make its crater in the column at x, z: on dry
// land that is loaded, deep enough for one, and not near the bed, a pot, a hearth or a
// fire mountain.
func (g *Game) starMayStrike(x, z int) bool {
	near := func(c world.Cell, reach float64) bool {
		return math.Hypot(float64(x-c.X), float64(z-c.Z)) < reach
	}
	if !g.world.Loaded(x, z) || g.bed != nil && near(*g.bed, bedSafe+bowlWide+1) {
		return false
	}
	if _, far, ok := g.world.Mountain(float64(x), float64(z)); ok && far < mountSafe {
		return false
	}
	for c := range g.pots {
		if near(c, storeSafe) {
			return false
		}
	}
	for c := range g.hearths {
		if near(c, storeSafe) {
			return false
		}
	}
	y := g.firm(x, z)
	over := g.world.At(x, y+1, z)
	return y >= bowlDeep && over != world.Water && over != world.Lava
}

// fireballs returns what the sky is to show of the falling stars: for each, the ways from
// the player's eyes to it and to the end of its tail; and how bright they are. With none
// in the sky the brightness is what is left of the flash of one that struck.
func (g *Game) fireballs() (ways [][2]geometry.Vector3, bright float64) {
	eye := g.player.head()
	g.balls = g.balls[:0]
	for _, m := range g.missiles {
		if fliers[m.kind].falls {
			g.balls = append(g.balls, [2]geometry.Vector3{m.pos.Sub(eye), m.pos.Sub(m.vel.Scale(starTail)).Sub(eye)})
		}
	}
	if len(g.balls) > 0 {
		return g.balls, 1
	}
	return nil, g.glare / flashTime
}

// impact has a star come down at a place. It is seen at once and heard when its boom
// has crossed the land, and the beasts near run from it. Where it may, it hurts what is
// there and makes its crater; over water, or what must be kept, it burns up and leaves
// nothing. What is deeper under where it struck than its crater goes is not hurt: three
// blocks of earth or stone overhead keep a star off.
func (g *Game) impact(at geometry.Vector3) {
	g.glare = flashTime
	g.booms = append(g.booms, boom{at: at, left: at.Distance(g.player.Pos) / boomPace})
	g.fauna.drive(at, fallScare, fallRun)
	x, z := int(math.Floor(at.X)), int(math.Floor(at.Z))
	if !g.starMayStrike(x, z) {
		return
	}
	y := g.firm(x, z)
	in := func(p geometry.Vector3) bool {
		return math.Hypot(p.X-at.X, p.Z-at.Z) <= bowlWide+1 && p.Y > float64(y-bowlDeep)-0.5 && p.Y < float64(y)+blastHigh
	}
	for i := range g.fauna.all {
		if c := &g.fauna.all[i]; !c.dead && in(c.Pos) {
			c.hit(at, blastHarm)
		}
	}
	if g.survival && in(g.player.Pos) {
		g.hurt(blastHarm)
	}
	g.crater(x, y, z)
}

// crater cuts the crater of a star whose middle met the ground at a cell: a bowl, with
// all that stood over it gone, its floor burnt to dark rock, or to glass where it and
// the rim were sand, fires round it that burn a day, and under its middle the core, hot
// for a while.
func (g *Game) crater(x, y, z int) {
	w := g.world
	const reach = 5 // how far from its middle the columns of a crater are looked at
	for dz := -reach; dz <= reach; dz++ {
		for dx := -reach; dx <= reach; dx++ {
			flat := math.Hypot(float64(dx), float64(dz))
			floor := y + 1 // the lowest cell of the column that is open
			for cy := max(w.Surface(x+dx, z+dz), y); cy > y-bowlDeep; cy-- {
				// Under the rim the bowl narrows as it deepens, and all that stood over it
				// is gone.
				dy := float64(min(cy-y, 0))
				if flat <= bowlWide*math.Sqrt(1-dy*dy/((bowlDeep+0.5)*(bowlDeep+0.5))) {
					w.Set(x+dx, cy, z+dz, world.Air)
					floor = cy
				}
			}
			switch under := w.At(x+dx, floor-1, z+dz); {
			case flat > bowlWide+1.5:
			case under == world.Sand:
				w.Set(x+dx, floor-1, z+dz, world.Glass) // sand the heat reached
			case flat <= bowlWide && under.Info().Solid && !under.Info().Fixed:
				w.Set(x+dx, floor-1, z+dz, world.Basalt) // and the floor of the bowl is burnt dark
			}
		}
	}
	// The core lies in the ground under the bowl. Where the star broke through a roof
	// into the open, it lies in the floor under that.
	for cy := y - bowlDeep; cy > max(y-bowlDeep-coreFalls, 0); cy-- {
		if b := w.At(x, cy, z).Info(); b.Solid && !b.Fixed {
			w.Set(x, cy, z, world.HotCore)
			g.embers = append(g.embers, world.Ember{At: world.Cell{X: x, Y: cy, Z: z}, Left: coreHot, Into: world.SkyIronCore})
			break
		}
	}

	turn := 2 * math.Pi * g.scatter.Float64()
	for i := range craterFires {
		bearing := turn + 2*math.Pi*float64(i)/craterFires
		fx, fz := x+int(math.Round((bowlWide+1.5)*math.Cos(bearing))), z+int(math.Round((bowlWide+1.5)*math.Sin(bearing)))
		// On the ground there, at about the height of the rim: not up in a tree.
		for fy := y + 3; fy >= y-2; fy-- {
			if w.At(fx, fy, fz) == world.Air && w.At(fx, fy-1, fz).Info().Solid {
				w.Set(fx, fy, fz, world.Fire)
				g.embers = append(g.embers, world.Ember{At: world.Cell{X: fx, Y: fy, Z: fz}, Left: starFire})
				break
			}
		}
	}
	g.shatter(x, y, z, w.At(x, y-bowlDeep-1, z))
	for range 6 {
		g.puff(cellMid(x, y, z), 1.2, smokeLife)
	}
}

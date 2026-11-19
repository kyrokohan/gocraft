package game

import (
	"math"
	"math/rand/v2"
	"slices"

	"glonk.dev/geometry"
	"glonk.dev/mesh"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/sound"
	"github.com/kyrokohan/skyspear/internal/world"
)

// How beasts live. Distances are in blocks, times in seconds.
const (
	creatureCount = 12    // mossbacks round the player
	creatureSpeed = 1.3   // a mossback, walking
	creatureRange = 48.0  // how far from the player beasts start, at the most
	creatureAwake = 88.0  // how far from the player they still move: the ground is loaded that far
	creatureGone  = 104.0 // how far from the player they are let go
	creatureHop   = 8.0   // the speed a beast leaves the ground at: enough to clear one block
	lookAhead     = 0.35  // how far past its own front it looks before it steps
	longDrop      = 3     // blocks: it does not walk off anything deeper
	rethink       = 0.4   // how long it stands after something turns it back

	fleeTime   = 3.0   // how long it runs from what hit it
	knockSpeed = 5.0   // how fast a hit sends it away, at first
	knockHop   = 4.0   // and up
	knockDrag  = 6.0   // how fast that slows, a part of it a second
	hurtShow   = 0.3   // how long it shows a hit
	mossRegrow = 300.0 // how long a mossback's moss takes to grow back once it is shorn

	fallTime   = 0.8 // how long a beast that has died takes to sink and fade
	herdReach  = 6.0 // how far from the middle of its herd a beast strays before it turns back
	groupReach = 5   // how far from the place found for a group its members start
	alarmTime  = 6.0 // how long a herd runs from what alarmed it
	sneakPart  = 0.4 // how much of the distance a herd takes alarm at is left for a crouched player
	bumpWait   = 1.0 // between two hurts from one beast running into the player
	bumpShove  = 0.8 // how far that throws the player
)

// temper is how a kind of beast behaves.
type temper uint8

const (
	loner  temper = iota // it goes its own way: it stands, walks and stands
	herder               // it keeps with its herd, and runs with it from what alarms it
)

// breed is a kind of beast as it lives: a row of the table. How it looks, and the box
// that blocks stop it in, is the renderer's.
type breed struct {
	walk, run float64 // how fast it goes, walking and running
	gait      float64 // how far its legs swing for each block it goes, in radians

	// health is how many points of harm kill it. A beast with none cannot be hurt: a
	// hit only drives it off.
	health float64

	// bump is the points a running one takes from a player it runs into.
	bump float64

	temper temper
	group  [2]int  // the fewest and the most that are found together
	most   int     // of its kind round the player
	young  float64 // the part of a group that is not grown

	// wary is how near a standing player may come before its herd takes alarm.
	wary float64

	// lands are the lands it is found in. A beast with none is found wherever there is
	// mossy earth.
	lands []world.Biome

	meat int // pieces of raw meat it leaves

	// call is the sound it makes to itself, about once in callEvery seconds, and cry
	// the one it makes when it is hit or takes alarm. carries is how far off they are
	// heard, for a beast whose voice carries farther than other sounds do.
	call, cry sound.Name
	callEvery float64
	carries   float64
}

// breeds is every kind of beast.
var breeds = [render.Kinds]breed{
	render.Mossback: {
		walk: creatureSpeed, run: 3.4, gait: 6.15,
		group: [2]int{1, 1}, most: creatureCount,
		call: sound.Call, cry: sound.Cry, callEvery: callEvery,
	},
	render.Bugler: {
		walk: 1.6, run: 6.5, gait: 2.6,
		health: 16, bump: 3, meat: 3,
		temper: herder, group: [2]int{5, 8}, most: 16, young: 0.3, wary: 9,
		lands: []world.Biome{world.Prairie, world.Highland},
		call:  sound.Horn, cry: sound.Alarm, callEvery: 70, carries: 3 * sound.Hearing,
	},
}

// beast is one animal: it walks about, stands a while, and walks on. It is stopped by
// blocks as the player is, hops up a single block, and turns back from walls, water and
// long drops. What more it does is its breed's.
type beast struct {
	kind render.Kind

	// Pos is the middle of the bottom of its body: the point between its feet.
	Pos geometry.Vector3

	// Yaw is the way it faces, as a camera's Yaw is.
	Yaw float64

	// Stride is how far through a step its legs are, in radians, and 0 when it stands.
	Stride float64

	// OnGround is true while it stands on a block.
	OnGround bool

	young  bool    // not grown: half the size
	herd   int     // which herd it goes with, or 0 for none
	health float64 // points of harm it can still take, for a breed that has health

	velY    float64    // speed upward
	walking bool       // or standing
	timer   float64    // until it changes its mind
	rng     *rand.Rand // its own, so that what it does can be told from a seed

	push   geometry.Vector3 // the way a hit sends it, and how fast, along the ground
	flee   float64          // how long it still runs
	hurt   float64          // how long it still shows a hit
	regrow float64          // how long a mossback's moss still takes to grow back, or 0 if it has it
	bumped float64          // until it can hurt the player again by running into it

	dead bool    // it has no health left
	fall float64 // from 0, as it dies, to 1, gone
}

// said is a sound a beast of a kind makes at a place.
type said struct {
	name sound.Name
	kind render.Kind
	at   geometry.Vector3
}

// alarm is a herd running from something.
type alarm struct {
	left float64          // how long it still runs
	from geometry.Vector3 // what it runs from
}

// fauna is the beasts round the player. The world has no end, so they cannot all be
// placed when it is made: each kind is found places near the player as it moves, a
// group at a time, up to as many as its breed says, and those left far behind are let go.
type fauna struct {
	all   []beast
	rng   *rand.Rand // where the next one goes: from the world's seed
	born  uint64     // how many there have been, to give each a generator of its own
	herds int        // how many herds there have been, to number the next
	timer float64    // until the beasts are next looked over

	// sneaking is whether the player is crouched, which lets it come nearer a herd.
	sneaking bool

	alarms map[int]alarm

	// What the last step left for the game to act on: the sounds the beasts made, and
	// the beasts that died and are gone.
	sounds []said
	fallen []beast
}

func emptyFauna(w *world.World) *fauna {
	return &fauna{rng: rand.New(rand.NewPCG(uint64(w.Seed), 0x6d6f7373)), alarms: map[int]alarm{}}
}

// newFauna returns the beasts of a world round a place, as many of each kind as there
// is room for, up to what its breed says. The same world and place always give the same.
func newFauna(w *world.World, round geometry.Vector3) *fauna {
	f := emptyFauna(w)
	for k := range render.Kinds {
		for try := 0; f.count(k) < breeds[k].most && try < 60*breeds[k].most; try++ {
			f.add(w, k, round, 4, creatureRange)
		}
	}
	return f
}

// faunaFrom returns the beasts a save holds, or new ones round a place if it holds
// none. A save from before there were kinds holds mossbacks only.
func faunaFrom(w *world.World, s world.Meta, round geometry.Vector3) *fauna {
	if len(s.Beasts) == 0 && len(s.Mossbacks) == 0 {
		return newFauna(w, round)
	}
	f := emptyFauna(w)
	own := func() *rand.Rand {
		f.born++
		return rand.New(rand.NewPCG(uint64(w.Seed), f.born))
	}
	for _, m := range s.Mossbacks {
		r := own()
		f.all = append(f.all, beast{Pos: m.Pos, Yaw: m.Yaw, regrow: m.Regrow, timer: r.Float64() * 2, rng: r})
	}
	for _, b := range s.Beasts {
		kind, ok := render.KindNamed(b.Kind)
		if !ok {
			continue // a kind of a later version of the game
		}
		r := own()
		f.all = append(f.all, beast{
			kind: kind, Pos: b.Pos, Yaw: b.Yaw, young: b.Young, herd: b.Herd,
			health: min(max(b.Health, 1), max(breeds[kind].health, 1)), regrow: b.Regrow,
			timer: r.Float64() * 2, rng: r,
		})
		f.herds = max(f.herds, b.Herd)
	}
	return f
}

// saved returns the beasts as a save holds them. One that is dying is not kept.
func (f *fauna) saved() []world.Beast {
	out := make([]world.Beast, 0, len(f.all))
	for _, c := range f.all {
		if !c.dead {
			out = append(out, world.Beast{
				Kind: c.kind.String(), Pos: c.Pos, Yaw: c.Yaw, Young: c.young, Herd: c.herd,
				Health: c.health, Regrow: c.regrow,
			})
		}
	}
	return out
}

// count returns how many beasts of a kind there are.
func (f *fauna) count(k render.Kind) (n int) {
	for i := range f.all {
		if f.all[i].kind == k {
			n++
		}
	}
	return n
}

// standing returns where a beast of a kind could stand in the column at x, z: on mossy
// earth, in a land of its breed, with room over it for its body, and not on top of
// another beast.
func (f *fauna) standing(w *world.World, k render.Kind, young bool, x, z int) (geometry.Vector3, bool) {
	y := w.Ground(x, z) // under any plant
	if w.At(x, y, z) != world.Turf {
		return geometry.Vector3{}, false // not mossy earth, or a place that is not loaded
	}
	if lands := breeds[k].lands; lands != nil && !slices.Contains(lands, world.BiomeAt(w.Seed, x, z)) {
		return geometry.Vector3{}, false
	}
	_, height := render.Size(k, young)
	for up := 1; up <= int(math.Ceil(height))+1; up++ {
		if y+up >= world.Height || w.At(x, y+up, z).Info().Solid || (up > 1 && w.At(x, y+up, z) != world.Air) {
			return geometry.Vector3{}, false // under a tree
		}
	}
	pos := geometry.Vector3{X: float64(x) + 0.5, Y: float64(y + 1), Z: float64(z) + 0.5}
	// One that goes alone keeps three blocks from any other. Those of a group stand
	// close, with room to move.
	gap := 3.0
	if width, _ := render.Size(k, young); breeds[k].group[1] > 1 {
		gap = width + 0.4
	}
	for i := range f.all {
		if f.all[i].Pos.Distance(pos) < gap {
			return geometry.Vector3{}, false
		}
	}
	return pos, true
}

// add tries one place for a group of a kind, between near and far from a point. It
// reports whether any of the group found room there.
func (f *fauna) add(w *world.World, k render.Kind, round geometry.Vector3, near, far float64) bool {
	b := &breeds[k]
	angle, away := f.rng.Float64()*2*math.Pi, near+f.rng.Float64()*(far-near)
	x := int(math.Floor(round.X + math.Cos(angle)*away))
	z := int(math.Floor(round.Z + math.Sin(angle)*away))
	if _, ok := f.standing(w, k, false, x, z); !ok {
		return false
	}

	size := b.group[0]
	if more := b.group[1] - b.group[0]; more > 0 {
		size += f.rng.IntN(more + 1)
	}
	size = min(size, b.most-f.count(k))
	herd := 0
	if b.group[1] > 1 {
		f.herds++
		herd = f.herds
	}
	placed := 0
	for try := 0; placed < size && try < 12*size; try++ {
		mx, mz := x, z
		if placed > 0 { // the first stands on the place itself, and the others round it
			mx += f.rng.IntN(2*groupReach+1) - groupReach
			mz += f.rng.IntN(2*groupReach+1) - groupReach
		}
		young := placed > 0 && f.rng.Float64() < b.young
		pos, ok := f.standing(w, k, young, mx, mz)
		if !ok {
			continue
		}
		f.born++
		own := rand.New(rand.NewPCG(uint64(w.Seed), f.born))
		f.all = append(f.all, beast{
			kind: k, Pos: pos, young: young, herd: herd, health: b.health,
			Yaw:   own.Float64()*2*math.Pi - math.Pi,
			timer: own.Float64() * 2,
			rng:   own,
		})
		placed++
	}
	return placed > 0
}

// middles returns the middle of every herd: where its living members are, on the whole.
func (f *fauna) middles() map[int]geometry.Vector3 {
	sum, n := map[int]geometry.Vector3{}, map[int]float64{}
	for i := range f.all {
		if c := &f.all[i]; c.herd != 0 && !c.dead {
			sum[c.herd], n[c.herd] = sum[c.herd].Add(c.Pos), n[c.herd]+1
		}
	}
	for herd, total := range sum {
		sum[herd] = total.Scale(1 / n[herd])
	}
	return sum
}

// raise alarms a herd, if it is not alarmed already: all of it run from the place the
// alarm came from. It reports whether the herd took alarm now.
func (f *fauna) raise(herd int, from geometry.Vector3) bool {
	if herd == 0 || f.alarms[herd].left > 0 {
		return false
	}
	if f.alarms == nil {
		f.alarms = map[int]alarm{}
	}
	f.alarms[herd] = alarm{left: alarmTime, from: from}
	for i := range f.all {
		c := &f.all[i]
		if c.herd != herd || c.dead {
			continue
		}
		away := awayFrom(c.Pos, from, c.Yaw)
		c.flee, c.walking = alarmTime, true
		c.Yaw = yawOf(away) + (c.rng.Float64()-0.5)*0.6 // each a little its own way
	}
	return true
}

// step moves the beasts for dt seconds, and now and then looks them over: those far
// behind the player are let go, and a new group is found a place ahead if there are too
// few of a kind. Those too far off to stand on loaded ground are not moved.
//
// It leaves the cries of the herds that took alarm in sounds, and the beasts that died
// and are gone in fallen, for the game to play and to put down what they leave.
func (f *fauna) step(w *world.World, player geometry.Vector3, dt float64) {
	f.sounds, f.fallen = f.sounds[:0], f.fallen[:0]
	for herd, a := range f.alarms {
		if a.left -= dt; a.left <= 0 {
			delete(f.alarms, herd)
		} else {
			f.alarms[herd] = a
		}
	}

	middles := f.middles()
	for i := range f.all {
		c := &f.all[i]
		if c.Pos.Distance(player) >= creatureAwake {
			continue
		}
		b := &breeds[c.kind]
		if wary := b.wary; wary > 0 && !c.dead {
			if f.sneaking {
				wary *= sneakPart
			}
			if c.Pos.Distance(player) < wary && f.raise(c.herd, player) {
				f.sounds = append(f.sounds, said{b.cry, c.kind, c.voice()}) // the one that saw it calls out
			}
		}
		var middle *geometry.Vector3
		if m, ok := middles[c.herd]; ok && b.temper == herder {
			middle = &m
		}
		c.step(w, dt, middle)
	}
	f.all = slices.DeleteFunc(f.all, func(c beast) bool {
		if c.dead && c.fall >= 1 {
			f.fallen = append(f.fallen, c)
			return true
		}
		return false
	})

	if f.timer -= dt; f.timer > 0 {
		return
	}
	f.timer = 1
	f.all = slices.DeleteFunc(f.all, func(c beast) bool {
		return c.Pos.Distance(player) > creatureGone
	})
	for k := range render.Kinds {
		if f.count(k) < breeds[k].most {
			f.add(w, k, player, creatureRange/2, creatureAwake-8) // out of the way, where it will be met
		}
	}
}

// voice returns where a beast's sounds come from: its head, more or less.
func (c *beast) voice() geometry.Vector3 {
	_, height := render.Size(c.kind, c.young)
	return c.Pos.Add(geometry.Vector3{Y: 0.85 * height})
}

// Pose returns where the beast is and how it stands, for drawing.
func (c *beast) Pose() render.Beast {
	return render.Beast{
		Kind: c.kind, Pos: c.Pos, Yaw: c.Yaw, Stride: c.Stride,
		Shorn: c.regrow > 0, Hurt: c.hurt > 0, Young: c.young, Fall: c.fall,
	}
}

// hit is the beast being hit by the player standing at from, by points of harm. It is
// knocked away, and runs. A beast with health loses the points, and with none left it
// dies. It reports whether the hit killed it.
func (c *beast) hit(from geometry.Vector3, points float64) (killed bool) {
	if c.dead {
		return false
	}
	away := awayFrom(c.Pos, from, c.Yaw)
	c.push = away.Scale(knockSpeed)
	if c.OnGround {
		c.velY = knockHop
	}
	c.hurt, c.flee = hurtShow, fleeTime
	c.walking, c.Yaw = true, yawOf(away)
	if breeds[c.kind].health > 0 {
		if c.health -= points; c.health <= 0 {
			c.dead, c.walking, c.flee = true, false, 0
			return true
		}
	}
	return false
}

// shear takes a mossback's moss, if it has any, and reports whether it had.
func (c *beast) shear() bool {
	if c.kind != render.Mossback || c.regrow > 0 || c.dead {
		return false
	}
	c.regrow = mossRegrow
	return true
}

// awayFrom returns the way along the ground from a point to another, one long: the way a
// hit at from sends what is at pos. Where the two are one above the other, it is the
// way yaw faces.
func awayFrom(pos, from geometry.Vector3, yaw float64) geometry.Vector3 {
	away := pos.Sub(from)
	away.Y = 0
	if away.Length() < 1e-6 {
		return mesh.Camera{Yaw: yaw}.Forward()
	}
	return away.Normalize()
}

// yawOf returns the Yaw that faces a way along the ground.
func yawOf(way geometry.Vector3) float64 {
	return math.Atan2(way.X, way.Z)
}

// decide changes the beast's mind when its time is up: it stands for one to three
// seconds, or turns some new way and walks for two to five. One that has strayed from
// the middle of its herd turns back toward it.
func (c *beast) decide(middle *geometry.Vector3) {
	if c.walking {
		c.walking, c.timer = false, 1+c.rng.Float64()*2
		return
	}
	c.walking, c.timer = true, 2+c.rng.Float64()*3
	c.Yaw = c.rng.Float64()*2*math.Pi - math.Pi
	if middle != nil {
		reach := herdReach
		if c.young {
			reach /= 2 // the young keep to the middle
		}
		if to := middle.Sub(c.Pos); math.Hypot(to.X, to.Z) > reach {
			c.Yaw = yawOf(to) + (c.rng.Float64()-0.5)*0.8
		}
	}
}

// turnBack stops the beast where something is in its way. It stands a moment and then
// picks a new way to go, which is what makes it look as if it thought about it.
func (c *beast) turnBack() {
	c.walking, c.timer = false, rethink
}

// wayAhead looks at the cell just past the beast's front and says what to do about it:
// walk on, hop up onto it, or turn back.
func (c *beast) wayAhead(w *world.World, ahead geometry.Vector3, afloat bool) (hop, blocked bool) {
	width, height := render.Size(c.kind, c.young)
	front := c.Pos.Add(ahead.Scale(width/2 + lookAhead))
	x, y, z := int(math.Floor(front.X)), int(math.Floor(c.Pos.Y+0.1)), int(math.Floor(front.Z))

	if solid(w, x, y, z) {
		// A block at the height of its feet. One block is a step it can hop, if it has
		// its feet on something and there is room on top for the whole of it.
		for up := 1; up <= int(math.Ceil(height)); up++ {
			if solid(w, x, y+up, z) {
				return false, true
			}
		}
		return c.OnGround || afloat, true
	}
	if w.At(x, y, z) == world.Water || w.At(x, y-1, z) == world.Water {
		return false, true
	}
	for down := 1; down <= longDrop; down++ {
		if solid(w, x, y-down, z) {
			return false, false // ground to step onto or down to
		}
	}
	return false, true // nothing under it for a long way
}

// Step moves a beast that goes alone for dt seconds.
func (c *beast) Step(w *world.World, dt float64) {
	c.step(w, dt, nil)
}

// step moves the beast for dt seconds. middle is the middle of its herd, or nil.
func (c *beast) step(w *world.World, dt float64, middle *geometry.Vector3) {
	b := &breeds[c.kind]
	width, height := render.Size(c.kind, c.young)
	c.hurt = max(c.hurt-dt, 0)
	c.regrow = max(c.regrow-dt, 0)
	c.bumped = max(c.bumped-dt, 0)
	if c.dead {
		c.fall = min(c.fall+dt/fallTime, 1)
		c.Stride = 0
	}
	speed := b.walk
	if c.flee > 0 {
		c.flee -= dt
		c.walking, c.timer, speed = true, max(c.timer, rethink), b.run
	}
	if c.timer -= dt; c.timer <= 0 && !c.dead {
		c.decide(middle)
	}

	afloat := inWater(w, c.Pos)
	if afloat {
		c.velY = 1.5 // it floats up, should it ever fall in
	} else {
		c.velY = max(c.velY-gravity*dt, -fallSpeed)
	}

	var move geometry.Vector3
	if c.walking && !c.dead {
		ahead := mesh.Camera{Yaw: c.Yaw}.Forward()
		switch hop, blocked := c.wayAhead(w, ahead, afloat); {
		case hop:
			c.velY = creatureHop
		case blocked && (c.OnGround || afloat) && c.flee > 0:
			c.Yaw += math.Pi / 2 * float64(1-2*c.rng.IntN(2)) // running, it turns aside
		case blocked && (c.OnGround || afloat):
			c.turnBack() // in the air it is in the middle of a hop, and goes on
		}
		if c.walking {
			move = ahead.Scale(speed * dt)
			c.Stride += b.gait * speed * dt
		}
	}
	if !c.walking {
		c.Stride = 0
	}
	move = move.Add(c.push.Scale(dt))
	c.push = c.push.Scale(max(1-knockDrag*dt, 0))
	move.Y = c.velY * dt

	stopped := travel(w, &c.Pos, width, height, move)
	c.OnGround = stopped[1] && move.Y < 0
	if stopped[1] {
		c.velY = 0
	}
	if (stopped[0] || stopped[2]) && c.walking && c.OnGround {
		c.turnBack() // it walked into something the look ahead did not see: a corner
	}
}

// voice plays a sound a beast makes, from its head.
func (g *Game) voice(c *beast, name sound.Name, gain float64) {
	g.sayFrom(c.kind, name, c.voice(), gain)
}

// sayFrom plays a sound a beast of a kind makes at a place, as far off as its voice
// carries.
func (g *Game) sayFrom(kind render.Kind, name sound.Name, at geometry.Vector3, gain float64) {
	if far := breeds[kind].carries; far > 0 {
		g.ears.PlayFar(name, at, gain, far)
		return
	}
	g.ears.Play(name, at, gain)
}

// meatSpread is how far from where a beast died its meat lies, at the most.
const meatSpread = 0.6

// stepBeasts moves the beasts for dt seconds, plays what they said, puts down what the
// dead leave, and has a beast that runs into the player hurt it.
func (g *Game) stepBeasts(dt float64) {
	f := g.fauna
	f.sneaking = g.player.Crouched
	f.step(g.world, g.player.Pos, dt)
	for _, s := range f.sounds {
		g.sayFrom(s.kind, s.name, s.at, 0.9)
	}
	for _, c := range f.fallen {
		for range breeds[c.kind].meat {
			at := c.Pos.Add(geometry.Vector3{
				X: (g.scatter.Float64() - 0.5) * 2 * meatSpread, Y: 0.3, Z: (g.scatter.Float64() - 0.5) * 2 * meatSpread,
			})
			g.drops = append(g.drops, drop{Drop: world.Drop{Stack: world.Stack{Item: world.Meat, Count: 1}, Pos: at}})
		}
	}
	if !g.survival {
		return // nothing hurts in creative
	}
	lo, hi := boxOf(g.player.Pos, playerWidth, playerHeight)
	for i := range f.all {
		c := &f.all[i]
		b := &breeds[c.kind]
		if b.bump == 0 || c.flee <= 0 || c.bumped > 0 || c.dead {
			continue
		}
		width, height := render.Size(c.kind, c.young)
		clo, chi := boxOf(c.Pos, width, height)
		if lo[0] >= chi[0] || hi[0] <= clo[0] || lo[1] >= chi[1] || hi[1] <= clo[1] || lo[2] >= chi[2] || hi[2] <= clo[2] {
			continue
		}
		// It runs the player down: a hurt, and a throw the way it runs.
		c.bumped = bumpWait
		travel(g.world, &g.player.Pos, playerWidth, playerHeight, mesh.Camera{Yaw: c.Yaw}.Forward().Scale(bumpShove))
		g.hurt(b.bump)
		if g.screen != playing {
			return // the player fainted
		}
	}
}

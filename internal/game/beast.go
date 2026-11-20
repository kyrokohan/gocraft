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

	thiefReach = 0.9  // how near a thief comes to take what the player holds
	robWait    = 8.0  // how long after a theft before the next
	chewTime   = 0.35 // how long a beast chews what it was fed before it takes more
	followFar  = 5.0  // how far from the player a tame beast lets itself fall behind
	followRun  = 10.0 // and how far before it runs to catch up
)

// temper is how a kind of beast behaves.
type temper uint8

const (
	loner  temper = iota // it goes its own way: it stands, walks and stands
	herder               // it keeps with its herd, and runs with it from what alarms it
	thief                // it keeps with its flock, runs as a herd does, and comes for food held out
	hunter               // it keeps with its pack, and goes for the player and for beasts it can kill
	guard                // it stands its ground, and turns on what comes near its young or itself
)

// breed is a kind of beast as it lives: a row of the table. How it looks, and the box
// that blocks stop it in, is the renderer's.
type breed struct {
	walk, run float64 // how fast it goes, walking and running
	gait      float64 // how far its legs swing for each block it goes, in radians

	// health is how many points of harm kill it. A beast with none cannot be hurt: a
	// hit only drives it off.
	health float64

	// bump is the points a running one takes from a player it runs into, and bite the
	// points a hunter's bite takes from what it has caught.
	bump, bite float64

	// visits is true for a hunter that comes and goes, and is not kept with the world.
	// sees and hears are how far a hunter notices a player who stands, and one who
	// moves.
	visits      bool
	sees, hears float64

	// How a hunter that visits comes and goes. from is the day of the world, counted
	// from its first midnight, before which none comes or takes notice of the player.
	// comes is how far from the player it comes out, at the least and the most. bears is
	// the brightest light it steps into and comes out in: 0 for one that light does not
	// hold. fears is how near a big fire it comes: 0 for one that keeps out of all light
	// anyway. stays is how many seconds it is about before it goes: 0 for one that goes
	// with the light. every is how many days after one has gone the next comes, at the
	// least and the most: none, for one that comes whenever there is room for it.
	from         float64
	comes, every [2]float64
	bears, fears float64
	stays        float64

	// stalks is how far a hunter sees a beast to hunt, snaps how long it waits between
	// two bites, and feeds how long it stands over a kill. cows is true for one that
	// lesser hunters run from.
	stalks, snaps, feeds float64
	cows                 bool

	temper temper
	group  [2]int  // the fewest and the most that are found together
	most   int     // of its kind round the player
	young  float64 // the part of a group that is not grown

	// wary is how near a standing player may come before its herd takes alarm, and
	// lure how far a thief comes from for food the player holds.
	wary, lure float64

	// eats is what it takes from the player's hand, and meals how many of them make it
	// trust the player: a beast with none cannot be tamed.
	eats  []world.Item
	meals int

	// lands are the lands it is found in. A beast with none is found wherever there is
	// mossy earth.
	lands []world.Biome

	meat, hide int // pieces of raw meat, and hides, it leaves

	// A guard does not run. warns is how near what it guards, its young or itself, a
	// threat may come before it turns on it and gives its cry. charges is how near
	// before it runs at the threat, for one that charges, and swings how far from the
	// middle of its tail that reaches, for one that strikes with it. blow is the points
	// its charge or its tail takes.
	warns, charges, swings, blow float64

	// A beast that is calm is moved by nothing: a hit does not hurt it, knock it back or
	// make it run. One that wades walks through the leaves of trees. stamp is the
	// points its foot takes from a player it comes down on, and foot how far aside of
	// its middle, and how far ahead and behind, its feet come down. quake is how far
	// off its steps shake the player's view: they are heard as far as its voice carries.
	calm, wades bool
	stamp       float64
	foot        [2]float64
	quake       float64

	// heavy is true for a beast that breaks through a pit cover, and that a long fall
	// hurts: the bigger it is, the more.
	heavy bool

	armor float64 // points taken off every hit it takes
	steps int     // how many blocks high a step it climbs: one, if it says none
	grown int     // the first so many of a group are grown, and the rest young: by chance, if it says none
	bare  bool    // it stands on bare rock and sand as well as on mossy earth

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
		eats: []world.Item{world.Berries, world.Fruit}, meals: 3,
		call: sound.Call, cry: sound.Cry, callEvery: callEvery,
	},
	render.Bugler: {
		walk: 1.6, run: 6.5, gait: 2.6,
		health: 16, bump: 3, meat: 3, hide: 1,
		heavy:  true,
		temper: herder, group: [2]int{5, 8}, most: 16, young: 0.3, wary: 9,
		lands: []world.Biome{world.Prairie, world.Highland},
		call:  sound.Horn, cry: sound.Alarm, callEvery: 70, carries: 3 * sound.Hearing,
	},
	render.Nipper: {
		walk: 1.8, run: 5.5, gait: 9,
		health: 3, meat: 1,
		temper: thief, group: [2]int{3, 5}, most: 10, wary: 5, lure: 9,
		lands: []world.Biome{world.Prairie, world.Jungle},
		call:  sound.Chirp, cry: sound.Chirp, callEvery: 18,
	},
	render.Quickclaw: {
		walk: 2.0, run: 4.8, gait: 4.2,
		health: 10, bite: 3, meat: 1, hide: 1,
		temper: hunter, group: [2]int{2, 3}, most: 3, sees: 20, hears: 14,
		visits: true, from: firstHunt, comes: [2]float64{hunterNear, hunterFar}, bears: hunterLit,
		stalks: preySight, snaps: biteWait, feeds: feedTime,
		call: sound.Growl, cry: sound.Screech, callEvery: 9,
	},

	// The armed plant-eaters. They are few, strong, and hard to kill.
	render.Shieldbrow: {
		walk: 1.2, run: 7.0, gait: 1.5,
		health: 60, meat: 8, hide: 3,
		heavy:  true,
		temper: guard, group: [2]int{3, 4}, grown: 2, most: 4,
		warns: 10, charges: 6, blow: 6, steps: 2,
		lands: []world.Biome{world.Prairie},
		call:  sound.Grunt, cry: sound.Bellow, callEvery: 45, carries: 2 * sound.Hearing,
	},
	render.Thornback: {
		walk: 0.9, run: 1.6, gait: 1.7,
		health: 45, meat: 6, hide: 2,
		heavy:  true,
		temper: guard, group: [2]int{1, 1}, most: 2,
		warns: 6, swings: 2.2, blow: 5, steps: 2,
		lands: []world.Biome{world.Highland, world.Jungle},
		call:  sound.Grunt, cry: sound.Rattle, callEvery: 50,
	},
	render.Boulderback: {
		walk: 0.7, run: 1.2, gait: 2.4,
		health: 50, armor: 3, meat: 6, hide: 2,
		heavy:  true,
		temper: guard, group: [2]int{1, 1}, most: 2,
		warns: 5, swings: 2.0, blow: 7, steps: 2,
		lands: []world.Biome{world.Badlands}, bare: true,
		call: sound.Grunt, cry: sound.Rattle, callEvery: 55,
	},

	// The giant among them: it goes in a pair, slowly, and takes no notice of anything.
	render.Cloudgrazer: {
		walk: 1.0, run: 1.0, gait: 1.15,
		heavy:  true,
		temper: herder, group: [2]int{2, 2}, most: 2,
		calm: true, wades: true, steps: 2,
		stamp: 8, foot: [2]float64{1.0, 2.3}, quake: 40,
		lands: []world.Biome{world.Prairie, world.Shore}, bare: true,
		call: sound.Drone, cry: sound.Drone, callEvery: 80, carries: 90,
	},

	// The top hunter. It comes rarely, by day or night, and one at a time; it is felt
	// before it is seen; light does not hold it, and only a big fire does; and it is
	// too strong to kill in the open.
	render.Dread: {
		walk: 2.2, run: 6.0, gait: 1.25,
		health: 200, bite: 8, meat: 12, hide: 4,
		heavy:  true,
		temper: hunter, group: [2]int{1, 1}, most: 1, sees: 40, hears: 28,
		visits: true, from: dreadFrom, comes: [2]float64{70, 86}, fears: 7, stays: 300, every: [2]float64{1, 2},
		stalks: 32, snaps: 1.5, feeds: 20, cows: true,
		steps: 2, quake: 60,
		call: sound.Rumble, cry: sound.Roar, callEvery: 14, carries: 90,
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
	climb   int        // how many blocks high the step it last hopped at is
	trod    int        // which pair of its feet came down in the last step, 1 or 2, or 0 for none
	peak    float64    // the highest it has been since it was last on the ground
	walking bool       // or standing
	timer   float64    // until it changes its mind
	rng     *rand.Rand // its own, so that what it does can be told from a seed

	push   geometry.Vector3 // the way a hit sends it, and how fast, along the ground
	flee   float64          // how long it still runs
	hurt   float64          // how long it still shows a hit
	regrow float64          // how long a mossback's moss still takes to grow back, or 0 if it has it
	bumped float64          // until it can hurt the player again by running into it

	dead   bool    // it has no health left
	fall   float64 // from 0, as it dies, to 1, gone
	carved bool    // a knife killed it, so it leaves more

	fed   int     // how many meals it has had from the player's hand
	tame  bool    // it trusts the player, and follows
	chew  float64 // how long it still chews its last meal
	hurry bool    // it runs this frame: after food, after prey, or to catch the player up

	// A hunter: how long it still keeps after the player, until it can bite again, how
	// long it still stands over its kill, how long it has stood in the light, and
	// whether it has turned to go, and for how long.
	keen, biteIn, feeding, lit float64
	leaving                    bool
	gone                       float64
	about                      float64 // how long a hunter that visits has been about

	// A guard: how long it still charges, until it can charge or strike again, and
	// until it warns again; whether it was hurt, and where what it turns on is; whether
	// its head is down; and how long its tail is still seen to swing, and to which side.
	ire, rest, warned float64
	angry, lower      bool
	foe               geometry.Vector3
	lash, lashSide    float64
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

	// sneaking is whether the player is crouched, which lets it come nearer a herd, and
	// bait whether the player holds food out, which brings thieves.
	sneaking, bait bool

	// What a hunter can know of the player: whether it moves, whether it is hidden
	// among tall plants, and whether it carries raw meat. hunted is whether hunters are
	// about at all, which they are not in creative, day the day of the world, counted
	// from its first midnight, which says which of them are, and bright how bright the
	// light at a place is. fires are the fires near the player that burn three or more
	// together, which is a big fire: a hunter that fears one keeps off them.
	moving, hidden, meaty, hunted bool
	winter                        bool // it is the Long Winter: herds are thin, and hunters do not mind the day
	day                           float64
	bright                        func(at geometry.Vector3) float64
	fires                         []geometry.Vector3

	// wait is how many seconds are left before the next of a kind that comes only now
	// and then may come.
	wait [render.Kinds]float64

	// shove is the way the blows of the guards and the feet of the giants threw the
	// player in the last step, and how far. treads are the steps of the giants in it:
	// where a foot came down, and of which kind.
	shove  geometry.Vector3
	treads []said

	// broken are the pit covers that heavy beasts broke through in the last step.
	broken []world.Cell

	robbed float64 // until a thief may take from the player again
	feet   int     // the height of the player's feet, which a hunter comes out near

	alarms map[int]alarm

	// What the last step left for the game to act on: the sounds the beasts made, the
	// beasts that died and are gone, and how many times a thief took from the player.
	sounds []said
	fallen []beast
	thefts int
	bitten float64 // points of harm the hunters' bites did the player
}

func emptyFauna(w *world.World) *fauna {
	return &fauna{rng: rand.New(rand.NewPCG(uint64(w.Seed), 0x6d6f7373)), alarms: map[int]alarm{}}
}

// newFauna returns the beasts of a world round a place, as many of each kind as there
// is room for, up to what its breed says. The same world and place always give the same.
func newFauna(w *world.World, round geometry.Vector3) *fauna {
	f := emptyFauna(w)
	for k := range render.Kinds {
		if breeds[k].visits {
			continue // those come in their own time
		}
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
	// A world that is opened again is given a while before the next of a kind that
	// comes only now and then: it does not come the moment the player is back.
	for k := range f.wait {
		if breeds[k].every[1] > 0 {
			f.wait[k] = dayLength * (0.2 + 0.6*f.rng.Float64())
		}
	}
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
		if !ok || breeds[kind].visits {
			continue // a kind of a later version of the game, or one that is not kept
		}
		r := own()
		f.all = append(f.all, beast{
			kind: kind, Pos: b.Pos, Yaw: b.Yaw, young: b.Young, herd: b.Herd,
			health: min(max(b.Health, 1), max(breeds[kind].health, 1)), regrow: b.Regrow,
			tame: b.Tame, fed: b.Fed,
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
		if !c.dead && !breeds[c.kind].visits {
			out = append(out, world.Beast{
				Kind: c.kind.String(), Pos: c.Pos, Yaw: c.Yaw, Young: c.young, Herd: c.herd,
				Health: c.health, Regrow: c.regrow, Tame: c.tame, Fed: c.fed,
			})
		}
	}
	return out
}

// count returns how many wild beasts of a kind there are. A tame one is the player's,
// and takes no wild one's place.
func (f *fauna) count(k render.Kind) (n int) {
	for i := range f.all {
		if f.all[i].kind == k && !f.all[i].tame {
			n++
		}
	}
	return n
}

// standing returns where a beast of a kind could stand in the column at x, z: on mossy
// earth, in a land of its breed, with room over it for its body, and not on top of
// another beast.
func (f *fauna) standing(w *world.World, k render.Kind, young bool, x, z int) (geometry.Vector3, bool) {
	if breeds[k].visits {
		return f.lair(w, k, x, z, f.feet)
	}
	y := w.Ground(x, z) // under any plant
	switch on := w.At(x, y, z); {
	case on == world.Turf:
	case breeds[k].bare && (on == world.Redrock || on == world.Sand || on == world.Sandstone):
	default:
		return geometry.Vector3{}, false // not ground it stands on, or a place that is not loaded
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
	f.feet = int(math.Floor(round.Y))
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
	// A group is whole or it is not there: there must be room under the most of its
	// kind for the fewest that go together.
	room := f.most(k) - f.count(k)
	if room < b.group[0] {
		return false
	}
	size = min(size, room)
	herd := 0
	if b.group[1] > 1 {
		f.herds++
		herd = f.herds
	}
	placed, before := 0, len(f.all)
	for try := 0; placed < size && try < 12*size; try++ {
		mx, mz := x, z
		if placed > 0 { // the first stands on the place itself, and the others round it
			mx += f.rng.IntN(2*groupReach+1) - groupReach
			mz += f.rng.IntN(2*groupReach+1) - groupReach
		}
		young := placed > 0 && f.rng.Float64() < b.young
		if b.grown > 0 {
			young = placed >= b.grown // a family: its grown ones, and then its young
		}
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
	if herd != 0 && placed < 2 {
		f.all = f.all[:before] // one alone is no herd: it found no room for another
		return false
	}
	return placed > 0
}

// middles returns the middle of every herd: where its living members are, on the whole,
// or only the young of them.
func (f *fauna) middles(young bool) map[int]geometry.Vector3 {
	sum, n := map[int]geometry.Vector3{}, map[int]float64{}
	for i := range f.all {
		if c := &f.all[i]; c.herd != 0 && !c.dead && (c.young || !young) {
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
		if breeds[c.kind].temper == guard && !c.young {
			c.angry, c.foe = true, from // it does not run: it turns on what alarmed its young
			continue
		}
		if breeds[c.kind].calm {
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
	f.sounds, f.fallen, f.thefts, f.bitten, f.shove = f.sounds[:0], f.fallen[:0], 0, 0, geometry.Vector3{}
	f.treads, f.broken = f.treads[:0], f.broken[:0]
	f.robbed = max(f.robbed-dt, 0)
	for herd, a := range f.alarms {
		if a.left -= dt; a.left <= 0 {
			delete(f.alarms, herd)
		} else {
			f.alarms[herd] = a
		}
	}

	middles, wards := f.middles(false), f.middles(true)
	for i := range f.all {
		c := &f.all[i]
		if c.Pos.Distance(player) >= creatureAwake {
			continue
		}
		b := &breeds[c.kind]
		to := geometry.Vector3{X: player.X - c.Pos.X, Z: player.Z - c.Pos.Z}
		far := to.Length()

		// A thief comes for food held out, where another time it would run: it darts
		// in, takes it, and is off, and its flock goes with it.
		c.hurry = false
		lured := b.temper == thief && f.bait && f.robbed == 0 && !c.dead && c.flee <= 0 && far < b.lure
		switch {
		case lured && far < thiefReach:
			f.thefts++
			f.robbed = robWait
			f.raise(c.herd, player)
			f.sounds = append(f.sounds, said{b.cry, c.kind, c.voice()})
		case lured:
			c.hurry, c.walking, c.timer, c.Yaw = true, true, max(c.timer, rethink), yawOf(to)
		}
		if wary := b.wary; wary > 0 && !c.dead && !lured {
			if f.sneaking {
				wary *= sneakPart
			}
			if c.Pos.Distance(player) < wary && f.raise(c.herd, player) {
				f.sounds = append(f.sounds, said{b.cry, c.kind, c.voice()}) // the one that saw it calls out
			}
		}
		// A tame beast keeps by the player: it comes after one that goes, runs to catch
		// up when it has fallen far behind, and stands by one that stands.
		if c.tame && !c.dead && c.flee <= 0 {
			if far > followFar {
				c.walking, c.Yaw, c.hurry = true, yawOf(to), far > followRun
			} else {
				c.walking = false
			}
			c.timer = max(c.timer, rethink)
		}
		if b.temper == hunter && !c.dead {
			f.hunt(w, i, player, dt)
		}
		if b.temper == guard && !c.dead {
			f.guard(i, player, dt)
		}
		var middle *geometry.Vector3
		if m, ok := middles[c.herd]; ok && b.temper != loner {
			middle = &m
		}
		if m, ok := wards[c.herd]; ok && b.temper == guard && !c.young {
			middle = &m // a grown guard keeps by its young, and not by the middle of its family
		}
		c.step(w, dt, middle)
		if c.trod != 0 {
			f.tread(i, player)
		}
		if b.heavy && c.OnGround && !c.dead {
			f.collapse(w, c)
		}
		if b.bump > 0 && c.flee > 0 && c.bumped == 0 && !c.dead {
			f.trample(i)
		}
	}
	f.all = slices.DeleteFunc(f.all, func(c beast) bool {
		if c.dead && c.fall >= 1 {
			f.fallen = append(f.fallen, c)
			f.went(c.kind)
			return true
		}
		// A hunter that has turned to go is let go once it is out of the way: far
		// enough not to be seen to vanish, for one that is about by day.
		far, long := leaveAt, leaveAfter
		if b := &breeds[c.kind]; b.stays > 0 {
			far, long = b.comes[0], 3*leaveAfter
		}
		if c.leaving && (c.gone > long || c.Pos.Distance(player) > far) {
			f.went(c.kind)
			return true
		}
		return false
	})
	for k := range f.wait {
		f.wait[k] = max(f.wait[k]-dt, 0)
	}

	if f.timer -= dt; f.timer > 0 {
		return
	}
	f.timer = 1
	f.all = slices.DeleteFunc(f.all, func(c beast) bool {
		if !c.tame && c.Pos.Distance(player) > creatureGone {
			f.went(c.kind)
			return true
		}
		return false
	})
	for k := range render.Kinds {
		switch b := &breeds[k]; {
		case f.count(k) >= f.most(k):
		case b.visits:
			if f.about(k) && f.wait[k] == 0 {
				f.add(w, k, player, b.comes[0], b.comes[1]) // near enough to find the player
			}
		case !b.visits:
			f.add(w, k, player, creatureRange/2, creatureAwake-8) // out of the way, where it will be met
		}
	}
}

// body returns the middles of the feet of the boxes that cover a beast's body, for what
// is aimed, swung or thrown at it: the one box that blocks stop it in, for most, and a
// row of them along the way it faces, for a beast much longer than that box is wide.
func (c *beast) body() (at [6]geometry.Vector3, n int) {
	width, _ := render.Size(c.kind, c.young)
	back, front := render.Span(c.kind, c.young)
	n = min(max(int(math.Ceil((front-back)/width)), 1), len(at))
	ahead := mesh.Camera{Yaw: c.Yaw}.Forward()
	for i := range n {
		along := (back + front) / 2
		if n > 1 {
			along = back + width/2 + float64(i)*(front-back-width)/float64(n-1)
		}
		at[i] = c.Pos.Add(ahead.Scale(along))
	}
	return at, n
}

// How a heavy beast fares with a pit.
const (
	fallSafe  = 2.0     // blocks it falls unhurt
	fallPart  = 1.0 / 6 // the part of its kind's health each block more costs it
	coverMost = 64      // the most covers that go at once: a cover wider than that is two
	coverTip  = 0.4     // how far past half its width it is tipped forward as the cover goes, in blocks
)

// collapse breaks the pit cover a heavy beast has its middle on, and all of the cover
// that joins it, and tips the beast forward into the pit: the whole of a cover gives
// way at once, as branches laid across a hole do.
func (f *fauna) collapse(w *world.World, c *beast) {
	under := world.Cell{X: int(math.Floor(c.Pos.X)), Y: int(math.Floor(c.Pos.Y - 0.5)), Z: int(math.Floor(c.Pos.Z))}
	if w.At(under.X, under.Y, under.Z) != world.PitCover {
		return
	}
	from := len(f.broken)
	for joined := []world.Cell{under}; len(joined) > 0 && len(f.broken)-from < coverMost; {
		at := joined[len(joined)-1]
		joined = joined[:len(joined)-1]
		if w.At(at.X, at.Y, at.Z) != world.PitCover {
			continue
		}
		w.Set(at.X, at.Y, at.Z, world.Air)
		f.broken = append(f.broken, at)
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			joined = append(joined, world.Cell{X: at.X + d[0], Y: at.Y, Z: at.Z + d[1]})
		}
	}
	// It was on the rim with its middle just over the cover: it goes on by half its
	// width and a little more, which takes the rest of it off the rim.
	width, _ := render.Size(c.kind, c.young)
	c.push = c.push.Add(mesh.Camera{Yaw: c.Yaw}.Forward().Scale(knockDrag * (width/2 + coverTip)))
}

// trample has a running beast that runs others down hurt the first lighter beast of
// another kind it runs into, as it hurts a player: a stampede kills what is in its way.
func (f *fauna) trample(i int) {
	c := &f.all[i]
	b := &breeds[c.kind]
	width, height := render.Size(c.kind, c.young)
	lo, hi := boxOf(c.Pos, width, height)
	for j := range f.all {
		o := &f.all[j]
		if o.kind == c.kind || o.dead || breeds[o.kind].heavy {
			continue // its own kind runs with it, and a heavy beast is not run down
		}
		ow, oh := render.Size(o.kind, o.young)
		olo, ohi := boxOf(o.Pos, ow, oh)
		if lo[0] >= ohi[0] || hi[0] <= olo[0] || lo[1] >= ohi[1] || hi[1] <= olo[1] || lo[2] >= ohi[2] || hi[2] <= olo[2] {
			continue
		}
		c.bumped = bumpWait
		o.hit(c.Pos, b.bump)
		f.sounds = append(f.sounds, said{breeds[o.kind].cry, o.kind, o.voice()})
		return
	}
}

// stampReach is how far from where a giant's foot comes down a player is under it.
const stampReach = 1.0

// tread is a pair of a giant's feet coming down: it is left for the game to sound and
// to shake the view with, and a player under either foot is hurt and thrown aside.
func (f *fauna) tread(i int, player geometry.Vector3) {
	c := &f.all[i]
	b := &breeds[c.kind]
	f.treads = append(f.treads, said{sound.Thud, c.kind, c.Pos})
	if b.stamp == 0 || player.Y < c.Pos.Y-1 || player.Y > c.Pos.Y+2 {
		return
	}
	turned := mesh.Camera{Yaw: c.Yaw}
	side := float64(3 - 2*c.trod) // the pair: a front foot on one side and the back foot on the other
	for _, end := range [2]float64{1, -1} {
		foot := c.Pos.Add(turned.Right().Scale(side * end * b.foot[0])).Add(turned.Forward().Scale(end * b.foot[1]))
		if math.Hypot(player.X-foot.X, player.Z-foot.Z) <= stampReach {
			f.bitten += b.stamp
			f.shove = f.shove.Add(awayFrom(player, foot, c.Yaw+math.Pi/2).Scale(blowShove * b.stamp))
		}
	}
}

// about reports whether the hunters of a kind are about: hunters are, and the world is
// old enough for that kind.
func (f *fauna) about(k render.Kind) bool {
	return f.hunted && f.day >= breeds[k].from
}

// went notes that a beast of a kind is gone, for a kind that comes only now and then:
// the next of it comes a day or two on.
func (f *fauna) went(k render.Kind) {
	if e := breeds[k].every; e[1] > 0 {
		f.wait[k] = dayLength * (e[0] + f.rng.Float64()*(e[1]-e[0]))
	}
}

// voice returns where a beast's sounds come from: its head, more or less.
func (c *beast) voice() geometry.Vector3 {
	_, height := render.Size(c.kind, c.young)
	return c.Pos.Add(geometry.Vector3{Y: 0.85 * height})
}

// Pose returns where the beast is and how it stands, for drawing.
func (c *beast) Pose() render.Beast {
	pose := render.Beast{
		Kind: c.kind, Pos: c.Pos, Yaw: c.Yaw, Stride: c.Stride,
		Shorn: c.regrow > 0, Hurt: c.hurt > 0, Young: c.young, Fall: c.fall,
	}
	if c.lower && !c.dead {
		pose.Lower = 1
	}
	if c.lash > 0 && !c.dead {
		pose.Lash = c.lashSide * math.Sin(math.Pi*(1-c.lash/swingShow)) // out, and back
	}
	return pose
}

// hit is the beast being hit from a place, by points of harm, less what its armor
// takes. It is knocked away, and runs, unless it is a guard. A beast with health loses
// the points, and with none left it dies. It reports whether the hit killed it.
func (c *beast) hit(from geometry.Vector3, points float64) (killed bool) {
	b := &breeds[c.kind]
	if c.dead || b.calm {
		return false
	}
	points = max(points-b.armor, 0)
	if b.temper == guard && !c.young {
		// A grown guard is too heavy to knock back, and it does not run: it turns on
		// what hurt it. A hit its armor took all of does not show.
		c.angry, c.foe = true, from
		if points > 0 {
			c.hurt = hurtShow
		}
	} else {
		away := awayFrom(c.Pos, from, c.Yaw)
		c.push = away.Scale(knockSpeed)
		if c.OnGround {
			c.velY = knockHop
		}
		c.hurt, c.flee = hurtShow, fleeTime
		c.walking, c.Yaw = true, yawOf(away)
	}
	if b.health > 0 {
		if c.health -= points; c.health <= 0 {
			c.dead, c.walking, c.flee = true, false, 0
			return true
		}
	}
	return false
}

// feed gives the beast a meal from the player's hand, if it is one that can be tamed, is
// not tame yet, takes that food and is not still chewing. Enough meals make it tame.
// It reports whether it ate, and whether that meal tamed it.
func (c *beast) feed(food world.Item) (ate, tamed bool) {
	b := &breeds[c.kind]
	if c.tame || c.dead || c.chew > 0 || !slices.Contains(b.eats, food) {
		return false, false
	}
	c.fed, c.chew = c.fed+1, chewTime
	c.flee = 0 // what feeds it is nothing to run from
	c.tame = c.fed >= b.meals
	return true, c.tame
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
// walk on, hop up onto it, or turn back. solid says which blocks stop the beast.
func (c *beast) wayAhead(w *world.World, ahead geometry.Vector3, afloat bool, solid func(*world.World, int, int, int) bool) (hop, blocked bool) {
	width, height := render.Size(c.kind, c.young)
	front := c.Pos.Add(ahead.Scale(width/2 + lookAhead))
	x, y, z := int(math.Floor(front.X)), int(math.Floor(c.Pos.Y+0.1)), int(math.Floor(front.Z))

	if solid(w, x, y, z) {
		// A block at the height of its feet. It is a step the beast can hop, if it is no
		// higher than its kind climbs, the beast has its feet on something, and there is
		// room on top for the whole of it.
		high := 1
		for high < max(breeds[c.kind].steps, 1) && solid(w, x, y+high, z) {
			high++
		}
		for up := high; up < high+int(math.Ceil(height)); up++ {
			if solid(w, x, y+up, z) {
				return false, true
			}
		}
		c.climb = high
		return c.OnGround || afloat, true
	}
	if in, under := w.At(x, y, z), w.At(x, y-1, z); in == world.Water || under == world.Water || in == world.Lava || under == world.Lava {
		return false, true // it keeps out of water, and out of lava
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
	c.chew = max(c.chew-dt, 0)
	if c.dead {
		c.fall = min(c.fall+dt/fallTime, 1)
		c.Stride = 0
	}
	speed := b.walk
	if c.hurry {
		speed = b.run
	}
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

	stops := solid
	if b.wades {
		stops = firm
	}
	c.trod = 0
	var move geometry.Vector3
	if c.walking && !c.dead {
		ahead := mesh.Camera{Yaw: c.Yaw}.Forward()
		switch hop, blocked := c.wayAhead(w, ahead, afloat, stops); {
		case hop:
			c.velY = creatureHop
			if c.climb > 1 {
				c.velY = math.Sqrt(2 * gravity * (float64(c.climb) + 0.2)) // enough to clear that many blocks
			}
		case blocked && (c.OnGround || afloat) && c.flee > 0:
			c.Yaw += math.Pi / 2 * float64(1-2*c.rng.IntN(2)) // running, it turns aside
		case blocked && (c.OnGround || afloat):
			c.turnBack() // in the air it is in the middle of a hop, and goes on
		}
		if c.walking {
			move = ahead.Scale(speed * dt)
			before := math.Floor(c.Stride / math.Pi)
			c.Stride += b.gait * speed * dt
			if now := math.Floor(c.Stride / math.Pi); now != before && b.quake > 0 {
				c.trod = 1 + int(now)&1 // a pair of its feet comes down each half of a stride, in turn
			}
		}
	}
	if !c.walking {
		c.Stride = 0
	}
	move = move.Add(c.push.Scale(dt))
	c.push = c.push.Scale(max(1-knockDrag*dt, 0))
	move.Y = c.velY * dt

	if c.OnGround || afloat {
		c.peak = c.Pos.Y
	}
	stopped := travelIn(w, &c.Pos, width, height, move, stops)
	c.OnGround = stopped[1] && move.Y < 0
	if stopped[1] {
		c.velY = 0
	}
	if c.peak = max(c.peak, c.Pos.Y); c.OnGround {
		// A heavy beast is hurt by a fall of more than fallSafe blocks: a part of what
		// health its kind has for each block more, so the bigger it is, the harder it
		// comes down.
		if drop := c.peak - c.Pos.Y; b.heavy && b.health > 0 && drop > fallSafe && !c.dead {
			c.hurt = hurtShow
			if c.health -= b.health * fallPart * (drop - fallSafe); c.health <= 0 {
				c.dead, c.walking, c.flee = true, false, 0
			}
		}
		c.peak = c.Pos.Y
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
	held := g.items[g.selected]
	f.sneaking = g.player.Crouched
	f.bait = g.survival && held.Item.Food() > 0 // in creative there is nothing to lose
	f.moving, f.hidden, f.hunted, f.day = g.paced, g.hidden(), g.survival, firstMorning+g.days
	g.findFires(dt)
	f.meaty = slices.ContainsFunc(g.items[:], func(s world.Stack) bool { return s.Item == world.Meat })
	f.winter = g.winter()
	f.bright = func(at geometry.Vector3) float64 {
		l := g.lightAt(at)
		if f.winter {
			l &= world.MaxLight // in the winter the ashen day is no light to a hunter: only a fire's is
		}
		return g.sky.Brightness(l)
	}
	g.sky.Set(timeOfDay(g.days)) // the light they keep out of is the light of this hour
	f.step(g.world, g.player.Pos, dt)
	if f.bitten > 0 && g.survival {
		travel(g.world, &g.player.Pos, playerWidth, playerHeight, f.shove) // a blow throws
		if g.hurt(f.bitten); g.screen != playing {
			return // the player fainted
		}
	}
	if f.thefts > 0 {
		g.items.takeOne(g.selected)
		g.tell("a nipper took your " + held.Item.Name())
	}
	for _, s := range f.sounds {
		g.sayFrom(s.kind, s.name, s.at, 0.9)
	}
	for _, at := range f.broken {
		g.ears.Play(sound.Break(materialOf(world.PitCover)), cellMid(at.X, at.Y, at.Z), 1)
		g.shatter(at.X, at.Y, at.Z, world.PitCover)
	}
	for _, s := range f.treads {
		// A giant's step is heard from far, and felt from nearer: the view shakes the
		// more the nearer it is.
		g.sayFrom(s.kind, s.name, s.at, 1)
		if far := breeds[s.kind].quake; s.at.Distance(g.player.Pos) < far {
			g.quake = max(g.quake, math.Pow(1-s.at.Distance(g.player.Pos)/far, 2))
		}
	}
	for _, c := range f.fallen {
		// A kill carved with a knife leaves one more piece of meat, and always a hide.
		meat, hides := breeds[c.kind].meat, breeds[c.kind].hide
		if c.carved {
			meat, hides = meat+1, max(hides, 1)
		}
		for i := range meat + hides {
			item := world.Meat
			if i >= meat {
				item = world.Hide
			}
			at := c.Pos.Add(geometry.Vector3{
				X: (g.scatter.Float64() - 0.5) * 2 * meatSpread, Y: 0.3, Z: (g.scatter.Float64() - 0.5) * 2 * meatSpread,
			})
			g.drops = append(g.drops, drop{Drop: world.Drop{Stack: world.Stack{Item: item, Count: 1}, Pos: at}})
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

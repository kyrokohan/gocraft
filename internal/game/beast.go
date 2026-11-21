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
	floatSpeed    = 1.5   // how fast a beast in water comes up to the top of it
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
	// lore is what the card of a painting of it says: what it eats and fears, what it
	// does to you, and how it is won or killed, in two to four short lines.
	lore []string

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

	// heralds is how many seconds before a pack that visits comes its cry is heard from
	// where it will come: none, for one that comes unheard. byDay is true for a hunter
	// that hunts in the light and lies still in the dark. springs is how far from what
	// it hunts it springs at it: none, for one that runs it down.
	heralds, springs float64
	byDay            bool

	temper temper
	group  [2]int  // the fewest and the most that are found together
	most   int     // of its kind round the player
	young  float64 // the part of a group that is not grown

	// wary is how near a standing player may come before its herd takes alarm, and
	// lure how far a thief comes from for food the player holds.
	wary, lure float64

	// eats is what it takes from the player's hand, and meals how many of them make it
	// trust the player: a beast with none cannot be tamed. daily is how many of them it
	// takes in a day, for one that takes no more. Of a guard only the young are won so.
	eats         []world.Item
	meals, daily int

	// lands are the lands it is found in. A beast with none is found wherever there is
	// mossy earth.
	lands []world.Biome

	meat, hide int // pieces of raw meat, and hides, it leaves
	tusks      int // and tusks, for a grown one that has them

	// rears is true for a guard that stands up on its hind legs: when it stops, as to
	// feed from a tree, and when it strikes.
	rears bool

	// moves is how it gets about: it walks, if its row says nothing. hover is how high
	// over the ground or the water one that flies keeps. curls is how many seconds one
	// that curls up when it is hit stays so: curled, nothing hurts it.
	moves gait
	hover float64
	curls float64

	// warms is true for a beast that the sun warms: it is as quick as its row says only
	// at noon, and slow in the morning, the evening and the winter. tether is how far
	// from where it lives one that keeps to a place goes after its prey.
	warms  bool
	tether float64

	// One that swims lies draft blocks under the top of the water, by its feet, and
	// keeps to water deep blocks deep or more. sinks is how much lower it goes when it
	// runs from something, or, a hunter, while it is after nothing. leaps is how many
	// seconds it swims between two leaps clear of the water, about, for one that leaps.
	// plays is from how far it comes to a player who is in the water, to swim by it.
	// jaws is how far ahead of its middle its jaws are: it bites from that much farther.
	draft, sinks, leaps, plays, jaws float64
	deep                             int

	// One that soars comes down to the water under it about once in dips seconds, and
	// one that roosts comes down for the night on high ground, where there is some near.
	// takes is how high over the ground a beast that walks flies when it runs: it has
	// wings, and takes to them.
	dips, takes float64
	roosts      bool

	// nests is how many herds of it in one keep a nest, and egg what is taken out of one.
	nests float64
	egg   world.Item

	// One that can be ridden: how fast it runs with a rider held on, how high its back
	// is, and how far ahead of its middle the rider sits. What Space does: how many blocks
	// high it jumps, or the points its charge takes. One that is hairy keeps its rider
	// warm, and one that crushes runs down the hunters in its way as it is ridden. One
	// that glides is not ridden: it carries the player down from a height.
	ride, back, seat, clears, rams float64
	hairy, crushes, glides         bool

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
		lore: []string{
			"grazes alone on the moss",
			"cannot be hurt: it runs",
			"three meals of berries or fruit tames it",
			"keeps out of tar pits",
		},
		walk: creatureSpeed, run: 3.4, gait: 6.15,
		group: [2]int{1, 1}, most: creatureCount,
		eats: []world.Item{world.Berries, world.Fruit}, meals: 3,
		call: sound.Call, cry: sound.Cry, callEvery: callEvery,
	},
	render.Bugler: {
		lore: []string{
			"eats berries, fruit and fiber",
			"the herd runs when you come within 9 blocks",
			"a runner hits you for 3 and throws you",
			"has 16 health; or hatch an egg and ride one",
		},
		walk: 1.6, run: 6.5, gait: 2.6,
		health: 16, bump: 3, meat: 3, hide: 1,
		heavy:  true,
		temper: herder, group: [2]int{5, 8}, most: 16, young: 0.3, wary: 9,
		nests: 1.0 / 3, egg: world.BuglerEgg,
		eats: []world.Item{world.Berries, world.Fruit, world.Fiber},
		ride: 9, back: 2.2, seat: -0.3, clears: 2,
		lands: []world.Biome{world.Prairie, world.Highland},
		call:  sound.Horn, cry: sound.Alarm, callEvery: 70, carries: 3 * sound.Hearing,
	},
	render.Nipper: {
		lore: []string{
			"the flock scatters within 5 blocks",
			"holding food, one darts in from 9 blocks",
			"it takes a piece and runs",
			"has 3 health and leaves 1 meat",
		},
		walk: 1.8, run: 5.5, gait: 9,
		health: 3, meat: 1,
		temper: thief, group: [2]int{3, 5}, most: 10, wary: 5, lure: 9,
		lands: []world.Biome{world.Prairie, world.Jungle},
		call:  sound.Chirp, cry: sound.Chirp, callEvery: 18,
	},
	render.Quickclaw: {
		lore: []string{
			"hunts in the dark, in packs of 2 or 3",
			"runs faster than you walk; bites for 3",
			"will not step into torch or hearth light",
			"has 10 health and leaves a hide",
		},
		walk: 2.0, run: 4.8, gait: 4.2,
		health: 10, bite: 3, meat: 1, hide: 1,
		temper: hunter, group: [2]int{2, 3}, most: 3, sees: 20, hears: 14,
		visits: true, from: firstHunt, comes: [2]float64{hunterNear, hunterFar}, bears: hunterLit,
		stalks: preySight, snaps: biteWait, feeds: feedTime,
		call: sound.Growl, cry: sound.Screech, callEvery: 9,
	},

	// The armed plant-eaters. They are few, strong, and hard to kill.
	render.Shieldbrow: {
		lore: []string{
			"warns within 10 of its young, charges at 6",
			"a charge hits for 6 and throws you",
			"slow to turn: something solid saves you",
			"has 60 health; hatch an egg to raise one",
		},
		walk: 1.2, run: 7.0, gait: 1.5,
		health: 60, meat: 8, hide: 3,
		heavy:  true,
		temper: guard, group: [2]int{3, 4}, grown: 2, most: 4,
		warns: 10, charges: 6, blow: 6, steps: 2,
		nests: 0.5, egg: world.ShieldbrowEgg,
		eats: []world.Item{world.Berries, world.Fruit, world.Fiber},
		ride: 3, back: 2.4, seat: -0.6, rams: 7,
		lands: []world.Biome{world.Prairie},
		call:  sound.Grunt, cry: sound.Bellow, callEvery: 45, carries: 2 * sound.Hearing,
	},
	render.Thornback: {
		lore: []string{
			"rattles within 6 blocks and turns its tail",
			"its tail strikes for 5 and throws you",
			"it does not run: stay out of tail reach",
			"has 45 health; leaves 6 meat and 2 hides",
		},
		walk: 0.9, run: 1.6, gait: 1.7,
		health: 45, meat: 6, hide: 2,
		heavy:  true,
		temper: guard, group: [2]int{1, 1}, most: 2,
		warns: 6, swings: 2.2, blow: 5, steps: 2,
		lands: []world.Biome{world.Highland, world.Jungle},
		call:  sound.Grunt, cry: sound.Rattle, callEvery: 50,
	},
	render.Boulderback: {
		lore: []string{
			"armor takes 3 off every hit",
			"a hand or stab does nothing; a club does 1",
			"a thrown spear does 2, a hardened one 3",
			"its tail club hits for 7; it does not run",
		},
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
		lore: []string{
			"takes no notice of you, walks through trees",
			"keep out from under it: its foot hits for 8",
			"cannot be hurt and does not run",
		},
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
		lore: []string{
			"outruns you; bites for 8 of your 20",
			"only a big fire holds it: 3 or more fires",
			"hide crouched in ferns, or in a narrow cave",
			"a covered pit and spears kill it: 200 health",
		},
		walk: 2.2, run: 6.0, gait: 1.25,
		health: 200, bite: 8, meat: 12, hide: 4,
		heavy:  true,
		temper: hunter, group: [2]int{1, 1}, most: 1, sees: 40, hears: 28,
		visits: true, from: dreadFrom, comes: [2]float64{70, 86}, fears: 7, stays: 300, every: [2]float64{1, 2},
		stalks: 32, snaps: 1.5, feeds: 20, cows: true,
		steps: 2, quake: 60,
		call: sound.Rumble, cry: sound.Roar, callEvery: 14, carries: 90,
	},

	// The life of the cold steppe. A herd of snowtusks guards its young as a family of
	// shieldbrows does, and its grown ones charge together. A slowclaw goes alone and
	// takes no notice of anything, until it is hit: then it stands up and comes.
	render.Snowtusk: {
		lore: []string{
			"trumpets within 10 of a calf; charges at 6",
			"each one hits for 7 and throws you",
			"crouch, hold fiber, berries or fruit",
			"feed the calf 5 meals, 3 a day: it is yours",
		},
		walk: 1.1, run: 6.0, gait: 1.2,
		health: 80, meat: 10, hide: 4, tusks: 2,
		heavy:  true,
		temper: guard, group: [2]int{3, 5}, grown: 3, most: 5,
		warns: 10, charges: 6, blow: 7, steps: 2, wades: true,
		eats: []world.Item{world.Fiber, world.Berries, world.Fruit}, meals: 5, daily: 3,
		ride: 2.5, back: 4.4, seat: -0.6, hairy: true, crushes: true,
		lands: []world.Biome{world.Tundra}, bare: true,
		call: sound.Grunt, cry: sound.Trumpet, callEvery: 50, carries: 2 * sound.Hearing,
	},
	render.Slowclaw: {
		lore: []string{
			"takes no notice of you, however near",
			"if hit, it comes slowly and hits for 6",
			"step away and it does not follow",
			"has 40 health; leaves 6 meat and 3 hides",
		},
		walk: 0.6, run: 1.6, gait: 2.2,
		health: 40, meat: 6, hide: 3,
		heavy:  true,
		temper: guard, group: [2]int{1, 1}, most: 2,
		charges: slowclawGrudge, blow: 6, rears: true,
		lands: []world.Biome{world.Tundra}, bare: true,
		call: sound.Grunt, cry: sound.Bellow, callEvery: 60,
	},

	// The hunters of the steppe. Howlers come by night in a pack, to a player who is on
	// the steppe, and their howl is heard before they are seen: light holds them off as
	// it does quickclaws. A daggercat lives there, alone, and hunts by day: it stalks,
	// springs, and backs off to spring again, and it will not come at a fire.
	render.Howler: {
		lore: []string{
			"packs of 4 to 6 hunt the steppe at night",
			"the howl comes 8 seconds before the pack",
			"quicker than quickclaws; a bite takes 3",
			"a torch in your hand keeps them off",
		},
		walk: 2.2, run: 5.4, gait: 5.0,
		health: 12, bite: 3, meat: 1, hide: 1,
		temper: hunter, group: [2]int{4, 6}, most: 6, sees: 26, hears: 18,
		visits: true, from: firstHunt, comes: [2]float64{hunterFar, hunterFar + 12}, bears: hunterLit,
		heralds: 8,
		stalks:  preySight, snaps: biteWait, feeds: feedTime,
		lands: []world.Biome{world.Tundra},
		call:  sound.Growl, cry: sound.Howl, callEvery: 11, carries: 4 * sound.Hearing,
	},
	render.Daggercat: {
		lore: []string{
			"hunts alone by day; sees you from 26 blocks",
			"springs from 8 blocks and hits for 10",
			"two steps to a side and it misses",
			"it keeps 12 blocks from you by a fire",
		},
		walk: 1.5, run: 5.6, gait: 3.4,
		health: 30, bite: 10, meat: 3, hide: 2,
		temper: hunter, group: [2]int{1, 1}, most: 1, sees: 26, hears: 12,
		from: firstHunt, byDay: true, springs: 8,
		stalks: 20, snaps: 4, feeds: feedTime,
		lands: []world.Biome{world.Tundra}, bare: true,
		call: sound.Growl, cry: sound.Snarl, callEvery: 40,
	},

	// The beasts of the coal swamp: strange, loud and harmless. A rattlewing flies, low
	// over the pools, and darts off when it is hit. A logcrawler creeps on the peat, and
	// curls up when it is hit: then nothing gets through its plates.
	render.Rattlewing: {
		lore: []string{
			"hangs in the air and never comes down",
			"does you no harm",
			"a hand hit sends it off; a spear kills it",
		},
		walk: 3.0, run: 8.0, gait: 60,
		health: 2,
		moves:  gaitFly, hover: 1.6,
		group: [2]int{1, 2}, most: 4,
		lands: []world.Biome{world.Swamp},
		call:  sound.Whirr, cry: sound.Whirr, callEvery: 7,
	},
	render.Logcrawler: {
		lore: []string{
			"creeps on the peat, keeps out of the water",
			"does you no harm",
			"when hit it curls up: nothing hurts it",
			"has 8 health and leaves 2 meat",
		},
		walk: 0.5, run: 0.9, gait: 9,
		health: 8, meat: 2, curls: 6,
		group: [2]int{1, 1}, most: 3,
		lands: []world.Biome{world.Swamp}, bare: true,
		call: sound.Rattle, cry: sound.Rattle, callEvery: 90,
	},

	// The two with a sail on the back. A sunback hunts the swamp and the badlands by
	// day, and is only as quick as the sun has made it. A sailjaw lies in still water
	// and lunges at what comes near, and does not go far from its water.
	render.Sunback: {
		lore: []string{
			"hunts by day; at noon it outruns you",
			"slow at sunrise and sunset: walk away",
			"it lies still in the dark",
			"bites for 6; has 35 health, 4 meat, 2 hides",
		},
		walk: 1.3, run: 6.0, gait: 4.0,
		health: 35, bite: 6, meat: 4, hide: 2,
		temper: hunter, group: [2]int{1, 1}, most: 2, sees: 22, hears: 12,
		from: firstHunt, byDay: true, warms: true,
		stalks: 14, snaps: 1.2, feeds: feedTime,
		lands: []world.Biome{world.Swamp, world.Badlands}, bare: true,
		call: sound.Hiss, cry: sound.Hiss, callEvery: 30,
	},
	render.Sailjaw: {
		lore: []string{
			"lies in still water, only its sail shows",
			"from night 2, it lunges at 5 blocks for 9",
			"two steps aside dodge it; slow on land",
			"it goes back to its water after 10 blocks",
		},
		walk: 1.0, run: 3.2, gait: 2.2,
		health: 60, bite: 9, meat: 6, hide: 3,
		heavy:  true,
		temper: hunter, group: [2]int{1, 1}, most: 1, sees: 7, hears: 6,
		from: firstHunt, moves: gaitWade, springs: 5, tether: 10,
		stalks: 6, snaps: 3, feeds: feedTime,
		lands: []world.Biome{world.Swamp, world.Jungle},
		call:  sound.Hiss, cry: sound.Hiss, callEvery: 45,
	},

	// The beasts of the sea, which swim. Lakenecks lie far out with their necks up,
	// and sink and swim off from what comes near. Leapfins go in schools, leap, and
	// come to swim by a player. The deepmaw hunts both, and a player, where the water
	// is six deep or more: it lies low until it has seen its prey, and then comes up.
	render.Lakeneck: {
		lore: []string{
			"lies far out in deep water, neck above it",
			"within 12 blocks it sinks and swims off",
			"swims faster than you; comes back up later",
			"has 40 health; leaves 6 meat and 2 hides",
		},
		walk: 1.2, run: 6.5, gait: 2.0,
		health: 40, meat: 6, hide: 2,
		temper: herder, group: [2]int{2, 3}, most: 3, wary: 12,
		moves: gaitSwim, draft: 1.1, deep: 5, sinks: 3,
		call: sound.Moan, cry: sound.Splash, callEvery: 40, carries: 2 * sound.Hearing,
	},
	render.Leapfin: {
		lore: []string{
			"goes in schools of 4 to 7 in deep water",
			"does not fear you: swim within 22 blocks",
			"the school comes and swims by you",
			"has 6 health and 2 meat; deepmaw hunts it",
		},
		walk: 3.0, run: 7.5, gait: 5.0,
		health: 6, meat: 2,
		temper: herder, group: [2]int{4, 7}, most: 7,
		moves: gaitSwim, draft: 0.7, deep: 3, leaps: 5, plays: 22,
		call: sound.Splash, cry: sound.Splash, callEvery: 9,
	},
	render.Deepmaw: {
		lore: []string{
			"lies low in water 6 deep or more",
			"from night 2 it sees you from 26 blocks",
			"it comes fast: a bite takes 10 of your 20",
			"in the shallows you are safe from it",
		},
		walk: 2.0, run: 7.0, gait: 1.2,
		health: 160, bite: 10, meat: 12, hide: 4,
		heavy:  true,
		temper: hunter, group: [2]int{1, 1}, most: 1, sees: 26,
		from: firstHunt, moves: gaitSwim, draft: 1.9, deep: 6, sinks: 3, jaws: 4.5,
		stalks: 20, snaps: 2.5, feeds: feedTime,
		call: sound.Splash, cry: sound.Splash, callEvery: math.Inf(1),
	},

	// The two great fliers. Galewings soar in circles over the shore and the sea, dip
	// to the water, sleep on high ground, and take no notice of a player. Stiltwings
	// stalk the badlands on foot, and take to their wings when one comes near.
	render.Galewing: {
		lore: []string{
			"soars over shore and sea; takes no notice",
			"by day, near its nest, it dives for 2",
			"hatch an egg by fire; feed it raw meat",
			"a raised one carries you down from heights",
		},
		walk: 6.0, run: 9.0, gait: 1.0,
		health: 12, meat: 2,
		moves: gaitSoar, hover: 14, dips: 25, roosts: true,
		group: [2]int{1, 2}, most: 3,
		nests: 1, egg: world.GalewingEgg, glides: true,
		eats:  []world.Item{world.Meat},
		lands: []world.Biome{world.Sea, world.Shore},
		call:  sound.Skree, cry: sound.Skree, callEvery: 35, carries: 3 * sound.Hearing,
	},
	render.Stiltwing: {
		lore: []string{
			"stalks the badlands on foot, tall as a tree",
			"within 14 blocks it takes to its wings",
			"flies about 60 blocks, faster than you walk",
			"does no harm; has 25 health and 4 meat",
		},
		walk: 1.4, run: 9.0, gait: 2.0,
		health: 25, meat: 4, hide: 1,
		temper: herder, group: [2]int{1, 2}, most: 2, wary: 14, takes: 10,
		lands: []world.Biome{world.Badlands}, bare: true,
		call: sound.Clack, cry: sound.Clack, callEvery: 45,
	},
}

// gait is how a beast gets about.
type gait uint8

const (
	gaitWalk gait = iota
	gaitFly       // it keeps a height over what is under it, and goes where it looks
	gaitWade      // it lies at the top of still water, and leaves the water only after its prey
	gaitSwim      // it keeps to water deep enough for it, and goes up and down in it
	gaitSoar      // it circles in the air, high over what is under it
)

// slowclawGrudge is how near what it was hit by a slowclaw must be for the slowclaw to
// go for it: nothing is that near that did not hit it, so it takes notice of nothing
// else.
const slowclawGrudge = 0.01

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

	fed int // how many meals it has had from the player's hand

	// For one that takes only so many meals a day: the day it last ate, and how many
	// meals it has had on that day.
	mealDay, mealsToday int

	// One the player raises: how many seconds it has grown, and how many more its last
	// meal lets it grow. One that is the player's and lost all its health limps home:
	// whether it does, and the day of the world on which it is whole again.
	age, fedFor float64
	limping     bool
	mendAt      float64
	tame        bool    // it trusts the player, and follows
	chew        float64 // how long it still chews its last meal
	hurry       bool    // it runs this frame: after food, after prey, or to catch the player up

	// One the player rides goes as the keys say, and not as it would: at so many blocks
	// a second this step, back if less than 0, and with a jump.
	ridden   bool
	carrying bool // a galewing that has the player in a glide
	reined   float64
	jumping  bool

	// A hunter: how long it still keeps after the player, until it can bite again, how
	// long it still stands over its kill, how long it has stood in the light, and
	// whether it has turned to go, and for how long.
	keen, biteIn, feeding, lit float64
	leap                       float64 // how long the spring of one that springs still lasts
	curl                       float64 // how long one that curls up when it is hit stays curled
	pace                       float64 // how much of its speed one that the sun warms has: all of it, if this says nothing
	leapt                      bool    // one that swims has left the water in a leap, and has not come back into it
	stalking                   bool    // a hunter is after a beast

	// One with wings: whether it is in the air, and, for one that roosts, whether it is
	// the hour to be on its roost, whether it has found one, and where. A soarer turns
	// so much a second, and is so many seconds into a dip yet.
	aloft, abed, roosting bool
	roost                 geometry.Vector3
	turn, dip             float64

	// One whose pair keeps a nest: where it is. It dives at a player who is near it:
	// whether one is now, and where; how long until it may dive again; and whether its
	// dive has just met the player.
	nest          geometry.Vector3
	nested        bool
	diving, dived bool
	quarry        geometry.Vector3
	mob           float64

	// den is where one that keeps to a place lives, once it has been seen there, and
	// homing whether it is on its way back there.
	den            geometry.Vector3
	denned, homing bool
	leaving        bool
	gone           float64
	about          float64 // how long a hunter that visits has been about

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

// egg is an egg that lies in the world: in the nest of a herd, wild, or where the player
// put it down.
type egg struct {
	kind render.Kind
	Pos  geometry.Vector3 // the middle of its bottom
	herd int              // whose nest it lies in, for a wild one
	wild bool             // nobody has taken it
	ring bool             // the first egg of a nest: the nest is drawn round it
	warm float64          // the seconds of a fire's warmth it has had
}

// About raising a beast, and one that is the player's.
const (
	hatchTime = dayLength     // how much of a fire's warmth hatches an egg, in seconds
	growTime  = 2 * dayLength // how long a young one that is fed grows before it is grown
	limpPace  = 0.5           // how much of its walk a beast that limps home has
	limpNear  = 3.0           // how near home it stops
	mendDays  = 1.0           // how many days after it fell it is whole
)

// About nests, and those who rob them.
const (
	thiefTime = 8.0  // how long the grown ones of a herd run at one who took its egg
	nestGuard = 20.0 // how near its nest a player may come before a galewing dives at it
	galeDive  = 2.0  // the points a galewing's dive takes
	diveWait  = 6.0  // from one dive of a galewing to its next
	diveReach = 1.6  // how near the head of a player a dive must come to meet it
	eggSize   = 0.55 // how far across an egg is
)

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
	eggs  []egg             // the eggs that lie in the world
	home  geometry.Vector3  // where a beast that is the player's limps to: the bed, or where a player starts
	start *geometry.Vector3 // where a player starts, once it has been looked for
	thief map[int]float64   // for a herd an egg was taken from: how long its grown ones still go for the thief
	rng   *rand.Rand        // where the next one goes: from the world's seed
	born  uint64            // how many there have been, to give each a generator of its own
	herds int               // how many herds there have been, to number the next
	timer float64           // until the beasts are next looked over

	// sneaking is whether the player is crouched, which lets it come nearer a herd, and
	// bait whether the player holds food out, which brings thieves. held is what the
	// player has in hand.
	sneaking, bait bool
	held           world.Item

	// What a hunter can know of the player: whether it moves, whether it is hidden
	// among tall plants, and whether it carries raw meat. hunted is whether hunters are
	// about at all, which they are not in creative, day the day of the world, counted
	// from its first midnight, which says which of them are, and bright how bright the
	// light at a place is. fires are the fires near the player that burn three or more
	// together, which is a big fire: a hunter that fears one keeps off them.
	moving, hidden, meaty, hunted bool
	winter                        bool    // it is the Long Winter: herds are thin, and hunters do not mind the day
	fireside                      bool    // the player is by a fire, which some hunters will not come to
	sun                           float64 // how bright the day is, from 0 to 1, whatever a hunter makes of it
	noon                          float64 // how high the sun stands, from 0, on the horizon or under it, to 1: none of it in the winter

	// For a pack that is heard before it comes: how many seconds ago its cry was heard,
	// none if it has not been, and where it was heard from, which is where it comes.
	howled [render.Kinds]float64
	howlAt [render.Kinds]geometry.Vector3
	day    float64
	bright func(at geometry.Vector3) float64
	fires  []geometry.Vector3

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
	bitten float64       // points of harm the hunters' bites did the player
	biters []render.Kind // the kinds of those that did it
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
		f := newFauna(w, round)
		f.hatchery(s.Eggs)
		return f
	}
	f := emptyFauna(w)
	defer f.hatchery(s.Eggs)
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
			age: b.Age, fedFor: b.FedFor, limping: b.Limping, mendAt: b.MendAt,
			ridden: b.Ridden, mealDay: b.MealDay, mealsToday: b.MealsToday,
			timer: r.Float64() * 2, rng: r,
			aloft: breeds[kind].moves == gaitSoar && !b.Limping && !(b.Tame && b.Young),
		})
		f.herds = max(f.herds, b.Herd)
	}
	return f
}

// hatchery takes in the eggs a save holds. A pair that roosts is told where its nest is.
func (f *fauna) hatchery(eggs []world.Egg) {
	for _, e := range eggs {
		kind, ok := render.KindNamed(e.Kind)
		if !ok || breeds[kind].egg == 0 {
			continue // a kind of a later version of the game
		}
		f.eggs = append(f.eggs, egg{kind: kind, Pos: e.Pos, herd: e.Herd, wild: e.Wild, ring: e.Ring, warm: e.Warm})
		f.herds = max(f.herds, e.Herd)
		for i := range f.all {
			if c := &f.all[i]; e.Wild && e.Herd != 0 && c.herd == e.Herd && c.kind == kind && breeds[kind].roosts {
				c.nest, c.nested = e.Pos, true
			}
		}
	}
}

// savedEggs returns the eggs as a save holds them.
func (f *fauna) savedEggs() []world.Egg {
	var out []world.Egg
	for _, e := range f.eggs {
		out = append(out, world.Egg{Kind: e.kind.String(), Pos: e.Pos, Herd: e.herd, Wild: e.wild, Ring: e.ring, Warm: e.warm})
	}
	return out
}

// saved returns the beasts as a save holds them. One that is dying is not kept.
func (f *fauna) saved() []world.Beast {
	out := make([]world.Beast, 0, len(f.all))
	for _, c := range f.all {
		if !c.dead && !breeds[c.kind].visits {
			out = append(out, world.Beast{
				Kind: c.kind.String(), Pos: c.Pos, Yaw: c.Yaw, Young: c.young, Herd: c.herd,
				Health: c.health, Regrow: c.regrow, Tame: c.tame, Fed: c.fed,
				Age: c.age, FedFor: c.fedFor, Limping: c.limping, MendAt: c.mendAt,
				Ridden: c.ridden, MealDay: c.mealDay, MealsToday: c.mealsToday,
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
// earth, or for some kinds on bare rock, sand and snow, in a land of its breed, with
// room over it for its body, and not on top of another beast.
func (f *fauna) standing(w *world.World, k render.Kind, young bool, x, z int) (geometry.Vector3, bool) {
	if breeds[k].visits {
		return f.lair(w, k, x, z, f.feet)
	}
	y := w.Ground(x, z) // under any plant
	switch on := w.At(x, y, z); {
	case on == world.Turf:
	case breeds[k].moves == gaitWade:
		return f.wading(w, k, x, z)
	case breeds[k].moves == gaitSwim:
		return f.afloat(w, k, x, z)
	case (breeds[k].moves == gaitFly || breeds[k].moves == gaitSoar) && (on == world.Water || on == world.Peat || on == world.Turf || on == world.Sand):
		y += int(math.Ceil(breeds[k].hover)) - 1 // it is in the air over it: over open ground or water, and not over a tree
	case breeds[k].bare && (on == world.Redrock || on == world.Sand || on == world.Sandstone || on == world.Snow || on == world.Peat):
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
			aloft: b.moves == gaitSoar,
		})
		placed++
	}
	if herd != 0 && placed < 2 {
		f.all = f.all[:before] // one alone is no herd: it found no room for another
		return false
	}
	if herd != 0 && b.nests > 0 {
		f.nest(w, k, herd, f.all[before].Pos)
	}
	return placed > 0
}

// highGround looks at so many columns within roostReach of a place for the highest
// ground a beast could roost on: bare ground or rock, open above, and roostHigh or more
// over a height. It reports whether it found any.
func highGround(w *world.World, rng *rand.Rand, round geometry.Vector3, floor float64, tries int) (geometry.Vector3, bool) {
	best, found := geometry.Vector3{Y: floor + roostHigh}, false
	for range tries {
		x := int(math.Floor(round.X)) + rng.IntN(2*roostReach+1) - roostReach
		z := int(math.Floor(round.Z)) + rng.IntN(2*roostReach+1) - roostReach
		y := w.Ground(x, z)
		if y < 0 || w.At(x, y, z).Info().Shade != world.MaxLight || !w.At(x, y, z).Info().Solid || w.At(x, y+1, z) != world.Air || w.At(x, y+2, z) != world.Air {
			continue // no ground, or a tree top, or no room
		}
		if top := float64(y + 1); top > best.Y {
			best, found = geometry.Vector3{X: float64(x) + 0.5, Y: top, Z: float64(z) + 0.5}, true
		}
	}
	return best, found
}

// nest gives a herd of a kind a nest of one or two eggs, by a chance that is the seed's
// and the herd's and not the fauna's own, so that asking does not change what comes
// after: where the herd was put, or for a kind that roosts on the high ground near it,
// if there is any. It reports whether it made one.
func (f *fauna) nest(w *world.World, k render.Kind, herd int, at geometry.Vector3) bool {
	b := &breeds[k]
	rng := rand.New(rand.NewPCG(uint64(w.Seed), 0x6e657374<<16^uint64(herd)))
	if rng.Float64() >= b.nests {
		return false
	}
	if b.roosts {
		floor, _ := (&beast{Pos: at}).over(w, b.hover+flyTurn)
		high, ok := highGround(w, rng, at, floor, 2*roostTries)
		if !ok {
			return false
		}
		at = high
		for i := range f.all {
			if c := &f.all[i]; c.herd == herd && c.kind == k {
				c.nest, c.nested = at, true
			}
		}
	}
	f.eggs = append(f.eggs, egg{kind: k, Pos: at, herd: herd, wild: true, ring: true})
	if rng.IntN(2) == 0 {
		f.eggs = append(f.eggs, egg{kind: k, Pos: at.Add(geometry.Vector3{X: 0.45, Z: 0.25}), herd: herd, wild: true})
	}
	return true
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

// bite has a beast do the player points of harm this frame.
func (f *fauna) bite(c *beast, points float64) {
	f.bitten += points
	f.biters = append(f.biters, c.kind)
}

// step moves the beasts for dt seconds, and now and then looks them over: those far
// behind the player are let go, and a new group is found a place ahead if there are too
// few of a kind. Those too far off to stand on loaded ground are not moved.
//
// It leaves the cries of the herds that took alarm in sounds, and the beasts that died
// and are gone in fallen, for the game to play and to put down what they leave.
func (f *fauna) step(w *world.World, player geometry.Vector3, dt float64) {
	f.sounds, f.fallen, f.thefts, f.bitten, f.shove = f.sounds[:0], f.fallen[:0], 0, 0, geometry.Vector3{}
	f.biters = f.biters[:0]
	f.treads, f.broken = f.treads[:0], f.broken[:0]
	f.robbed = max(f.robbed-dt, 0)
	for herd, a := range f.alarms {
		if a.left -= dt; a.left <= 0 {
			delete(f.alarms, herd)
		} else {
			f.alarms[herd] = a
		}
	}
	for herd, left := range f.thief {
		if left -= dt; left <= 0 {
			delete(f.thief, herd)
		} else {
			f.thief[herd] = left
		}
	}

	middles, wards := f.middles(false), f.middles(true)
	for i := range f.all {
		c := &f.all[i]
		if c.Pos.Distance(player) >= creatureAwake || c.carrying {
			continue // the game moves one that carries the player
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
		robbed := c.herd != 0 && f.thief[c.herd] > 0
		if robbed && b.temper == herder && !c.young && !c.dead {
			// Its herd was robbed of an egg: it runs at the thief, and not from it.
			c.flee, c.walking, c.Yaw = max(c.flee, 2*rethink), true, yawOf(to)
			c.timer = max(c.timer, rethink)
		}
		if wary := b.wary; wary > 0 && !c.dead && !lured && !robbed {
			if f.sneaking {
				wary *= sneakPart
			}
			if c.Pos.Distance(player) < wary && f.raise(c.herd, player) {
				f.sounds = append(f.sounds, said{b.cry, c.kind, c.voice()}) // the one that saw it calls out
			}
		}
		// One that plays comes to a player who is in the water, and keeps by it.
		if b.plays > 0 && !c.dead && c.flee <= 0 && far < b.plays && far > playNear && swims(w, player) {
			c.walking, c.Yaw, c.hurry = true, yawOf(to), far > playRun
			c.timer = max(c.timer, rethink)
		}
		// A tame beast keeps by the player: it comes after one that goes, runs to catch
		// up when it has fallen far behind, and stands by one that stands.
		if c.tame && !c.dead && c.flee <= 0 && !c.limping && !c.ridden {
			if far > followFar {
				c.walking, c.Yaw, c.hurry = true, yawOf(to), far > followRun
			} else {
				c.walking = false
			}
			c.timer = max(c.timer, rethink)
		}
		// A young one that is the player's grows while its last meal lasts.
		if grows := min(dt, c.fedFor); c.tame && c.young && grows > 0 && !c.dead {
			c.fedFor -= grows
			if c.age += grows; c.age > growTime-1e-6 { // steps of time do not add up to it exactly
				c.young, c.health = false, b.health
			}
		}
		if b.temper == hunter && !c.dead {
			f.hunt(w, i, player, dt)
		}
		if b.temper == guard && !c.dead && !c.limping && !c.ridden {
			f.guard(i, player, dt)
		}
		if c.ridden && b.rams > 0 && !c.dead {
			f.ram(i, player, dt)
		}
		if c.ridden && b.crushes && c.reined > 0 && c.bumped == 0 && !c.dead && f.butt(i, b.blow, player) {
			c.bumped = bumpWait // it ran a hunter down, and goes on
		}
		var middle *geometry.Vector3
		if m, ok := middles[c.herd]; ok && b.temper != loner {
			middle = &m
		}
		if m, ok := wards[c.herd]; ok && b.temper == guard && !c.young {
			middle = &m // a grown guard keeps by its young, and not by the middle of its family
		}
		if b.warms {
			c.pace = coldPace + (1-coldPace)*f.noon
		}
		c.abed = b.roosts && f.sun < sunUp
		if c.tame && b.moves == gaitSoar {
			c.den, c.denned = player, true // one that is the player's keeps over it by day,
			if c.abed {
				c.den = f.home // and roosts by its bed at night
			}
		}
		if !b.warms {
			c.pace = 0
		}
		if c.limping && !c.dead {
			// It lost all its health, and is the player's: it goes home at a limp, and
			// a day after it fell it is whole.
			if c.mendAt == 0 {
				c.mendAt = f.day + mendDays
			}
			home := geometry.Vector3{X: f.home.X - c.Pos.X, Z: f.home.Z - c.Pos.Z}
			c.flee, c.hurry, c.pace, c.timer = 0, false, limpPace, max(c.timer, rethink)
			if c.walking = home.Length() > limpNear; c.walking {
				c.Yaw = yawOf(home)
			}
			if f.day >= c.mendAt {
				c.limping, c.mendAt, c.health, c.pace = false, 0, b.health, 0
			}
		}
		// One whose nest a player has come near dives at it, by day, where hunters are.
		c.diving = c.nested && f.hunted && !c.abed && math.Hypot(player.X-c.nest.X, player.Z-c.nest.Z) < nestGuard
		c.quarry = player.Add(geometry.Vector3{Y: playerHeight * 0.8})
		if c.ridden {
			c.flee, c.hurry, c.timer = 0, false, max(c.timer, rethink) // it has no mind of its own
		}
		c.step(w, dt, middle)
		if c.dived {
			c.dived = false
			f.bite(c, galeDive)
			f.sounds = append(f.sounds, said{b.cry, c.kind, c.voice()})
		}
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
	// A nest left far behind is let go with its herd. An egg the player put down stays.
	f.eggs = slices.DeleteFunc(f.eggs, func(e egg) bool { return e.wild && e.Pos.Distance(player) > creatureGone })
	for k := range render.Kinds {
		switch b := &breeds[k]; {
		case f.count(k) >= f.most(k):
		case b.visits && b.heralds > 0:
			f.herald(w, k, player)
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
			f.bite(c, b.stamp)
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
	if (c.lower || c.curl > 0 || c.ridden) && !c.dead {
		pose.Lower = 1 // a ridden one carries its head low, and its rider sees over it
	}
	if pose.Aloft = c.aloft && !c.dead; pose.Aloft {
		pose.Stride = 0 // its legs trail, and do not walk
	}
	if c.leapt && !c.dead {
		pose.Lower = min(max(c.velY/leapUp, -1), 1) // nose up as it rises, and down as it falls
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
	if c.dead || b.calm || c.curl > 0 {
		return false // curled up, nothing gets through its plates
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
	if b.curls > 0 {
		c.curl, c.flee, c.push, c.walking = b.curls, 0, geometry.Vector3{}, false // it curls up where it is
	}
	if b.health > 0 && !c.limping {
		if c.health -= points; c.health <= 0 {
			return c.falls()
		}
	}
	return false
}

// feed gives the beast a meal from the player's hand, if it is one that can be tamed, is
// not tame yet, takes that food and is not still chewing. Enough meals make it tame.
// It reports whether it ate, and whether that meal tamed it.
func (c *beast) feed(food world.Item, day float64) (ate, tamed bool) {
	b := &breeds[c.kind]
	if c.dead || c.limping || c.chew > 0 || !slices.Contains(b.eats, food) {
		return false, false
	}
	switch {
	case c.tame && c.young && c.fedFor <= 0:
		// A young one that is the player's: a meal is a day of growing, and it takes
		// no other while that lasts.
		c.fedFor, c.chew = dayLength, chewTime
		return true, false
	case !c.tame && b.meals > 0 && (b.temper != guard || c.young) && !c.sated(day):
		if today := int(math.Floor(day)); today != c.mealDay {
			c.mealDay, c.mealsToday = today, 0
		}
		c.fed, c.mealsToday, c.chew = c.fed+1, c.mealsToday+1, chewTime
		c.flee = 0 // what feeds it is nothing to run from
		if c.tame = c.fed >= b.meals; c.tame && c.young {
			c.herd = 0 // it leaves its family for the player
		}
		return true, c.tame
	}
	return false, false
}

// sated reports whether a wild beast that takes only so many meals a day has had them
// on the day given.
func (c *beast) sated(day float64) bool {
	daily := breeds[c.kind].daily
	return !c.tame && daily > 0 && c.mealDay == int(math.Floor(day)) && c.mealsToday >= daily
}

// falls is a beast with no health left. A wild one dies. One that is the player's does
// not: it limps home, and nothing hurts it until it is whole. It reports whether it died.
func (c *beast) falls() (died bool) {
	c.walking, c.flee, c.leap = false, 0, 0
	if c.tame {
		c.limping, c.health, c.aloft = true, 1, false
		return false
	}
	c.dead = true
	return true
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
	// Tar looks like ground, and a beast walks onto it: but one that nothing kills knows
	// it, and keeps out, and so does one that is the player's.
	wary := breeds[c.kind].health == 0 || c.tame
	climb, blocked := lookAheadOf(w, c.Pos, ahead, width, height, max(breeds[c.kind].steps, 1), wary, solid)
	if climb > 0 {
		c.climb = climb
		return c.OnGround || afloat, true
	}
	return false, blocked
}

// lookAheadOf looks at the cell just past the front of a body that stands at pos, width
// across and height high, going the way ahead, and says what to do about it: walk on,
// climb onto it, or turn back. climb is how many blocks high the step is, if there is
// room on top for the whole body, and is never more than steps. A wary body keeps off
// tar. solid says which blocks stop it.
func lookAheadOf(w *world.World, pos, ahead geometry.Vector3, width, height float64, steps int, wary bool, solid func(*world.World, int, int, int) bool) (climb int, blocked bool) {
	front := pos.Add(ahead.Scale(width/2 + lookAhead))
	x, y, z := int(math.Floor(front.X)), int(math.Floor(pos.Y+0.1)), int(math.Floor(front.Z))

	if solid(w, x, y, z) {
		// A block at the height of its feet: a step it can climb, if it is no higher than
		// steps and there is room on top for the whole of it.
		high := 1
		for high < steps && solid(w, x, y+high, z) {
			high++
		}
		for up := high; up < high+int(math.Ceil(height)); up++ {
			if solid(w, x, y+up, z) {
				return 0, true
			}
		}
		return high, true
	}
	if in, under := w.At(x, y, z), w.At(x, y-1, z); in == world.Water || under == world.Water || in == world.Lava || under == world.Lava {
		return 0, true // it keeps out of water, and out of lava
	}
	for down := 1; down <= longDrop; down++ {
		switch below := w.At(x, y-down, z); {
		case below == world.Water || below == world.Lava:
			return 0, true // water under a bank: it keeps out of that too
		case below == world.Tar:
			return 0, wary
		case solid(w, x, y-down, z):
			return 0, false // ground to step onto or down to
		}
	}
	return 0, true // nothing under it for a long way
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
	if c.pace > 0 {
		speed *= c.pace
	}
	if c.leap > 0 {
		speed = springSpeed
	}
	if c.flee > 0 {
		c.flee -= dt
		c.walking, c.timer, speed = true, max(c.timer, rethink), b.run
	}
	if c.ridden {
		speed = c.reined
	}
	if c.curl > 0 {
		// Curled up, it does not move, whatever it would run from.
		c.curl = max(c.curl-dt, 0)
		c.flee, c.walking, c.timer = 0, false, max(c.timer, rethink)
	}
	if b.moves == gaitFly && !c.dead {
		c.fly(w, speed, dt)
		return
	}
	if b.moves == gaitSwim && !c.dead {
		c.swim(w, speed, dt, middle)
		return
	}
	if b.moves == gaitSoar && !c.dead && !c.limping && !(c.tame && c.young) {
		if c.aloft || !c.abed {
			c.soar(w, speed, dt)
			return
		}
		c.walking, c.timer = false, max(c.timer, rethink) // on its roost, until the sun is up
	}
	if b.takes > 0 && !c.dead && (c.aloft || c.flee > 0) {
		// One that walks has taken to its wings: it flies the way it ran, and when it
		// is done with running comes down where there is ground to stand on.
		c.aloft, c.walking = true, true
		if c.flee <= 0 {
			speed = b.run / 2 // it is done with running, and glides
		}
		if high, down := c.wing(w, speed, b.takes, dt, c.flee <= 0, false); down && high < setDown {
			c.aloft, c.walking, c.timer = false, false, rethink
		}
		return
	}
	mired := inTar(w, c.Pos)
	if mired && !c.dead {
		// Tar holds it for good: it goes nowhere, and dies there in time.
		c.flee, c.leap, c.walking, c.timer = 0, 0, false, max(c.timer, rethink)
		if b.health > 0 {
			if c.health -= b.health / tarDeath * dt; c.health <= 0 && !c.limping {
				c.falls()
			}
		}
	}
	if c.timer -= dt; c.timer <= 0 && !c.dead {
		c.decide(middle)
	}

	afloat := inWater(w, c.Pos)
	wades := b.moves == gaitWade
	// One that wades lies at the top of the water: its feet a block under it, in water
	// of any depth. Deeper than that the water bears it up, by lift this step, and once
	// it lies there the water is ground to it.
	borne, lies, lift := false, false, 0.0
	if wades && afloat && c.velY <= 0 && !c.dead {
		x, y, z := int(math.Floor(c.Pos.X)), int(math.Floor(c.Pos.Y)), int(math.Floor(c.Pos.Z))
		for w.At(x, y+1, z) == world.Water {
			y++
		}
		if under := float64(y) - c.Pos.Y; under >= 0 {
			borne, lies, lift = true, under < 0.01, min(under, floatSpeed*dt)
		}
	}
	switch {
	case mired:
		c.velY = -tarSink // it sinks, slowly
	case borne:
		c.velY = 0
	case afloat && !wades:
		c.velY = floatSpeed // it floats up, should it ever fall in
	default:
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
		hop, blocked := c.wayAhead(w, ahead, afloat, stops)
		if wades {
			hop, blocked = c.wadeAhead(w, ahead)
		}
		if c.ridden {
			// It goes where the rider says, and hops a step only on its way ahead.
			hop, blocked = hop && speed > 0, false
		}
		switch {
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
	if c.jumping && c.OnGround && !c.dead {
		c.velY = math.Sqrt(2 * gravity * (b.clears + 0.2)) // the rider has it jump
	}
	move = move.Add(c.push.Scale(dt))
	c.push = c.push.Scale(max(1-knockDrag*dt, 0))
	move.Y = c.velY*dt + lift

	if c.OnGround || afloat {
		c.peak = c.Pos.Y
	}
	if c.ridden {
		height = max(height, b.back-render.SeatHeight+playerHeight) // and the rider's head
	}
	stopped := travelIn(w, &c.Pos, width, height, move, stops)
	c.OnGround = stopped[1] && move.Y < 0 || lies && move.Y == 0
	if stopped[1] {
		c.velY = 0
	}
	if !c.ridden {
		c.peak = max(c.peak, c.Pos.Y) // the jump a rider asks for is no fall: it falls from where it left the ground
	}
	if c.OnGround {
		// A heavy beast is hurt by a fall of more than fallSafe blocks: a part of what
		// health its kind has for each block more, so the bigger it is, the harder it
		// comes down.
		if drop := c.peak - c.Pos.Y; b.heavy && b.health > 0 && drop > fallSafe && !c.dead {
			c.hurt = hurtShow
			if c.health -= b.health * fallPart * (drop - fallSafe); c.health <= 0 {
				c.falls()
			}
		}
		c.peak = c.Pos.Y
	}
	if (stopped[0] || stopped[2]) && c.walking && c.OnGround && !c.ridden {
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
	f.sneaking, f.held = g.player.Crouched, held.Item
	f.bait = g.survival && held.Item.Food() > 0 // in creative there is nothing to lose
	f.moving, f.hidden, f.hunted, f.day = g.paced, g.hidden(), g.survival, firstMorning+g.days
	if m := f.mount(); m != nil {
		f.moving = m.walking // a rider is seen as its mount goes
	}
	// Home, for a beast that limps there: the bed, or with none where a player starts,
	// which is looked for once, when the first beast needs it.
	switch limps := slices.ContainsFunc(f.all, func(c beast) bool { return c.limping }); {
	case g.bed != nil:
		f.home = cellMid(g.bed.X, g.bed.Y, g.bed.Z)
	case limps && f.start == nil:
		at := g.world.Spawn()
		f.start = &at
		fallthrough
	case limps:
		f.home = *f.start
	}
	g.findFires(dt)
	f.meaty = slices.ContainsFunc(g.items[:], func(s world.Stack) bool { return s.Item == world.Meat })
	f.winter, f.fireside = g.winter(), g.byFire()
	f.bright = func(at geometry.Vector3) float64 {
		l := g.lightAt(at)
		if f.winter {
			l &= world.MaxLight // in the winter the ashen day is no light to a hunter: only a fire's is
		}
		return g.sky.Brightness(l)
	}
	g.sky.Set(timeOfDay(g.days)) // the light they keep out of is the light of this hour
	f.sun = g.sky.Brightness(world.Light(world.MaxLight << 4))
	f.noon = 0
	if !f.winter {
		f.noon = max(math.Sin(2*math.Pi*(timeOfDay(g.days)-render.Sunrise)), 0)
	}
	f.step(g.world, g.player.Pos, dt)
	g.seat()
	g.hang()
	for _, k := range f.biters {
		g.meet(beastPicture(k))
	}
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
		tusks := 0
		if !c.young {
			tusks = breeds[c.kind].tusks
		}
		for i := range meat + hides + tusks {
			item := world.Meat
			switch {
			case i >= meat+hides:
				item = world.Tusk
			case i >= meat:
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
		g.meet(beastPicture(c.kind))
		g.hurt(b.bump)
		if g.screen != playing {
			return // the player fainted
		}
	}
}

// How a beast that flies gets about.
const (
	flyTurn  = 12.0 // how far under it it looks for the ground or the water it keeps over
	flyClimb = 4.0  // how fast it rises and sinks to keep its height, at the most, in blocks a second
)

// fly moves a beast that flies for dt seconds, at a speed. It keeps its height over what
// is under it, ground or water, goes the way it faces, hangs in the air between two
// darts, turns from what is in its way and from the edge of its lands, and its wings
// beat all the while.
func (c *beast) fly(w *world.World, speed, dt float64) {
	if c.timer -= dt; c.timer <= 0 {
		// A new mind: it hangs where it is, or darts off some new way.
		c.walking = c.rng.Float64() < 0.65
		c.timer = 0.4 + 1.8*c.rng.Float64()
		c.Yaw = c.rng.Float64()*2*math.Pi - math.Pi
	}
	c.wing(w, speed, breeds[c.kind].hover, dt, false, true)
}

// over returns the height of what is under a beast in the air, ground or water, and
// what block that is: it looks so far down, and finds air if there is nothing that near.
func (c *beast) over(w *world.World, down float64) (floor float64, on world.Block) {
	x, z := int(math.Floor(c.Pos.X)), int(math.Floor(c.Pos.Z))
	for y := int(math.Floor(c.Pos.Y)); float64(y) > c.Pos.Y-down; y-- {
		if under := w.At(x, y, z); under != world.Air && !under.Info().Stands() {
			return float64(y + 1), under
		}
	}
	return c.Pos.Y - down, world.Air
}

// wing moves a beast that is in the air for dt seconds, at a speed, the way it faces. It
// keeps so high over what is under it, ground or water. One that would settle comes
// down instead where what is under it is ground to stand on. One that is bound to its
// lands turns back at the edge of them. It returns how high over what is under it the
// beast then is, and whether it is coming down to settle.
func (c *beast) wing(w *world.World, speed, high, dt float64, settle, bound bool) (over float64, down bool) {
	b := &breeds[c.kind]
	width, height := render.Size(c.kind, c.young)
	floor, on := c.over(w, high+flyTurn)
	if down = settle && on.Info().Solid; down {
		high = 0
	}
	c.velY = min(max(3*(floor+high-c.Pos.Y), -flyClimb), flyClimb)

	move := geometry.Vector3{Y: c.velY * dt}
	if c.walking {
		ahead := mesh.Camera{Yaw: c.Yaw}.Forward()
		move = move.Add(ahead.Scale(speed * dt))
		// It keeps to its lands: at the edge of them it turns back.
		next := c.Pos.Add(ahead.Scale(2))
		if land, ok := w.Land(next.X, next.Z); ok && bound && b.lands != nil && !slices.Contains(b.lands, land) {
			c.Yaw += math.Pi
			move.X, move.Z = 0, 0
		}
	}
	move = move.Add(c.push.Scale(dt))
	c.push = c.push.Scale(max(1-knockDrag*dt, 0))
	if stopped := travelIn(w, &c.Pos, width, height, move, solid); stopped[0] || stopped[2] {
		c.Yaw += math.Pi / 2 * float64(1-2*c.rng.IntN(2)) // something is in its way: it turns aside
	}
	c.OnGround, c.peak = false, c.Pos.Y
	c.Stride += b.gait * dt
	return c.Pos.Y - floor, down
}

// How a beast that soars gets about, and where it roosts.
const (
	soarNear   = 10.0 // the least radius of a circle it turns in, in blocks, and soarWide how much wider one may be
	soarWide   = 10.0
	soarAhead  = 14.0 // how far ahead it looks for the edge of its lands
	soarRoam   = 24.0 // how far from where it was first seen it goes before it turns for home
	soarMind   = 8.0  // the least seconds it keeps to a circle, and soarLong how many more it may
	soarLong   = 14.0
	dipTime    = 6.0 // how long a dip to the water lasts
	dipHeight  = 1.4 // how high over the water it skims
	roostReach = 40  // how far off it looks for high ground to roost on
	roostTries = 24  // how many columns it looks at, each time it looks
	roostHigh  = 6.0 // how much higher than what it is over ground must be for a roost
	roostNear  = 2.5 // how near its roost it begins to come down
	roostClear = 3.0 // how far over its roost it keeps until then
	setDown    = 0.1 // how near the ground one that comes down is down
)

// soar moves a beast that soars for dt seconds, at a speed. It turns in a circle, and
// now and then in another, tighter where the land ahead is none of its own. Over water
// it dips to it at times. At the hour to roost it looks for high ground near, goes to
// it and comes down on it.
func (c *beast) soar(w *world.World, speed, dt float64) {
	b := &breeds[c.kind]
	c.aloft, c.walking = true, true
	floor, on := c.over(w, b.hover+flyTurn)
	if c.timer -= dt; c.timer <= 0 || c.turn == 0 {
		// A new circle: wider or tighter, and one time in three the other way round.
		c.timer = soarMind + soarLong*c.rng.Float64()
		way := math.Copysign(1, c.turn)
		if c.turn == 0 && c.rng.IntN(2) == 0 || c.turn != 0 && c.rng.IntN(3) == 0 {
			way = -way
		}
		c.turn = way * speed / (soarNear + soarWide*c.rng.Float64())
		if c.abed && !c.roosting {
			round := c.Pos
			if c.tame {
				round = c.den
			}
			if c.roost, c.roosting = c.nest, c.nested; !c.nested {
				c.roost, c.roosting = highGround(w, c.rng, round, floor, roostTries)
			}
			c.timer = min(c.timer, 3) // it looks again soon, if it found none
		}
	}
	if !c.abed {
		c.roosting = false
	}
	high, settle := b.hover, false
	if c.mob = max(c.mob-dt, 0); c.diving && c.mob == 0 && !c.roosting {
		// A dive at one who is near its nest: straight at it, as fast as it flies, down
		// to its head. Then it goes up and round again.
		to := c.quarry.Sub(c.Pos)
		c.Yaw, c.dip = yawOf(to), 0
		if to.Length() < diveReach {
			c.dived, c.mob = true, diveWait
		}
		c.wing(w, breeds[c.kind].run, max(c.quarry.Y-floor, 0.5), dt, false, false)
		return
	}
	if c.roosting {
		to := geometry.Vector3{X: c.roost.X - c.Pos.X, Z: c.roost.Z - c.Pos.Z}
		high = max(high, c.roost.Y+roostClear-floor) // over its roost, however high that stands
		if far := to.Length(); far > roostNear {
			c.Yaw = yawOf(to)
		} else {
			settle, speed = true, min(speed, 2*far)
		}
	} else {
		// Where it was first seen in the air is its home: it keeps its circles by it,
		// and over its lands, by turning tighter when it heads away from either.
		if !c.denned {
			c.den, c.denned = c.Pos, true
		}
		home := geometry.Vector3{X: c.den.X - c.Pos.X, Z: c.den.Z - c.Pos.Z}
		if c.tame && home.Length() > keepOver {
			c.Yaw, speed = yawOf(home), b.run // it has fallen behind the player, and comes straight on
		} else {
			c.Yaw += c.turn * dt
			ahead := mesh.Camera{Yaw: c.Yaw}.Forward()
			next := c.Pos.Add(ahead.Scale(soarAhead))
			if land, ok := w.Land(next.X, next.Z); ok && !c.tame && !slices.Contains(b.lands, land) || home.Length() > soarRoam && home.Dot(ahead) < 0 {
				c.Yaw += 2 * c.turn * dt
			}
		}
		switch {
		case on != world.Water:
			c.dip = 0
		case c.dip > 0:
			c.dip, high = c.dip-dt, dipHeight
		case b.dips > 0 && c.rng.Float64() < dt/b.dips:
			c.dip = dipTime
		}
	}
	if over, down := c.wing(w, speed, high, dt, settle, false); down && over < setDown {
		c.aloft, c.walking = false, false // it is on its roost
	}
}

// How a beast that swims gets about.
const (
	swimClimb = 3.0  // how fast it rises and sinks to where it lies, at the most, in blocks a second
	leapUp    = 8.5  // the speed one that leaps leaves the water at: enough to clear it by a block
	playNear  = 3.0  // how near a player one that plays comes
	playRun   = 8.0  // how far from the player it hurries to catch up
	deepMost  = 12   // as deep as water is counted: deeper is as deep as that
	seaGap    = 4.0  // how far apart those that swim alone are put
	swimAhead = 1.0  // how far past its own front one that swims looks at the depth
	swimTurn  = 0.25 // how much of a turn it turns aside at the edge of its water when it runs
)

// deepAt returns how deep the water in the column at x, z is, from its top, which must
// be open to the sky: none where the top of the column is not water. Water deeper than
// deepMost is counted as that deep.
func deepAt(w *world.World, x, z int) (deep int) {
	for y := w.Ground(x, z); deep < deepMost && y >= 0 && w.At(x, y, z) == world.Water; y-- {
		deep++
	}
	return deep
}

// swims reports whether a body at a place is in the water: its feet are in it, or just
// over it, as a swimmer's are that comes up for air.
func swims(w *world.World, at geometry.Vector3) bool {
	return inWater(w, at) || inWater(w, at.Sub(geometry.Vector3{Y: 0.6}))
}

// afloat returns where a beast that swims could lie in the column at x, z: at the top
// of water deep enough for it, with the sky over it, and room between it and the others.
func (f *fauna) afloat(w *world.World, k render.Kind, x, z int) (geometry.Vector3, bool) {
	b := &breeds[k]
	if deepAt(w, x, z) < b.deep {
		return geometry.Vector3{}, false
	}
	pos := geometry.Vector3{X: float64(x) + 0.5, Y: float64(w.Ground(x, z)+1) - b.draft, Z: float64(z) + 0.5}
	gap := seaGap
	if width, _ := render.Size(k, false); b.group[1] > 1 {
		gap = width + 0.4
	}
	for i := range f.all {
		if f.all[i].Pos.Distance(pos) < gap {
			return geometry.Vector3{}, false
		}
	}
	return pos, true
}

// swim moves a beast that swims for dt seconds, at a speed. It lies as far under the
// top of the water as its kind does, or lower when it runs or lurks, goes the way it
// faces, and turns from water too shallow for it. One that leaps leaves the water now
// and then as it swims, and falls back into it.
func (c *beast) swim(w *world.World, speed, dt float64, middle *geometry.Vector3) {
	b := &breeds[c.kind]
	width, height := render.Size(c.kind, c.young)
	if c.timer -= dt; c.timer <= 0 {
		c.decide(middle)
	}
	x, y, z := int(math.Floor(c.Pos.X)), int(math.Floor(c.Pos.Y)), int(math.Floor(c.Pos.Z))
	wet := w.At(x, y, z) == world.Water
	lies := 0.0 // the height its feet are at when it lies where it means to
	if wet {
		top := y
		for w.At(x, top+1, z) == world.Water {
			top++
		}
		lies = float64(top+1) - b.draft
		if c.flee > 0 || b.temper == hunter && c.keen <= 0 && !c.stalking {
			lies -= b.sinks
		}
	}
	switch {
	case c.leapt && wet && c.velY < 0 && c.Pos.Y <= lies+b.draft:
		c.leapt = false // it is back in the water
		fallthrough
	case wet && !c.leapt:
		c.velY = min(max(3*(lies-c.Pos.Y), -swimClimb), swimClimb)
		if b.leaps > 0 && c.walking && c.flee <= 0 && math.Abs(lies-c.Pos.Y) < 0.2 && c.rng.Float64() < dt/b.leaps {
			c.leapt, c.velY = true, leapUp
		}
	default:
		c.velY = max(c.velY-gravity*dt, -fallSpeed) // in the air, or left dry
	}

	move := geometry.Vector3{Y: c.velY * dt}
	if c.walking {
		ahead := mesh.Camera{Yaw: c.Yaw}.Forward()
		next := c.Pos.Add(ahead.Scale(width/2 + swimAhead))
		switch shallow := deepAt(w, int(math.Floor(next.X)), int(math.Floor(next.Z))) < b.deep; {
		case !shallow:
			move = move.Add(ahead.Scale(speed * dt))
		case c.keen > 0 || c.stalking:
			// What it is after is beyond water it can swim in: it waits at the edge.
		case c.flee > 0:
			c.Yaw += 2 * math.Pi * swimTurn * float64(1-2*c.rng.IntN(2)) // running, it turns aside
		default:
			c.turnBack()
		}
	}
	move = move.Add(c.push.Scale(dt))
	c.push = c.push.Scale(max(1-knockDrag*dt, 0))
	if stopped := travelIn(w, &c.Pos, width, height, move, solid); (stopped[0] || stopped[2]) && c.walking {
		c.Yaw += math.Pi / 2 * float64(1-2*c.rng.IntN(2)) // something is in its way: it turns aside
	} else if stopped[1] && c.velY > 0 {
		c.velY = 0
	}
	c.OnGround, c.peak = false, c.Pos.Y
	if c.walking {
		c.Stride += b.gait * speed * dt
	}
}

// coldPace is how much of its speed a beast that the sun warms has with no sun on it.
const coldPace = 0.4

// wadeAhead says what a beast that wades does about what is just ahead of it, as
// wayAhead does for one that walks. Water is its ground: it goes on through it, climbs
// out of it only after its prey, and at other times turns back from the bank.
func (c *beast) wadeAhead(w *world.World, ahead geometry.Vector3) (hop, blocked bool) {
	width, height := render.Size(c.kind, c.young)
	front := c.Pos.Add(ahead.Scale(width/2 + lookAhead))
	x, y, z := int(math.Floor(front.X)), int(math.Floor(c.Pos.Y+0.1)), int(math.Floor(front.Z))
	hunting := c.keen > 0 || c.leap > 0 || c.homing
	switch {
	case solid(w, x, y, z):
		// The bank, or the floor rising: it climbs a step of it, if it is after
		// something or the step is under water, and there is room.
		for up := 1; up < 1+int(math.Ceil(height)); up++ {
			if solid(w, x, y+up, z) {
				return false, true
			}
		}
		if !hunting && w.At(x, y+1, z) != world.Water {
			return false, true // it would leave the water
		}
		c.climb = 1
		return c.OnGround, true
	case hunting:
		return false, false
	}
	// Open ahead: water, which it keeps to, or a way down into it.
	for down := 0; down <= 2; down++ {
		if w.At(x, y-down, z) == world.Water {
			return false, false
		}
		if solid(w, x, y-down, z) {
			break
		}
	}
	return false, true
}

// wading returns where a beast that wades could lie in the column at x, z: at the top
// of still water one or two deep, in one of its lands or at the edge of one, with
// nothing over it.
func (f *fauna) wading(w *world.World, k render.Kind, x, z int) (geometry.Vector3, bool) {
	top := w.Ground(x, z)
	if w.At(x, top, z) != world.Water {
		return geometry.Vector3{}, false
	}
	floor := top
	for floor > top-3 && w.At(x, floor, z) == world.Water {
		floor--
	}
	if deep := top - floor; deep > 2 || !solid(w, x, floor, z) || w.At(x, top+1, z) != world.Air || w.At(x, top+2, z) != world.Air {
		return geometry.Vector3{}, false
	}
	// It is wider than a block: the water must be open all round the column, too.
	_, height := render.Size(k, false)
	for dz := -1; dz <= 1; dz++ {
		for dx := -1; dx <= 1; dx++ {
			for up := 1; up <= int(math.Ceil(height)); up++ {
				if solid(w, x+dx, floor+up, z+dz) {
					return geometry.Vector3{}, false
				}
			}
		}
	}
	inLand := func(x, z int) bool { return slices.Contains(breeds[k].lands, world.BiomeAt(w.Seed, x, z)) }
	if !inLand(x, z) && !inLand(x+5, z) && !inLand(x-5, z) && !inLand(x, z+5) && !inLand(x, z-5) {
		return geometry.Vector3{}, false
	}
	pos := geometry.Vector3{X: float64(x) + 0.5, Y: float64(top), Z: float64(z) + 0.5}
	for i := range f.all {
		if f.all[i].Pos.Distance(pos) < 6 {
			return geometry.Vector3{}, false
		}
	}
	return pos, true
}

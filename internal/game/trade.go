package game

import (
	"math/rand/v2"
	"slices"

	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/sound"
	"github.com/kyrokohan/skyspear/internal/world"
)

// Trade with the people of the camps is by showing. Hold a thing and press E on a person:
// if their camp wants it, they show over their head what they would give for it, and E
// again while they show it trades. A crouched player gives it instead. Each trade, gift
// and help makes the camp trust the player more.
const (
	offerTime  = 5.0  // how long a person shows an offer, in seconds
	thanksTime = 3.0  // and their thanks, a hand
	tradeTrust = 1    // what a trade adds to a camp's trust
	giftTrust  = 3    // and a gift
	helpTrust  = 5    // and a hunter or a guard beast killed near it
	helpReach  = 48.0 // how near a camp's fire a beast must die for its people to see it
)

// worth is what a thing is worth to the people of the camps, in the cheapest things.
var worth = map[world.Item]int{
	world.Meat: 2, world.Flint: 2, world.Ochre: 2, world.Resin: 2,
	world.Hide: 3, world.CookedMeat: 3,
	world.ClayPot.Item(): 4,
	world.Amber:          6, world.CrystalShard: 6, world.HardSpear: 6, world.Obsidian: 6,
	world.Tusk: 8, world.FurCloak: 8,
}

// What a camp may want and may give. Each wants one precious thing and two plain ones,
// and gives one cheap thing, so it has something to give for anything it wants, and two
// dear ones.
var (
	preciousWants = []world.Item{world.Amber, world.Tusk, world.CrystalShard}
	plainWants    = []world.Item{world.Meat, world.Hide, world.Flint, world.Ochre}
	cheapGoods    = []world.Item{world.Flint, world.Resin}
	dearGoods     = []world.Item{world.CookedMeat, world.ClayPot.Item(), world.HardSpear, world.Obsidian, world.FurCloak}
)

// market is what the people of one camp want and what they give for it.
type market struct {
	wants, goods []world.Item
}

// campMarket returns the market of the camp whose fire is at a cell, as the seed makes it.
func campMarket(seed int64, fire world.Cell) market {
	rng := rand.New(rand.NewPCG(uint64(seed)^0x74726164, uint64(uint32(fire.X))<<32|uint64(uint32(fire.Z))))
	m := market{wants: []world.Item{preciousWants[rng.IntN(len(preciousWants))]}}
	for _, i := range rng.Perm(len(plainWants))[:2] {
		m.wants = append(m.wants, plainWants[i])
	}
	for _, c := range cheapGoods {
		if !slices.Contains(m.wants, c) {
			m.goods = append(m.goods, c)
			break
		}
	}
	for _, i := range rng.Perm(len(dearGoods))[:2] {
		m.goods = append(m.goods, dearGoods[i])
	}
	return m
}

// offer returns what the camp gives for one of a thing it wants: as many as it is worth
// of the dearest of its goods worth no more than it.
func (m market) offer(want world.Item) (give world.Item, n int) {
	for _, g := range m.goods {
		if worth[g] <= worth[want] && worth[g] > worth[give] {
			give = g
		}
	}
	return give, worth[want] / worth[give]
}

// saying is what a person shows over their head: a thing, with how many when there are
// more than one, or a hand, which is thanks.
type saying struct {
	item  world.Item
	count int
	hand  bool
}

var thanks = saying{hand: true}

// says has a person show something over their head for a time, after which they show
// what they want again.
func (one *person) says(s saying, seconds float64) {
	one.say, one.sayFor = s, seconds
}

// aimPerson finds the nearest person a ray from origin along dir, one long, meets before
// limit.
func (g *Game) aimPerson(origin, dir geometry.Vector3, limit float64) (i int, ok bool) {
	best := limit
	for j := range g.people {
		one := &g.people[j]
		at, height := one.Pos, playerHeight
		if one.seated {
			at.Y -= render.SeatHeight - seatLift
			height = render.SeatHeight + 0.6
		}
		if t, hit := rayBox(origin, dir, at, playerWidth*aimWider, height); hit && t < best && one.away == 0 {
			best, i, ok = t, j, true
		}
	}
	return i, ok
}

// showTo shows the person aimed at what is in the player's hand. A thing their camp wants
// gets an offer, and when they show that offer already, a trade. A crouched player gives
// it to them. For anything else they show what they want.
func (g *Game) showTo(i int) {
	one := &g.people[i]
	held := g.items[g.selected]
	switch {
	case one.tribe:
		// Shown what a job is done with, one at the fire takes that job, and puts down
		// what they carried for the last. A spear they are given. Anything else, and they
		// greet the player.
		if j, ok := jobFor(held.Item); ok && held.Count > 0 && !one.follows {
			if j != one.job {
				g.unload(one)
				one.job, one.errand, one.hurry, one.workIn = j, errandNone, false, 0
			}
			if j == sentry && one.carry[0].Count == 0 {
				one.carry[0] = world.Stack{Item: held.Item, Count: 1, Wear: held.Wear}
				if g.survival {
					g.items.takeOne(g.selected)
				}
			}
			one.says(saying{item: held.Item}, thanksTime)
			g.tell(jobTold[j])
			return
		}
		one.says(thanks, thanksTime)
		return
	case held.Count == 0 && one.asks:
		g.invite(i)
		return
	}
	m := campMarket(g.world.Seed, one.fire)
	if held.Count == 0 || !slices.Contains(m.wants, held.Item) {
		one.says(saying{item: one.want}, 0)
		return
	}
	give, n := m.offer(held.Item)
	offered := saying{item: give, count: n}
	switch {
	case g.player.Crouched:
		g.trust[one.fire] += giftTrust
	case one.say == offered && one.sayFor > 0:
		if left := g.items.add(world.Stack{Item: give, Count: n}); left > 0 {
			g.putDown(world.Stack{Item: give, Count: left})
		}
		g.trust[one.fire] += tradeTrust
	default:
		one.says(offered, offerTime)
		return
	}
	// A trade or a gift: they take one, thank the player, and want the next thing.
	if g.survival {
		g.items.takeOne(g.selected)
	}
	g.ears.PlayNear(sound.Pickup, 0.6)
	one.want = m.wants[(slices.Index(m.wants, one.want)+1)%len(m.wants)]
	one.says(thanks, thanksTime)
}

// helped is a beast the player killed: a hunter or a guard killed near a camp is help to
// its people, who saw it, and trust the player the more.
func (g *Game) helped(c *beast) {
	if t := breeds[c.kind].temper; t != hunter && t != guard {
		return
	}
	var seen []world.Cell
	for i := range g.people {
		one := &g.people[i]
		if one.tribe || cellMid(one.fire.X, one.fire.Y, one.fire.Z).Distance(c.Pos) > helpReach {
			continue
		}
		if !slices.Contains(seen, one.fire) {
			seen = append(seen, one.fire)
			g.trust[one.fire] += helpTrust
		}
		one.says(thanks, thanksTime)
	}
	if len(seen) > 0 {
		g.tell("the people of the camp saw you kill the " + c.kind.String())
	}
}

// savedCamps returns the trust of each camp as a save holds it, in the order of their
// fires, so that a save of the same world is the same.
func (g *Game) savedCamps() []world.Camp {
	var camps []world.Camp
	for fire, trust := range g.trust {
		camps = append(camps, world.Camp{Fire: fire, Trust: trust})
	}
	slices.SortFunc(camps, func(a, b world.Camp) int { return compareCells(a.Fire, b.Fire) })
	return camps
}

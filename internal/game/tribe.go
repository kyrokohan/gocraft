package game

import (
	"math"
	"math/rand/v2"

	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/assets"
	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/world"
)

// The tribe: people who came from the camps to live at the player's fire. A camp that
// trusts the player enough has one of its people ask to come, with a fire over their
// head, and E with an empty hand says yes. They follow the player as a tame beast does,
// and one left far behind finds their own way, and is at the fire by the next morning.
const (
	askTrust  = 8   // the trust at which a camp asks to send one of its people, and twice it a second
	maxTribe  = 6   // how many live at the player's fire
	perCamp   = 2   // and how many of them from one camp
	homeReach = 8   // how near the bed a hearth must be to be the player's fire, in blocks
	personRun = 5.0 // how fast one who follows runs to catch up, in blocks a second
)

// asking is what one shows who asks to come to the player's fire: a fire.
var asking = saying{item: world.Fire.Item()}

// homeFire returns the player's fire: the hearth nearest the bed it last lay on, within
// homeReach of it, cold or burning. With no bed, or none near it, there is none.
func (g *Game) homeFire() *world.Cell {
	b := g.bed
	if b == nil || g.world.At(b.X, b.Y, b.Z) != world.FurBed {
		return nil
	}
	var best *world.Cell
	near := float64(homeReach) + 1e-9
	for y := b.Y - homeReach; y <= b.Y+homeReach; y++ {
		for z := b.Z - homeReach; z <= b.Z+homeReach; z++ {
			for x := b.X - homeReach; x <= b.X+homeReach; x++ {
				far := math.Sqrt(float64((x-b.X)*(x-b.X) + (y-b.Y)*(y-b.Y) + (z-b.Z)*(z-b.Z)))
				if far < near && isHearth(g.world.At(x, y, z)) {
					best, near = &world.Cell{X: x, Y: y, Z: z}, far
				}
			}
		}
	}
	return best
}

// seeHome finds the player's fire again where the ground round the bed is loaded. Where
// it is not, the fire is as it was last seen.
func (g *Game) seeHome() {
	switch {
	case g.bed == nil:
		g.home = nil
	case g.world.Loaded(g.bed.X, g.bed.Z):
		g.home = g.homeFire()
	}
}

// invite is E with an empty hand to one who asks to come. With a fire they come. With
// none they show what is missing: a bed, or a hearth by it.
func (g *Game) invite(i int) {
	one := &g.people[i]
	g.seeHome()
	switch {
	case g.bed == nil || g.world.Loaded(g.bed.X, g.bed.Z) && g.world.At(g.bed.X, g.bed.Y, g.bed.Z) != world.FurBed:
		one.says(saying{item: world.FurBed.Item()}, offerTime)
	case g.home == nil:
		one.says(saying{item: world.Hearth.Item()}, offerTime)
	default:
		g.join(i, *g.home)
		g.tell("one of the people of the camp comes to live at your fire")
	}
}

// join has a person of a camp come to live at a fire. They follow the player there, and
// have a place by it of their own.
func (g *Game) join(i int, home world.Cell) {
	n := 0
	for _, one := range g.people {
		if one.tribe {
			n++
		}
	}
	one := &g.people[i]
	one.tribe, one.follows, one.asks, one.fire = true, true, false, home
	one.seat = 2 * math.Pi * float64(n) / maxTribe
	one.says(thanks, thanksTime)
}

// settleTribe has, once a second, those of the tribe live by the player's fire where it
// is now; one who follows and was left far behind go their own way; one on their way
// who has come by the morning be at the fire; and the camps whose trust is enough ask
// to send one more, while there is room there.
func (g *Game) settleTribe() {
	g.seeHome()
	from, n := map[world.Cell]int{}, 0
	for i := range g.people {
		one := &g.people[i]
		if !one.tribe {
			continue
		}
		from[one.camp]++
		n++
		if g.home != nil {
			one.fire = *g.home
		}
		switch {
		case one.away > 0:
			if g.days >= one.away {
				one.arrive(g.world)
			}
		case one.follows && one.Pos.Distance(g.player.Pos) > creatureGone:
			one.away = nextMorning(g.days)
		}
	}
	asked := map[world.Cell]bool{}
	for i := range g.people {
		one := &g.people[i]
		one.asks = !one.tribe && !asked[one.camp] && n < maxTribe && from[one.camp] < perCamp &&
			g.trust[one.camp] >= askTrust*(from[one.camp]+1)
		asked[one.camp] = asked[one.camp] || one.asks
	}
}

// nextMorning returns the first sunrise after days, in days.
func nextMorning(days float64) float64 {
	return math.Floor(firstMorning+days-render.Sunrise) + 1 + render.Sunrise - firstMorning
}

// arrive puts one who found their own way at their place by the fire they live at,
// standing a while.
func (one *person) arrive(w *world.World) {
	at := cellMid(one.fire.X, one.fire.Y, one.fire.Z).Add(geometry.Vector3{X: seatOut * math.Cos(one.seat), Z: seatOut * math.Sin(one.seat)})
	at.Y = groundAt(w, at, one.fire.Y)
	one.Pos, one.away, one.follows, one.walking, one.hurry = at, 0, false, false, false
	one.timer, one.velY = 2+4*one.rng.Float64(), 0
}

// follow has one who follows keep by the player, dt seconds on: they come after a
// player who goes, run to catch up when far behind, and stand by one who stands. When
// the player has brought them to the fire they live at, they stay there.
func (one *person) follow(player geometry.Vector3, dt float64) {
	if mid := cellMid(one.fire.X, one.fire.Y, one.fire.Z); math.Hypot(one.Pos.X-mid.X, one.Pos.Z-mid.Z) < campRoam {
		one.follows, one.walking, one.hurry, one.timer = false, false, false, 2+4*one.rng.Float64()
		return
	}
	one.seated = false
	if one.aside > 0 {
		one.aside -= dt
		return
	}
	to := player.Sub(one.Pos)
	to.Y = 0
	far := to.Length()
	if one.walking, one.hurry = far > followFar, far > followRun; one.walking {
		one.Yaw = yawOf(to)
	}
}

// savedTribe returns who came to the player's fire as a save holds them.
func (g *Game) savedTribe() []world.Member {
	var tribe []world.Member
	for _, one := range g.people {
		if one.tribe {
			tribe = append(tribe, world.Member{Camp: one.camp, Look: one.look, Pos: one.Pos, Yaw: one.Yaw, Follows: one.follows, Away: one.away})
		}
	}
	return tribe
}

// loadTribe takes who came to the player's fire from a save. Each lives at the fire as
// it was last seen, or, with none, where they stand.
func (g *Game) loadTribe(s world.Meta) {
	g.home, g.people = s.Home, nil
	for i, m := range s.Tribe {
		one := person{
			Pos: m.Pos, Yaw: m.Yaw, look: min(m.Look, assets.PersonLooks-1), camp: m.Camp, fire: world.Cell{X: int(math.Floor(m.Pos.X)), Y: int(math.Floor(m.Pos.Y)), Z: int(math.Floor(m.Pos.Z))},
			tribe: true, follows: m.Follows, away: m.Away, seat: 2 * math.Pi * float64(i) / maxTribe,
			rng: rand.New(rand.NewPCG(uint64(g.world.Seed), uint64(i))),
		}
		if g.home != nil {
			one.fire = *g.home
		}
		g.people = append(g.people, one)
	}
}

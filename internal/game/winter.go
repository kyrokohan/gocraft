package game

import (
	"math"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/world"
)

// The Long Winter, and warmth. For winterDays after the Skyspear strikes every land is
// cold, as the tundra and the crags always are. In the cold a body loses its warmth,
// and with none left it loses health. A fire near, or the water of a hot spring, gives
// warmth back, and a fur cloak that is carried keeps the cold of the day off. The open
// ground is white with frost, the herds are thin, and the hunters of the night come by
// day. Warmth is in points, as health is, and times are in seconds.
const (
	maxWarmth = 20.0
	coldTime  = 120.0 // how long the cold takes to take all of a body's warmth, with no fire and no furs
	warmTime  = 20.0  // how long a fire or a hot spring takes to give it all back
	backTime  = 40.0  // and how long a land that is not cold takes
	chillTime = 4.0   // from one point of health lost to the cold to the next, with no warmth left
	warmReach = 6.0   // how near a fire warms
	frostTime = 150.0 // how long the frost takes to come over the ground when the winter begins
)

// winter reports whether it is the Long Winter: the days after the Skyspear struck.
func (g *Game) winter() bool {
	after := g.greatFall() - fallLasts
	return after >= 0 && after < winterDays*dayLength
}

// frost returns how white the open ground is, from 0 to 1: the frost comes in the first
// minutes of the winter, lies all through it, and goes as the ash thins after.
func (g *Game) frost() float64 {
	after := g.greatFall() - fallLasts
	switch winter := winterDays * dayLength; {
	case after < 0:
		return 0
	case after < winter:
		return min(after/frostTime, 1)
	default:
		return max(1-(after-winter)/(ashAfter*dayLength), 0)
	}
}

// cold reports whether the player is in the cold: anywhere in the winter, and at any
// time in a cold land.
func (g *Game) cold() bool {
	return g.winter() || g.world.Cold(g.player.Pos.X, g.player.Pos.Z)
}

// byFire reports whether something burns within warmReach of the player: a fire, a
// hearth that is lit, a torch that stands, lava, or the hot core of a star. The fires
// near are looked for once a second, for the hunters, and those are the ones asked.
func (g *Game) byFire() bool {
	for _, fire := range g.burning {
		if fire.Distance(g.player.Pos) <= warmReach {
			return true
		}
	}
	return false
}

// warm runs the player's warmth for dt seconds, in survival. By a fire or in a hot
// spring it comes back; in the cold it goes, but not by day from one who carries a fur
// cloak, and half as fast by night; and elsewhere it comes back slowly. With none left
// the cold takes a point of health every chillTime, and it can make the player faint.
func (g *Game) warm(dt float64) {
	p := g.player.Pos
	m := g.fauna.mount()
	switch {
	case g.byFire() || inWater(g.world, p) && g.world.Spring(p.X, p.Z):
		g.warmth = min(g.warmth+maxWarmth/warmTime*dt, maxWarmth)
	case g.cold() && m != nil && breeds[m.kind].hairy:
		// On the back of a beast in long hair the cold does not reach the rider.
	case g.cold():
		loss := maxWarmth / coldTime
		if g.items.count([]world.Item{world.FurCloak}) > 0 {
			loss /= 2
			if t := timeOfDay(g.days); t >= render.Sunrise && t <= render.Sunset {
				loss = 0
			}
		}
		g.warmth = max(g.warmth-loss*dt, 0)
	default:
		g.warmth = min(g.warmth+maxWarmth/backTime*dt, maxWarmth)
	}
	if g.warmth > 0 {
		g.chill = 0
		return
	}
	if g.chill += dt; g.chill >= chillTime {
		g.chill = 0
		g.hurt(1)
	}
}

// most returns how many wild beasts of a kind there may be round the player: what its
// breed says, and in the winter half of that for the beasts that live there, but never
// fewer than go together. The hunters that come and go are as many as ever.
func (f *fauna) most(k render.Kind) int {
	b := &breeds[k]
	if !f.winter || b.visits {
		return b.most
	}
	return max(int(math.Ceil(float64(b.most)/2)), b.group[0])
}

package game

import (
	"math"

	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/sound"
	"github.com/kyrokohan/skyspear/internal/world"
)

// A geyser stands in the middle of a hot spring. It spouts by the clock of the world,
// each at its own moment of a round they all keep, so that nothing of it is in a save:
// the ground trembles, and then water and steam go up, and a player in the middle of
// the pool goes up with them. Times are in seconds and distances in blocks.
const (
	geyserEvery = 30.0 // from one spout of a geyser to its next
	geyserSpout = 3.0  // how long it spouts
	geyserWarn  = 2.0  // how long before a spout the ground trembles
	geyserThrow = 26.0 // how fast it throws a player up, in blocks a second: enough for twelve blocks
	geyserReach = 1.3  // how far from its middle a player is thrown
	geyserNear  = 48.0 // how far off a geyser is seen and heard to spout
	geyserFelt  = 20.0 // how far off the tremble before a spout is felt
	geyserShake = 0.3  // how hard the ground trembles beside one about to spout
	steamEvery  = 0.08 // from one puff of a spout to the next
	steamRise   = 8.0  // how fast its steam goes up, in blocks a second
)

// steamItem is what a puff of steam is drawn as a piece of: white.
var steamItem = world.Snow.Item()

// round returns how far through its round a geyser is, by the clock of the world: it
// spouts for the first geyserSpout seconds of each geyserEvery. Where it stands says
// where in the round it is, so that two do not go together.
func (g *Game) round(vent world.Cell) float64 {
	own := float64(uint32(vent.X*73856093^vent.Z*19349663)&0xffff) / 0x10000
	return math.Mod(g.days*dayLength+geyserEvery*own, geyserEvery)
}

// geysers runs the geysers near the player for dt seconds: the tremble before a spout,
// the spout's sound and steam, and the throw of a player who is in the middle of the
// pool when it goes.
func (g *Game) geysers(dt float64) {
	p := &g.player
	for _, vent := range g.world.Geysers(p.Pos.X, p.Pos.Z, geyserNear) {
		at := cellMid(vent.X, vent.Y, vent.Z)
		far := math.Hypot(at.X-p.Pos.X, at.Z-p.Pos.Z)
		switch round := g.round(vent); {
		case round < geyserSpout:
			if round < dt {
				g.ears.PlayFar(sound.Geyser, at, 1, geyserNear)
			}
			if math.Floor(round/steamEvery) != math.Floor((round-dt)/steamEvery) {
				g.puffOf(steamItem, at.Add(geometry.Vector3{Y: 0.5}), 0.45, 1.6, steamRise)
			}
			if far <= geyserReach && p.Pos.Y > float64(vent.Y)-1.5 && p.Pos.Y < float64(vent.Y)+3 && p.velY < geyserThrow/4 {
				p.velY = geyserThrow
			}
		case round > geyserEvery-geyserWarn && far < geyserFelt:
			g.quake = max(g.quake, geyserShake*(1-far/geyserFelt))
		}
	}
}

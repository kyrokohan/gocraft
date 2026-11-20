package game

import (
	"math"
	"testing"

	"glonk.dev/engine"
	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/sound"
	"github.com/kyrokohan/skyspear/internal/world"
)

// atMountain is a game in survival in the world of seed 7, with no beasts and none to
// come, its player at the foot of the fire mountain nearest where that world begins,
// and the cell the first geyser of that mountain spouts from.
func atMountain(t *testing.T) (*Game, world.Cell) {
	t.Helper()
	g, err := New(Options{Seed: 7})
	if err != nil {
		t.Fatal(err)
	}
	g.Play()
	g.begin(true)
	g.fauna.all, g.fauna.timer = nil, math.Inf(1)
	crater, _, _ := g.world.Mountain(g.player.Pos.X, g.player.Pos.Z)
	g.world.Follow(float64(crater.X), float64(crater.Z))
	g.world.LoadAll()
	vents := g.world.Geysers(float64(crater.X), float64(crater.Z), 80)
	if len(vents) == 0 {
		t.Fatal("the mountain of seed 7 has no geyser")
	}
	stand(g, geometry.Vector3{X: float64(vents[0].X) + 0.5, Y: float64(vents[0].Y), Z: float64(vents[0].Z) + 0.5})
	hush(g)
	return g, vents[0]
}

// stand puts the player of a game at a place, as if it had always stood there: it has
// not fallen from where it was.
func stand(g *Game, at geometry.Vector3) {
	g.player.Pos, g.peak = at, at.Y
}

// quiet sets the clock of a game so that the geyser at a cell is so many seconds into
// its round.
func quiet(g *Game, vent world.Cell, into float64) {
	g.days = 0
	g.days = (math.Mod(into-g.round(vent)+geyserEvery, geyserEvery)) / dayLength
}

func TestAPlayerInAHotSpringMendsInHalfTheTime(t *testing.T) {
	mends := func(in bool) float64 {
		g, vent := atMountain(t)
		quiet(g, vent, 8) // between two spouts, for the whole of the test
		if !in {
			out := g.player.Pos
			out.X += 9 // on the ash, past the rings
			out.Y = float64(g.world.Ground(int(math.Floor(out.X)), int(math.Floor(out.Z)))) + 1
			stand(g, out)
		}
		g.health, g.food = 6, maxFood
		hold(t, g, seconds(4*mendTime+0.5))
		return g.health - 6
	}
	in, out := mends(true), mends(false)
	if out != 4 || in != 2*out {
		t.Errorf("in %v seconds a player mended %v points in a hot spring and %v out of it: wanted twice as many in it", 4*mendTime, in, out)
	}

	// A hungry player does not starve twice as fast there.
	g, vent := atMountain(t)
	quiet(g, vent, 8)
	g.health, g.food = 10, 0
	hold(t, g, seconds(2*mendTime+0.5))
	if lost := 10 - g.health; lost != 2 {
		t.Errorf("a starving player in a hot spring lost %v points in %v seconds, wanted the 2 it loses anywhere", lost, 2*mendTime)
	}
}

func TestAGeyserThrowsAPlayerOnItTwelveBlocksUpWhenItGoes(t *testing.T) {
	g, vent := atMountain(t)
	quiet(g, vent, 10)
	hold(t, g, seconds(3))
	if up := g.player.Pos.Y - float64(vent.Y); math.Abs(up) > 0.6 || steam(g) != 0 || heard(g)[sound.Geyser] != 0 {
		t.Fatalf("between two spouts the player is %.1f over the water, with %d puffs of steam, and %d spouts heard", up, steam(g), heard(g)[sound.Geyser])
	}

	// The ground trembles before it goes.
	quiet(g, vent, geyserEvery-geyserWarn+0.2)
	hold(t, g, seconds(1))
	if g.quake == 0 {
		t.Error("the ground does not tremble under a player on a geyser that is about to spout")
	}

	quiet(g, vent, geyserEvery-0.1)
	highest, puffs := -math.MaxFloat64, 0
	for range seconds(geyserSpout + 1.5) {
		hold(t, g, 1)
		highest, puffs = max(highest, g.player.Pos.Y-float64(vent.Y)), max(puffs, steam(g))
	}
	if highest < 10 || highest > 15 {
		t.Errorf("the geyser threw the player %.1f blocks up, wanted about twelve", highest)
	}
	if heard(g)[sound.Geyser] == 0 {
		t.Errorf("the spout was not heard: %v", g.ears.Playing())
	}
	if puffs < 5 {
		t.Errorf("at the most %d puffs of steam were in the air as it spouted", puffs)
	}
	// It came down in the pool, which broke the fall.
	hold(t, g, seconds(4))
	if g.health != maxHealth {
		t.Errorf("the player who rode the geyser and fell back into its pool lost %v points", maxHealth-g.health)
	}

	// A player beside it is not thrown.
	g, vent = atMountain(t)
	stand(g, g.player.Pos.Add(geometry.Vector3{X: 2.5, Y: 1.2}))
	quiet(g, vent, 10)
	hold(t, g, seconds(1)) // to come to rest, in the pool or on its first ring
	rest := g.player.Pos.Y
	quiet(g, vent, geyserEvery-0.1)
	high := 0.0
	for range seconds(geyserSpout) {
		hold(t, g, 1)
		high = max(high, g.player.Pos.Y-rest)
	}
	if high > 0.3 {
		t.Errorf("a player two and a half blocks from the geyser was thrown %.1f up", high)
	}
	// Each geyser has its own moment of the round.
	if a, b := g.round(vent), g.round(world.Cell{X: vent.X + 37, Y: vent.Y, Z: vent.Z - 12}); math.Abs(a-b) < 0.01 {
		t.Error("two geysers in different places spout at the same moment")
	}
}

func TestTheDreadCannotFollowIntoALavaTube(t *testing.T) {
	g, _ := atMountain(t)
	mouth, chamber, ok := g.world.Tube(g.player.Pos.X, g.player.Pos.Z)
	if !ok {
		t.Fatal("the mountain has no tube")
	}
	// The player is well inside, and the dread is put at the mouth, in the open.
	in := geometry.Vector3{
		X: float64(mouth.X) + 0.5 + 0.55*float64(chamber.X-mouth.X),
		Y: float64(mouth.Y),
		Z: float64(mouth.Z) + 0.5 + 0.55*float64(chamber.Z-mouth.Z),
	}
	stand(g, in)
	if g.world.At(int(math.Floor(in.X)), int(math.Floor(in.Y)), int(math.Floor(in.Z))) != world.Air {
		t.Fatalf("the place inside the tube at %+v is %v", in, g.world.At(int(math.Floor(in.X)), int(math.Floor(in.Y)), int(math.Floor(in.Z))))
	}
	out := geometry.Vector3{
		X: float64(mouth.X) + 0.5 - 0.3*float64(chamber.X-mouth.X),
		Z: float64(mouth.Z) + 0.5 - 0.3*float64(chamber.Z-mouth.Z),
	}
	out.Y = float64(g.world.Ground(int(math.Floor(out.X)), int(math.Floor(out.Z)))) + 1
	one := dread(out.X, out.Z)
	one.Pos.Y = out.Y
	g.fauna.all = []beast{one}
	g.days = thirdDay
	g.player.Yaw = yawOf(geometry.Vector3{X: float64(chamber.X - mouth.X), Z: float64(chamber.Z - mouth.Z)})
	nearest, noticed := math.Inf(1), false
	for i := range seconds(24) {
		key := engine.Key('w') // the player walks up and down the tube, and is heard
		if i/20%2 == 1 {
			key = 's'
		}
		hold(t, g, 1, key)
		nearest = min(nearest, g.fauna.all[0].Pos.Distance(g.player.Pos))
		noticed = noticed || g.fauna.all[0].keen > 0
	}
	if !noticed {
		t.Fatal("the dread at the mouth did not notice a player who walks about in the tube")
	}
	if g.health != maxHealth || nearest < 3 {
		t.Errorf("the dread came to %.1f blocks from a player in the tube, who has %v health: wanted it kept out", nearest, g.health)
	}
}

// steam counts the puffs of steam in the air.
func steam(g *Game) (n int) {
	for _, b := range g.bits {
		if b.smoke && b.item == steamItem {
			n++
		}
	}
	return n
}

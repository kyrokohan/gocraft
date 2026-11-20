package game

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"testing"

	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/sound"
	"github.com/kyrokohan/skyspear/internal/world"
)

// dryWay is a bearing from the mountain of seed 7 along which the land is dry out to 130
// blocks from its crater, so that a beast can stand there.
const dryWay = 1.96

// outFrom returns the place on the ground so far from the middle of a crater, along a
// bearing, with the world of the game brought in round it.
func outFrom(g *Game, crater world.Cell, bearing, far float64) geometry.Vector3 {
	x, z := float64(crater.X)+0.5+far*math.Cos(bearing), float64(crater.Z)+0.5+far*math.Sin(bearing)
	g.world.Follow(x, z)
	g.world.LoadAll()
	return geometry.Vector3{X: x, Y: float64(g.world.Ground(int(math.Floor(x)), int(math.Floor(z)))) + 1, Z: z}
}

// onCone is a game in survival whose player stands on the side of the fire mountain of
// seed 7, 28 blocks from the middle of its crater: too far from any geyser to feel it.
// With it come the crater and the bearing the player stands at.
func onCone(t *testing.T) (*Game, world.Cell, float64) {
	t.Helper()
	g, vent := atMountain(t)
	crater, _, _ := g.world.Mountain(g.player.Pos.X, g.player.Pos.Z)
	bearing := math.Atan2(float64(vent.Z-crater.Z), float64(vent.X-crater.X))
	stand(g, outFrom(g, crater, bearing, 28))
	g.days = 1
	return g, crater, bearing
}

// eruptionAt sets the clock of a game to so many seconds after the mountain near its
// player first began to erupt: fewer than none is before it.
func eruptionAt(t *testing.T, g *Game, into float64) {
	t.Helper()
	for day := 4.0; day < 8; day += 0.001 {
		if began, _, ok := g.world.Erupted(g.player.Pos.X, g.player.Pos.Z, day); ok {
			g.days = began + into/dayLength
			return
		}
	}
	t.Fatal("the mountain did not erupt in the first week")
}

// flatFrom returns how far a place is from a cell, along the ground.
func flatFrom(at geometry.Vector3, c world.Cell) float64 {
	return math.Hypot(at.X-float64(c.X)-0.5, at.Z-float64(c.Z)-0.5)
}

func TestNoMountainEruptsInTheFirstFourDaysOrFarFromThePlayer(t *testing.T) {
	g, crater, bearing := onCone(t)
	calm := func(when string) {
		t.Helper()
		ash, fire := g.ash()
		if g.quake != 0 || len(g.missiles) != 0 || ash != 0 || fire != 0 || heard(g)[sound.Tremor] != 0 {
			t.Errorf("%s the ground shakes by %.2f, %d rocks fly, the sky holds %.2f of ash and %.2f of fire, and %d rumbles were heard: wanted none of it",
				when, g.quake, len(g.missiles), ash, fire, heard(g)[sound.Tremor])
		}
	}
	for _, day := range []float64{0.2, 2.2, 3.95} {
		g.days = day
		hold(t, g, seconds(3))
		calm(fmt.Sprintf("on day %.2f", day))
	}

	// F3 says how long it is until it does.
	g.debug = true
	if text, _ := drawn(t, g); !strings.Contains(text, "erupts  in ") {
		t.Errorf("on day 3.95, on a mountain, the frame numbers do not say when it erupts:\n%s", text)
	}

	// It erupts, and a player on it knows.
	eruptionAt(t, g, eruptWarn+20)
	hold(t, g, seconds(3))
	if ash, _ := g.ash(); g.quake == 0 || len(g.missiles) == 0 || ash == 0 || heard(g)[sound.Tremor] == 0 {
		t.Fatalf("on a mountain that erupts the ground shakes by %.2f, %d rocks fly, and the sky holds %.2f of ash", g.quake, len(g.missiles), ash)
	}

	// A player too far off does not.
	stand(g, outFrom(g, crater, bearing, eruptReach+20))
	g.missiles, g.quake = nil, 0
	hush(g)
	eruptionAt(t, g, eruptWarn+20)
	hold(t, g, seconds(3))
	calm(fmt.Sprintf("%.0f blocks from a mountain that erupts", flatFrom(g.player.Pos, crater)))
	if text, _ := drawn(t, g); strings.Contains(text, "erupts") {
		t.Errorf("too far from a mountain to know of it, the frame numbers say when it erupts:\n%s", text)
	}
}

func TestTheGroundTremblesMoreAndMoreForAMinuteAndBeastsNearTheCraterRun(t *testing.T) {
	g, crater, _ := onCone(t)
	shake := func(into float64) float64 {
		eruptionAt(t, g, into)
		g.quake = 0
		hold(t, g, seconds(1))
		if len(g.missiles) != 0 {
			t.Errorf("%v seconds after the ground began to tremble a rock flies: wanted none for %v", into, eruptWarn)
		}
		return g.quake
	}
	before, early, mid, late := shake(-5), shake(5), shake(30), shake(57)
	if before != 0 || early <= 0 || mid <= early || late <= mid || late > 1 {
		t.Errorf("the ground shakes by %.3f before, and by %.3f, %.3f and %.3f through the minute: wanted none, and then more and more", before, early, mid, late)
	}
	eruptionAt(t, g, 40)
	hold(t, g, seconds(3))
	if heard(g)[sound.Tremor] == 0 {
		t.Errorf("the mountain was not heard to rumble: %v", g.ears.Playing())
	}

	// Beasts within 100 blocks of the crater run from it, and one farther off does not.
	g, crater, _ = onCone(t)
	stand(g, outFrom(g, crater, dryWay, 110))
	near, far := giant(render.Mossback, 0, 0), giant(render.Mossback, 0, 0)
	near.Pos, far.Pos = outFrom(g, crater, dryWay, 92), outFrom(g, crater, dryWay, 126)
	outFrom(g, crater, dryWay, 110) // the world is round the player again
	for _, c := range []beast{near, far} {
		if b := g.world.At(int(math.Floor(c.Pos.X)), int(c.Pos.Y)-1, int(math.Floor(c.Pos.Z))); !b.Info().Solid {
			t.Fatalf("the beast %.0f blocks out stands on %v: the test wants dry land along its bearing", flatFrom(c.Pos, crater), b)
		}
	}
	g.fauna.all = []beast{near, far}
	eruptionAt(t, g, 10)
	hold(t, g, seconds(5))
	a, b := &g.fauna.all[0], &g.fauna.all[1]
	if gone := flatFrom(a.Pos, crater) - 92; a.flee <= 0 || gone < 1.5 {
		t.Errorf("a mossback 92 blocks from the crater went %.1f blocks farther from it in five seconds of the tremor, and runs for %.1f s more: wanted it to run", gone, a.flee)
	}
	if moved := b.Pos.Distance(far.Pos); b.flee > 0 || moved > 0.5 {
		t.Errorf("a mossback 126 blocks from the crater moved %.1f blocks and runs for %.1f s: wanted it to stay", moved, b.flee)
	}
}

func TestBurningRocksLandWithinSeventyBlocksOfTheCraterAndNoneNearTheBed(t *testing.T) {
	g, crater, bearing := onCone(t)
	g.begin(false) // in creative, where a rock does not hurt
	at := outFrom(g, crater, bearing+math.Pi/2, 45)
	g.bed = &world.Cell{X: int(math.Floor(at.X)), Y: int(at.Y), Z: int(math.Floor(at.Z))}
	outFrom(g, crater, bearing, 28) // the world is round the player again

	eruptionAt(t, g, eruptWarn)
	under := map[world.Cell]world.Block{} // what lies under each fire a rock left
	for range seconds(eruptTime + 10) {
		hold(t, g, 1)
		for _, e := range g.embers {
			if g.world.At(e.At.X, e.At.Y, e.At.Z) != world.Fire {
				t.Fatalf("the fire a rock left at %v is %v", e.At, g.world.At(e.At.X, e.At.Y, e.At.Z))
			}
			under[e.At] = g.world.At(e.At.X, e.At.Y-1, e.At.Z)
		}
	}
	if len(g.missiles) != 0 {
		t.Errorf("%d rocks still fly ten seconds after the mountain was done", len(g.missiles))
	}
	kinds := map[world.Block]int{}
	for c, b := range under {
		kinds[b]++
		mid := cellMid(c.X, c.Y, c.Z)
		if far := flatFrom(mid, crater); far > 70 {
			t.Errorf("a rock landed %.0f blocks from the crater, wanted none past 70", far)
		}
		if far := flatFrom(mid, *g.bed); far < bedSafe {
			t.Errorf("a rock landed %.0f blocks from the bed, wanted none within %v", far, bedSafe)
		}
	}
	if len(under) < 40 {
		t.Errorf("%d rocks landed in %v seconds, wanted 40 or more", len(under), eruptTime)
	}
	if kinds[world.ObsidianRock] == 0 || kinds[world.Basalt] == 0 || kinds[world.ObsidianRock]+kinds[world.Basalt] != len(under) {
		t.Errorf("the rocks left %v: wanted obsidian and dark rock, and nothing else", kinds)
	}
	if kinds[world.ObsidianRock] >= kinds[world.Basalt] {
		t.Errorf("the rocks left %d of obsidian and %d of dark rock: wanted obsidian the rarer", kinds[world.ObsidianRock], kinds[world.Basalt])
	}
}

func TestABurningRockHurtsWhatItHitsAndLeavesRockUnderAFireThatBurnsOut(t *testing.T) {
	g, crater, bearing := onCone(t)
	fall := func(onto geometry.Vector3) {
		t.Helper()
		g.missiles = []missile{rockTo(onto.Add(geometry.Vector3{X: 3, Y: 14}), onto)}
		hold(t, g, seconds(5))
		if len(g.missiles) != 0 {
			t.Fatalf("five seconds on, the rock still flies at %+v", g.missiles[0].pos)
		}
	}

	// On the player, who is too hungry to mend.
	g.food = mendFood - 1
	fall(g.player.Pos.Add(geometry.Vector3{Y: 1}))
	if lost := maxHealth - g.health; lost != rockHarm || rockHarm != 10 {
		t.Errorf("a burning rock took %v points from the player, wanted 10", lost)
	}
	if len(g.embers) != 0 {
		t.Errorf("a rock that broke on the player left %d fires", len(g.embers))
	}

	// On a beast.
	one := giant(render.Bugler, 0, 0)
	one.Pos = outFrom(g, crater, bearing, 38)
	g.fauna.all = []beast{one}
	fall(one.Pos.Add(geometry.Vector3{Y: 1.5}))
	if c := &g.fauna.all[0]; c.health != breeds[render.Bugler].health-rockHarm {
		t.Errorf("a burning rock left a bugler %v points of %v, wanted %v fewer", c.health, breeds[render.Bugler].health, rockHarm)
	}
	g.fauna.all = nil

	// On the ground: it leaves a block of rock, and a fire on it that burns for a minute.
	ground := outFrom(g, crater, bearing, 34)
	x, y, z := int(math.Floor(ground.X)), int(ground.Y), int(math.Floor(ground.Z))
	hush(g)
	fall(ground)
	if b := g.world.At(x, y, z); b != world.ObsidianRock && b != world.Basalt {
		t.Fatalf("where a rock came down lies %v, wanted obsidian or dark rock", b)
	}
	if b := g.world.At(x, y+1, z); b != world.Fire || len(g.embers) != 1 {
		t.Fatalf("over the rock is %v, with %d fires kept: wanted a fire", b, len(g.embers))
	}
	if heard(g)[sound.Thud] == 0 {
		t.Errorf("the rock was not heard to land: %v", g.ears.Playing())
	}
	hold(t, g, seconds(emberTime-8)) // it landed in the five seconds of the fall
	if g.world.At(x, y+1, z) != world.Fire {
		t.Errorf("the fire of a rock went out before its %v seconds were up", emberTime)
	}
	hold(t, g, seconds(8))
	if b := g.world.At(x, y+1, z); b != world.Air || len(g.embers) != 0 || g.world.At(x, y, z) == world.Air {
		t.Errorf("%v seconds after the rock landed its fire is %v with %d kept, and the rock is %v: wanted the fire out and the rock there", emberTime, b, len(g.embers), g.world.At(x, y, z))
	}

	// Near the bed it leaves nothing, and in water it leaves nothing.
	near := outFrom(g, crater, bearing, 31)
	g.bed = &world.Cell{X: int(math.Floor(near.X)) + 10, Y: int(near.Y), Z: int(math.Floor(near.Z))}
	fall(near)
	if b := g.world.At(int(math.Floor(near.X)), int(near.Y), int(math.Floor(near.Z))); b != world.Air || len(g.embers) != 0 {
		t.Errorf("a rock that came down ten blocks from the bed left %v and %d fires", b, len(g.embers))
	}
	g.bed = nil
	vent := g.world.Geysers(float64(crater.X), float64(crater.Z), 80)[0]
	fall(cellMid(vent.X, vent.Y, vent.Z))
	if b := g.world.At(vent.X, vent.Y, vent.Z); b != world.Water || len(g.embers) != 0 {
		t.Errorf("a rock that fell in a hot spring left %v and %d fires, wanted the water as it was", b, len(g.embers))
	}
}

func TestAshDarkensTheSkyWhileAMountainEruptsAndClearsHalfADayAfter(t *testing.T) {
	g, crater, bearing := onCone(t)
	at := func(into float64) (ash, fire float64) {
		eruptionAt(t, g, into)
		return g.ash()
	}
	clear := func() bool {
		drawn(t, g)
		return g.sky.Fog.Color == render.NewSky(timeOfDay(g.days)).Fog.Color
	}
	end := eruptWarn + eruptTime
	if ash, fire := at(eruptWarn - 1); ash != 0 || fire != 0 || !clear() {
		t.Errorf("while the ground only trembles the sky holds %.2f of ash and %.2f of fire, wanted none", ash, fire)
	}
	if ash, fire := at(eruptWarn + eruptTime/2); ash < 0.9 || fire != 1 || clear() {
		t.Errorf("half way through, the sky holds %.2f of ash and %.2f of fire, and the frame is clear: %v", ash, fire, clear())
	}
	if ash, fire := at(end + 0.25*dayLength); math.Abs(ash-0.5) > 0.05 || fire != 0 || clear() {
		t.Errorf("a quarter of a day after, the sky holds %.2f of ash and %.2f of fire, wanted half the ash and no fire", ash, fire)
	}
	if ash, _ := at(end + 0.5*dayLength + 1); ash != 0 || !clear() {
		t.Errorf("half a day after, the sky holds %.2f of ash, wanted it clear", ash)
	}

	// From far off there is less of it.
	stand(g, outFrom(g, crater, bearing, (eruptReach+eruptFull)/2))
	if ash, _ := at(eruptWarn + eruptTime/2); math.Abs(ash-0.5) > 0.05 {
		t.Errorf("%.0f blocks off, the sky holds %.2f of the ash of a mountain that erupts, wanted half", flatFrom(g.player.Pos, crater), ash)
	}
}

func TestAnEruptionSavedHalfWayGoesOnWhenTheWorldIsOpened(t *testing.T) {
	path := filepath.Join(t.TempDir(), "world")
	g, _, _ := onCone(t)
	g.begin(false)
	g.opts.SavePath = path
	eruptionAt(t, g, eruptWarn+30)
	for range seconds(20) {
		if hold(t, g, 1); len(g.embers) >= 2 {
			break
		}
	}
	if len(g.embers) < 2 {
		t.Fatalf("twenty seconds of an eruption left %d fires", len(g.embers))
	}
	fires, days := append([]world.Ember(nil), g.embers...), g.days
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}

	back := savedGame(t, path)
	back.Play()
	if back.days != days || len(back.embers) != len(fires) {
		t.Fatalf("the world came back on day %v with %d fires, wanted day %v and %d", back.days, len(back.embers), days, len(fires))
	}
	hold(t, back, seconds(4))
	if ash, fire := back.ash(); len(back.missiles) == 0 || back.quake == 0 || ash == 0 || fire != 1 {
		t.Errorf("four seconds after it was opened %d rocks fly, the ground shakes by %.2f and the sky holds %.2f of ash: wanted the eruption to go on", len(back.missiles), back.quake, ash)
	}
	hold(t, back, seconds(emberTime))
	for _, e := range fires {
		if b := back.world.At(e.At.X, e.At.Y, e.At.Z); b == world.Fire {
			t.Errorf("the fire at %v, saved with %.0f s left, still burns %v seconds on", e.At, e.Left, emberTime+4)
		}
	}
}

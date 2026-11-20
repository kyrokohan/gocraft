package game

import (
	"math"
	"math/rand/v2"
	"path/filepath"
	"testing"

	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/sound"
	"github.com/kyrokohan/skyspear/internal/world"
)

// thirdDay is the days of play to the noon of the third day, when the dread is about.
const thirdDay = 2.2

// wide is a plain like plain, a hundred blocks each way from the origin: room for the
// dread to come from as far as it does.
func wide() *world.World {
	w := world.New()
	for z := -100; z <= 100; z++ {
		for x := -100; x <= 100; x++ {
			w.Set(x, 5, z, world.Stone)
		}
	}
	return w
}

// dread is the dread on the plain at x, z, that goes about as it will.
func dread(x, z float64) beast {
	return beast{
		kind: render.Dread, health: breeds[render.Dread].health,
		Pos:   geometry.Vector3{X: x, Y: 6, Z: z},
		timer: math.Inf(1), rng: rand.New(rand.NewPCG(13, uint64(x*100+z))),
	}
}

// hunting is a game in survival on the third day, on the plain, with the beasts given
// and none to come, whose player stands at x = 0.5 and the z given.
func hunting(t *testing.T, z float64, beasts ...beast) *Game {
	t.Helper()
	g := among(t, z, beasts...)
	g.days = thirdDay
	return g
}

func TestNoDreadComesBeforeTheThirdDayAndNeverTwo(t *testing.T) {
	g := inSurvival(t)
	g.world = wide()
	g.player.Pos = geometry.Vector3{X: 0.5, Y: 6, Z: 0.5}
	g.fauna.all = nil
	g.items[0] = world.Stack{Item: world.Torch.Item(), Count: 3} // which keeps the quickclaws of the second night off
	for _, days := range []float64{0, 0.9, 1.5, 1.9} {           // the first two days, by day and by night
		g.days = days
		hold(t, g, seconds(40))
		if n := len(only(g.fauna.all, render.Dread)); n != 0 {
			t.Fatalf("after %.2f days of the world a dread has come", days)
		}
		g.fauna.dismiss()
	}

	g.days = thirdDay
	hold(t, g, seconds(60))
	one := only(g.fauna.all, render.Dread)
	if len(one) != 1 {
		t.Fatalf("on the third day %d dreads came in a minute, wanted one", len(one))
	}
	hold(t, g, seconds(120))
	if n := len(only(g.fauna.all, render.Dread)); n > 1 {
		t.Errorf("%d dreads are about at once", n)
	}

	// It came from far off, where it is felt before it is seen.
	g = inSurvival(t)
	g.world = wide()
	g.player.Pos = geometry.Vector3{X: 0.5, Y: 6, Z: 0.5}
	g.fauna.all, g.days = nil, thirdDay
	b := breeds[render.Dread]
	for range seconds(60) {
		if hold(t, g, 1); len(only(g.fauna.all, render.Dread)) == 1 {
			break
		}
	}
	came := only(g.fauna.all, render.Dread)
	if len(came) != 1 {
		t.Fatal("no dread came in a minute of the third day")
	}
	if d := came[0].Pos.Distance(g.player.Pos); d < b.comes[0]-1 || d > b.comes[1]+1 {
		t.Errorf("the dread came out %.0f blocks off, wanted %v to %v", d, b.comes[0], b.comes[1])
	}

	// Never in creative.
	c := onFloor(t)
	c.world = wide()
	c.fauna.all, c.days = nil, thirdDay
	hold(t, c, seconds(60))
	if n := len(only(c.fauna.all, render.Dread)); n != 0 {
		t.Errorf("%d dreads came in creative", n)
	}
}

func TestTheDreadRunsDownAWalkingPlayerAndBitesHard(t *testing.T) {
	b := breeds[render.Dread]
	g := hunting(t, 0.5, dread(0.5, -14.5)) // behind the player, who walks away from it
	hold(t, g, seconds(14), 'w')
	c := g.fauna.all[0]
	if d := c.Pos.Distance(g.player.Pos); d > 5 {
		t.Errorf("after fourteen seconds the dread is %.1f behind a walking player, wanted it to have caught up", d)
	}
	if lost := maxHealth - g.health; lost < b.bite || g.screen != playing && g.screen != fainted {
		t.Errorf("the player lost %v health to the dread, wanted a bite of %v or more", lost, b.bite)
	}
	if heard(g)[sound.Roar] == 0 {
		t.Errorf("the dread that saw its prey did not roar: %v", g.ears.Playing())
	}
	if b.run <= walkSpeed || b.bite < 3*breeds[render.Quickclaw].bite/2 || b.health < 10*breeds[render.Quickclaw].health {
		t.Errorf("the dread runs at %v, bites for %v and has %v health: wanted far more than a quickclaw", b.run, b.bite, b.health)
	}
}

func TestTheDreadLosesAPlayerWhoHidesAndSeesOneWhoStandsFromFar(t *testing.T) {
	w := plain()
	f := &fauna{rng: rand.New(rand.NewPCG(1, 2)), hunted: true, day: 3}
	c := dread(0.5, 0.5)
	at := func(z float64) geometry.Vector3 { return geometry.Vector3{X: 0.5, Y: 6, Z: z} }
	if !f.notices(w, &c, at(35)) {
		t.Error("the dread does not see a player who stands 35 blocks off")
	}
	f.sneaking, f.hidden = true, true
	if f.notices(w, &c, at(8)) {
		t.Error("the dread sees a player crouched in tall ferns eight blocks off")
	}
	if !f.notices(w, &c, at(2)) {
		t.Error("the dread does not see a player crouched in tall ferns at its feet")
	}
	f.day = 1.9
	f.sneaking, f.hidden = false, false
	if f.about(render.Dread) || !f.about(render.Quickclaw) {
		t.Errorf("on the second night the dread is about: %v, and quickclaws: %v", f.about(render.Dread), f.about(render.Quickclaw))
	}
}

func TestTheDreadCannotFollowThroughAGapTwoBlocksWide(t *testing.T) {
	through := func(gap int) float64 {
		g := hunting(t, 20.5, dread(0.5, 2.5))
		for x := -30; x <= 30; x++ {
			if x >= 0 && x < gap {
				continue // the gap, which the player stands beyond
			}
			for y := 6; y < 14; y++ {
				g.world.Set(x, y, 12, world.Stone)
			}
		}
		g.player.Pos.X = float64(gap) / 2
		g.fauna.all[0].Pos.X = float64(gap) / 2
		hold(t, g, seconds(15))
		return g.fauna.all[0].Pos.Z
	}
	if z := through(2); z > 12 {
		t.Errorf("the dread got through a gap two blocks wide, to z=%.1f", z)
	}
	if z := through(4); z < 13 {
		t.Errorf("the dread did not get through a gap four blocks wide: it is at z=%.1f", z)
	}
	if w, _ := render.Size(render.Dread, false); w <= 2 {
		t.Errorf("the dread's box is %v wide, wanted more than two blocks", w)
	}
}

func TestATorchDoesNotHoldTheDreadAndThreeFiresTogetherDo(t *testing.T) {
	b := breeds[render.Dread]
	with := func(fires []world.Block) (nearest, health float64) {
		g := hunting(t, 0.5, dread(0.5, 26.5))
		g.days = thirdDay + 0.5 // the night of the third day
		for i, fire := range fires {
			x := -2 + 2*i // behind the player, out of the way the dread comes
			g.world.Set(x, 6, -2, world.Stone)
			g.world.Set(x, 7, -2, fire)
		}
		nearest = math.Inf(1)
		for range seconds(20) {
			hold(t, g, 1)
			nearest = min(nearest, g.fauna.all[0].Pos.Distance(g.player.Pos))
		}
		return nearest, g.health
	}
	if _, health := with([]world.Block{world.Torch}); health == maxHealth {
		t.Error("a player by one torch was not bitten by the dread")
	}
	if _, health := with([]world.Block{world.HearthLit, world.HearthLit}); health == maxHealth {
		t.Error("a player by two hearths was not bitten by the dread: two are not a big fire")
	}
	for _, fires := range [][]world.Block{
		{world.HearthLit, world.HearthLit, world.HearthLit},
		{world.Torch, world.Fire, world.HearthLit},
	} {
		nearest, health := with(fires)
		if health != maxHealth {
			t.Errorf("a player by a fire of %v was bitten: health %v", fires, health)
		}
		if nearest < b.fears-2.5 || nearest > b.fears+4 {
			t.Errorf("the dread came to %.1f from a player by a fire of %v, wanted it held about %v off", nearest, fires, b.fears)
		}
	}
	// Cold hearths are no fire.
	if _, health := with([]world.Block{world.Hearth, world.Hearth, world.Hearth}); health == maxHealth {
		t.Error("three cold hearths held the dread off")
	}
}

func TestTheDreadHuntsAHerdWhenItHasNotNoticedThePlayer(t *testing.T) {
	g := hunting(t, -38.5)
	g.fauna = herdOf(6, 0.5, 30.5)
	g.fauna.all = append(g.fauna.all, dread(0.5, 8.5)) // 22 blocks from the herd, 47 from the player
	hold(t, g, seconds(1))
	if c := g.fauna.all[6]; c.keen > 0 {
		t.Fatal("the dread noticed a player 47 blocks behind it, who stands still")
	}
	fed := false
	for range seconds(40) {
		hold(t, g, 1)
		for _, c := range g.fauna.all {
			fed = fed || c.kind == render.Dread && c.feeding > 0
		}
	}
	if !fed {
		t.Errorf("in forty seconds the dread made no kill of a herd 22 blocks off: %d buglers left", len(only(g.fauna.all, render.Bugler)))
	}
	if g.fauna.alarms[1].left <= 0 && len(only(g.fauna.all, render.Bugler)) == 6 {
		t.Error("the herd took no alarm and lost none")
	}
	if g.health != maxHealth {
		t.Errorf("the player, far off, lost %v health", maxHealth-g.health)
	}
}

func TestQuickclawsRunFromTheDread(t *testing.T) {
	g := hunting(t, -38.5, dread(0.5, 20.5), claw(4.5, 26.5))
	g.days = thirdDay + 0.5 // the night
	from := g.fauna.all[1].Pos.Distance(g.fauna.all[0].Pos)
	hold(t, g, seconds(4))
	claws := only(g.fauna.all, render.Quickclaw)
	if len(claws) != 1 {
		t.Fatalf("%d quickclaws are left, wanted the one", len(claws))
	}
	if d := claws[0].Pos.Distance(only(g.fauna.all, render.Dread)[0].Pos); d < from+6 || claws[0].flee <= 0 {
		t.Errorf("a quickclaw %.1f from the dread is %.1f from it four seconds on, running %v: wanted it to run", from, d, claws[0].flee > 0)
	}
}

func TestTheDreadGoesWhenItsTimeIsUpAndTheNextComesADayOrTwoOn(t *testing.T) {
	b := breeds[render.Dread]
	g := hunting(t, 0.5, dread(0.5, 40.5))
	g.world = wide()
	g.fauna.all[0].about = b.stays - 1
	hold(t, g, seconds(3))
	if c := g.fauna.all[0]; !c.leaving {
		t.Fatal("the dread is still about after its time")
	}
	hold(t, g, seconds(40))
	if n := len(only(g.fauna.all, render.Dread)); n != 0 {
		t.Fatalf("forty seconds after it turned to go the dread is still there")
	}
	if w := g.fauna.wait[render.Dread]; w < dayLength*b.every[0]-41 || w > dayLength*b.every[1] {
		t.Errorf("the next dread may come in %.0f s, wanted %v to %v days of %v s", w, b.every[0], b.every[1], dayLength)
	}
	g.fauna.timer = 1 // beasts come again
	hold(t, g, seconds(60))
	if n := len(only(g.fauna.all, render.Dread)); n != 0 {
		t.Error("another dread came within a minute of the last")
	}

	// One that is killed, or let go when the player sleeps, is waited for as long.
	g = hunting(t, 0.5, dread(0.5, 40.5))
	g.fauna.dismiss()
	if len(g.fauna.all) != 0 || g.fauna.wait[render.Dread] < dayLength*b.every[0] {
		t.Errorf("after a sleep: %d beasts, and the next dread in %.0f s", len(g.fauna.all), g.fauna.wait[render.Dread])
	}
}

func TestTheDreadIsNotSavedAndAWorldOpenedAgainIsGivenAWhile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "world")
	g := inSurvival(t)
	g.opts.SavePath = path
	g.days = thirdDay + 1
	g.fauna.all = []beast{dread(20.5, 30.5), giant(render.Bugler, 25.5, 25.5)}
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}
	back := savedGame(t, path)
	if n := len(only(back.fauna.all, render.Dread)); n != 0 || len(back.fauna.all) != 1 {
		t.Errorf("the world came back with %d dreads among %d beasts, wanted the bugler alone", n, len(back.fauna.all))
	}
	if w := back.fauna.wait[render.Dread]; w < 60 || w > dayLength {
		t.Errorf("a world opened again may have a dread in %.0f s, wanted a while, and under a day", w)
	}
}

func TestTheDreadsStepsAreFeltBeforeItIsSeen(t *testing.T) {
	b := breeds[render.Dread]
	g := hunting(t, -38.5, dread(0.5, 12.5)) // 51 blocks off: within its quake, beyond its sight of one who stands... no: it is seen
	g.fauna.all[0].walking, g.fauna.all[0].Yaw = true, math.Pi/2
	peak := 0.0
	for range seconds(6) {
		g.fauna.all[0].walking, g.fauna.all[0].timer = true, math.Inf(1)
		hold(t, g, 1)
		peak = max(peak, g.quake)
	}
	if heard(g)[sound.Thud] == 0 || peak == 0 {
		t.Errorf("a dread that walks 50 blocks off: %d thuds, and the ground shook %.3f", heard(g)[sound.Thud], peak)
	}
	if b.quake < 60 || b.quake <= breeds[render.Cloudgrazer].quake {
		t.Errorf("the dread is felt from %v blocks, wanted 60, and farther than a cloudgrazer", b.quake)
	}
}

func TestTheDreadIsTooStrongToKillInTheOpen(t *testing.T) {
	b := breeds[render.Dread]
	g := hunting(t, 0.5, dread(0.5, 30.5))
	spears := 0
	for ; !g.fauna.all[0].dead && spears < 200; spears++ {
		g.wound(0, world.HardSpear.Damage()+thrownMore, 1, false)
	}
	if spears < 30 {
		t.Errorf("the dread died of %d thrown spears, wanted more than a player can throw before it is on them", spears)
	}
	hold(t, g, seconds(fallTime+0.5))
	meat, hides := 0, 0
	for _, d := range g.drops {
		switch d.Stack.Item {
		case world.Meat:
			meat += d.Stack.Count
		case world.Hide:
			hides += d.Stack.Count
		}
	}
	if meat != b.meat || hides != b.hide || b.meat < 12 {
		t.Errorf("a dead dread left %d meat and %d hides, wanted %d and %d", meat, hides, b.meat, b.hide)
	}
	if w := g.fauna.wait[render.Dread]; w < dayLength*b.every[0]-1 {
		t.Errorf("after one is killed the next may come in %.0f s", w)
	}
}

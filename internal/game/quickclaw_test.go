package game

import (
	"math"
	"math/rand/v2"
	"testing"

	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/sound"
	"github.com/kyrokohan/skyspear/internal/world"
)

// claw is a quickclaw on the test floor.
func claw(x, z float64) beast {
	return beast{
		kind: render.Quickclaw, herd: 9, health: breeds[render.Quickclaw].health,
		Pos:   geometry.Vector3{X: x, Y: 6, Z: z},
		timer: math.Inf(1), rng: rand.New(rand.NewPCG(11, uint64(x*100+z))),
	}
}

// atNight is a game in survival on the test floor at midnight, with no beasts and none
// to come: a test puts in the ones it wants.
func atNight(t *testing.T) *Game {
	t.Helper()
	g := inSurvival(t)
	g.days = midnight
	g.fauna.all, g.fauna.timer = nil, math.Inf(1)
	return g
}

func TestQuickclawsComeOnlyInTheDarkAndOnlyInSurvival(t *testing.T) {
	g := inSurvival(t)
	g.days = midnight
	g.fauna.all = nil
	hold(t, g, seconds(30))
	pack := only(g.fauna.all, render.Quickclaw)
	b := breeds[render.Quickclaw]
	if len(pack) < b.group[0] || len(pack) > b.most {
		t.Fatalf("after half a minute of night there are %d quickclaws, wanted a pack of %d to %d", len(pack), b.group[0], b.most)
	}

	for name, g := range map[string]*Game{"by day": inSurvival(t), "in creative": onFloor(t)} {
		if name == "in creative" {
			g.days = midnight
		}
		g.fauna.all = nil
		hold(t, g, seconds(30))
		if n := len(only(g.fauna.all, render.Quickclaw)); n != 0 {
			t.Errorf("%s: %d quickclaws came", name, n)
		}
	}
}

func TestAQuickclawRunsDownAWalkingPlayerAndBites(t *testing.T) {
	g := atNight(t)
	g.world = plain()
	g.player.Pos = geometry.Vector3{X: 0.5, Y: 6, Z: 0.5}
	g.fauna.all = []beast{claw(0.5, -3.5)} // four blocks behind the player, who walks away from it

	hush(g)
	hold(t, g, seconds(12), 'w')
	c := g.fauna.all[0]
	if d := c.Pos.Distance(g.player.Pos); d > 2.5 {
		t.Errorf("after twelve seconds the quickclaw is %.1f behind a walking player, wanted it to have caught up", d)
	}
	if lost := maxHealth - g.health; lost < breeds[render.Quickclaw].bite {
		t.Errorf("the player lost %v health to a quickclaw that caught it, wanted a bite of %v or more", lost, breeds[render.Quickclaw].bite)
	}
	if h := heard(g); h[sound.Screech] == 0 {
		t.Errorf("a quickclaw that saw its prey was not heard: %v", g.ears.Playing())
	}
	if breeds[render.Quickclaw].run <= walkSpeed {
		t.Error("a quickclaw is no faster than the player walks")
	}
}

func TestAQuickclawStopsAtTheEdgeOfATorchsLight(t *testing.T) {
	g := atNight(t)
	g.world.Set(20, 6, 20, world.Torch) // where the player stands
	g.fauna.all = []beast{claw(20.5, 32.5)}

	nearest := math.Inf(1)
	for range seconds(15) {
		hold(t, g, 1)
		nearest = min(nearest, g.fauna.all[0].Pos.Distance(g.player.Pos))
	}
	if nearest < 2.5 || nearest > 6 {
		t.Errorf("the quickclaw came to %.1f from a player by a torch, wanted it to wait at the edge of the light", nearest)
	}
	if g.health != maxHealth {
		t.Errorf("the player by a torch was bitten: health %v", g.health)
	}
	drawn(t, g)
	if len(g.poses) != 1 || !g.poses[0].Glow {
		t.Errorf("the eyes of a quickclaw in the dark do not shine: %+v", g.poses)
	}

	// With the torch gone it comes on.
	g.world.Set(20, 6, 20, world.Air)
	hold(t, g, seconds(6))
	if g.health == maxHealth {
		t.Error("with the torch gone the quickclaw did not come for the player")
	}
}

func TestAHiddenPlayerIsNotNoticed(t *testing.T) {
	b := breeds[render.Quickclaw]
	for _, tc := range []struct {
		name                           string
		far                            float64
		crouched, moving, hidden, meat bool
		noticed                        bool
	}{
		{"standing in sight", b.sees - 1, false, false, false, false, true},
		{"standing beyond sight", b.sees + 1, false, false, false, false, false},
		{"crouched at the same place", b.sees - 1, true, false, false, false, false},
		{"crouched, near", b.sees/2 - 1, true, false, false, false, true},
		{"crouched in tall ferns, near", 5, true, false, true, false, false},
		{"crouched in tall ferns, at its feet", 2, true, false, true, false, true},
		{"standing beyond sight, with meat", b.sees + 5, false, false, false, true, true},
	} {
		w := plain()
		f := &fauna{rng: rand.New(rand.NewPCG(1, 2)), hunted: true}
		f.sneaking, f.moving, f.hidden, f.meaty = tc.crouched, tc.moving, tc.hidden, tc.meat
		c := claw(0.5, 0.5)
		player := geometry.Vector3{X: 0.5, Y: 6, Z: 0.5 + tc.far}
		if got := f.notices(w, &c, player); got != tc.noticed {
			t.Errorf("%s, %.0f away: noticed %v, wanted %v", tc.name, tc.far, got, tc.noticed)
		}
	}
}

func TestAPlayerIsHeardWhenItMovesAndHiddenByAWall(t *testing.T) {
	b := breeds[render.Quickclaw]
	w := plain()
	for z := -40; z < 80; z++ { // a wall across the plain, higher than either of them
		for y := 6; y < 10; y++ {
			w.Set(z, y, 6, world.Stone)
		}
	}
	c := claw(0.5, 0.5)
	f := &fauna{rng: rand.New(rand.NewPCG(1, 2)), hunted: true}

	beyond := geometry.Vector3{X: 0.5, Y: 6, Z: 0.5 + b.hears - 2} // the other side of the wall
	if f.notices(w, &c, beyond) {
		t.Error("a player standing still behind a wall was noticed")
	}
	f.moving = true
	if !f.notices(w, &c, beyond) {
		t.Error("a player walking behind a wall, within hearing, was not heard")
	}
	f.sneaking = true
	if f.notices(w, &c, beyond) {
		t.Error("a player creeping behind a wall was heard")
	}
	close := claw(0.5, 4.5) // right against the wall, with the player creeping past its other side
	if !f.notices(w, &close, geometry.Vector3{X: 0.5, Y: 6, Z: 7.3}) {
		t.Error("a player creeping right behind the wall was not heard")
	}
}

func TestTheGameTellsTheBeastsWhatThePlayerDoes(t *testing.T) {
	g := atNight(t)
	hold(t, g, 5, 'w')
	if f := g.fauna; !f.moving || f.sneaking || f.hidden || f.meaty || !f.hunted {
		t.Errorf("a player walking in survival: %+v", *f)
	}
	g.world.Set(20, 6, 21, world.TallFern)
	g.player.Pos = geometry.Vector3{X: 20.5, Y: 6, Z: 21.5}
	holding(g, world.Meat, 1)
	hold(t, g, 1, 'c')
	hold(t, g, 2)
	if f := g.fauna; f.moving || !f.sneaking || !f.hidden || !f.meaty {
		t.Errorf("a player crouched in tall ferns with meat: moving %v, sneaking %v, hidden %v, meaty %v", f.moving, f.sneaking, f.hidden, f.meaty)
	}
	if text, _ := drawn(t, g); !contains(text, "hidden") {
		t.Errorf("the screen does not say the player is hidden:\n%s", text)
	}
	hold(t, g, 1, 'c') // stands up
	if text, _ := drawn(t, g); contains(text, "hidden") || g.fauna.hidden {
		t.Error("a player standing in tall ferns is hidden")
	}
}

func TestQuickclawsHuntTheHerdsWhenNoPlayerIsNoticed(t *testing.T) {
	w := plain()
	f := herdOf(5, 20, 20)
	f.hunted = true
	f.all = append(f.all, claw(20.5, 32.5))
	hunter := len(f.all) - 1
	far := geometry.Vector3{X: 20, Y: 6, Z: -40} // the player is far off, and not noticed

	health := 0.0
	for i := range f.all[:hunter] {
		health += f.all[i].health
	}
	alarmed := false
	for range 25 * 30 {
		f.step(w, far, frame)
		alarmed = alarmed || len(f.alarms) > 0
	}
	left := 0.0
	for i := range f.all {
		if f.all[i].kind == render.Bugler {
			left += f.all[i].health
		}
	}
	if left >= health {
		t.Errorf("after 25 seconds beside a herd the quickclaw has bitten none of it: health %v of %v", left, health)
	}
	if !alarmed {
		t.Error("the herd did not take alarm at the hunter among it")
	}
}

func TestAQuickclawThatMakesAKillStandsOverItToEat(t *testing.T) {
	w := plain()
	f := &fauna{rng: rand.New(rand.NewPCG(1, 2)), timer: math.Inf(1), hunted: true}
	prey := flockOf(1, 20.5, 22)[0]
	prey.health, prey.timer = 1, math.Inf(1)
	f.all = []beast{claw(20.5, 20.5), prey}

	for range 3 * 30 {
		f.step(w, geometry.Vector3{X: 20, Y: 6, Z: -40}, frame)
	}
	c := f.all[0]
	if c.kind != render.Quickclaw || c.feeding <= 0 || c.walking {
		t.Fatalf("after its kill the quickclaw is %+v, wanted it standing to eat", c)
	}
	at := c.Pos
	for range 3 * 30 {
		f.step(w, geometry.Vector3{X: 20, Y: 6, Z: -40}, frame)
	}
	if f.all[0].Pos.Distance(at) > 0.5 {
		t.Error("the quickclaw left its kill at once")
	}
}

func TestAHurtQuickclawAloneRunsAndOneOfAPackComesAgain(t *testing.T) {
	alone := atNight(t)
	alone.fauna.all = []beast{claw(20.5, 22.5)}
	alone.fauna.all[0].health = 3
	holding(alone, world.Stick, 1)
	alone.strike(prey{i: 0})
	if c := alone.fauna.all[0]; c.flee < 4 || c.health != 2 {
		t.Errorf("a quickclaw alone, hit down to %v health, runs for %v more seconds: wanted it to run", c.health, c.flee)
	}

	pack := atNight(t)
	pack.fauna.all = []beast{claw(20.5, 22.5), claw(22.5, 22.5)}
	pack.fauna.all[0].health = 3
	holding(pack, world.Stick, 1)
	pack.strike(prey{i: 0})
	if c := pack.fauna.all[0]; c.flee > 1 {
		t.Errorf("a hurt quickclaw with its pack beside it runs for %v seconds, wanted it to come again", c.flee)
	}
}

func TestQuickclawsLeaveWithTheDay(t *testing.T) {
	g := atNight(t)
	g.fauna.all = []beast{claw(20.5, 30.5), claw(22.5, 30.5)}
	g.days = 0 // morning
	hold(t, g, seconds(25))
	if n := len(only(g.fauna.all, render.Quickclaw)); n != 0 {
		t.Errorf("%d quickclaws are still about after 25 seconds of day", n)
	}
	if g.health != maxHealth {
		t.Errorf("a quickclaw bit the player by day: health %v", g.health)
	}
}

func TestQuickclawsAreNotSavedAndLoseAPlayerWhoFaintsOrSleeps(t *testing.T) {
	g := atNight(t)
	g.fauna.all = []beast{claw(20.5, 24.5), {kind: render.Mossback, Pos: geometry.Vector3{X: 25.5, Y: 6, Z: 20.5}}}
	if saved := g.fauna.saved(); len(saved) != 1 || saved[0].Kind != "mossback" {
		t.Errorf("the save holds %+v, wanted the mossback alone", saved)
	}

	g.world.Set(21, 6, 20, world.FurBed)
	g.lieDown(world.Cell{X: 21, Y: 6, Z: 20})
	if text, _ := drawn(t, g); !contains(text, "cannot sleep with a hunter near") || g.days != midnight {
		t.Errorf("with a quickclaw four blocks off the player slept, or was not told why not:\n%s", text)
	}
	g.fauna.all[0].Pos.Z = 60
	g.lieDown(world.Cell{X: 21, Y: 6, Z: 20})
	if len(only(g.fauna.all, render.Quickclaw)) != 0 || len(g.fauna.all) != 1 {
		t.Errorf("after a night's sleep the beasts are %+v, wanted the hunters gone and the mossback there", g.fauna.all)
	}

	g = atNight(t)
	g.fauna.all = []beast{claw(20.5, 24.5)}
	g.wake()
	if len(g.fauna.all) != 0 {
		t.Error("the hunters did not lose a player who fainted and woke elsewhere")
	}
}

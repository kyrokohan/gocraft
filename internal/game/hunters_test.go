package game

import (
	"math"
	"testing"

	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/sound"
	"github.com/kyrokohan/skyspear/internal/world"
)

// steppeGame is a game in survival in the world of seed 7, with its player on the
// steppe, holding torches, no beasts, and the next of them looked for at once.
func steppeGame(t *testing.T, days float64) *Game {
	t.Helper()
	g, err := New(Options{Seed: 7})
	if err != nil {
		t.Fatal(err)
	}
	g.Play()
	g.begin(true)
	_, at := onSteppe(t, 7)
	g.world.Follow(at.X, at.Z)
	g.world.LoadAll()
	at.Y = float64(g.world.Ground(int(math.Floor(at.X)), int(math.Floor(at.Z)))) + 1
	stand(g, at)
	g.items[0] = world.Stack{Item: world.Torch.Item(), Count: 5} // which keeps what comes off
	g.days = days
	g.fallen = g.days
	g.fauna.all, g.fauna.timer = nil, 0
	hush(g)
	return g
}

func TestHowlersAreHeardBeforeTheyComeToThePlayerOnTheSteppeByNight(t *testing.T) {
	g := steppeGame(t, midnight)
	howlAt, cameAt, most := -1.0, -1.0, 0
	for i := range seconds(40) {
		hold(t, g, 1)
		now := float64(i) * frame
		most = max(most, g.fauna.count(render.Howler))
		if howlAt < 0 && heard(g)[sound.Howl] > 0 {
			howlAt = now
		}
		if cameAt < 0 && g.fauna.count(render.Howler) > 0 {
			cameAt = now
		}
	}
	b := breeds[render.Howler]
	if howlAt < 0 || cameAt < 0 || cameAt < howlAt+b.heralds-1.5 {
		t.Fatalf("the howl was heard after %.1f s and the pack came after %.1f s: wanted the howl first, by %v seconds", howlAt, cameAt, b.heralds)
	}
	if most < 4 || most > 6 {
		t.Errorf("a pack of %d howlers came, wanted four to six", most)
	}
	// With a torch in hand they keep off, as quickclaws do.
	nearest := math.Inf(1)
	for range seconds(20) {
		hold(t, g, 1)
		for _, c := range only(g.fauna.all, render.Howler) {
			nearest = min(nearest, c.Pos.Distance(g.player.Pos))
		}
	}
	if nearest < 2.5 || g.health != maxHealth {
		t.Errorf("the pack came to %.1f blocks from a player with a torch, who has lost %v points", nearest, maxHealth-g.health)
	}

	// Not by day, not on the first night, and not off the steppe.
	for name, quiet := range map[string]*Game{
		"at noon on the steppe":            steppeGame(t, midnight+0.5),
		"on the first night on the steppe": steppeGame(t, firstMidnight),
	} {
		hold(t, quiet, seconds(30))
		if n := quiet.fauna.count(render.Howler); n != 0 || heard(quiet)[sound.Howl] != 0 {
			t.Errorf("%s %d howlers came, and %d howls were heard", name, n, heard(quiet)[sound.Howl])
		}
	}
	home := steppeGame(t, midnight)
	spawn := home.world.Spawn()
	if land, _ := home.world.Land(spawn.X, spawn.Z); land == world.Tundra {
		t.Skip("the world of seed 7 begins on the steppe")
	}
	home.world.Follow(spawn.X, spawn.Z)
	home.world.LoadAll()
	stand(home, home.world.Spawn())
	hold(t, home, seconds(30))
	if n := home.fauna.count(render.Howler); n != 0 || heard(home)[sound.Howl] != 0 {
		t.Errorf("off the steppe %d howlers came at midnight, and %d howls were heard", n, heard(home)[sound.Howl])
	}
}

func TestADaggercatStalksSpringsFromEightBlocksAndBacksOff(t *testing.T) {
	g := hunting(t, 0.5, giant(render.Daggercat, 0.5, 18.5)) // at noon, 18 blocks off
	b := breeds[render.Daggercat]
	c := &g.fauna.all[0]
	sprang, from := false, 0.0
	for range seconds(20) {
		hold(t, g, 1)
		if c.leap > 0 && !sprang {
			sprang, from = true, c.Pos.Distance(g.player.Pos)
		}
		if g.health < maxHealth {
			break
		}
	}
	if !sprang || from > b.springs+0.5 || from < b.springs-2.5 || heard(g)[sound.Snarl] == 0 {
		t.Fatalf("a daggercat sprang: %v, from %.1f blocks, with %d snarls: wanted a spring from about %v", sprang, from, heard(g)[sound.Snarl], b.springs)
	}
	if lost := maxHealth - g.health; lost != b.bite || lost != 10 {
		t.Fatalf("the spring of a daggercat took %v points, wanted 10", lost)
	}
	hold(t, g, seconds(2))
	if far := c.Pos.Distance(g.player.Pos); c.flee <= 0 || far < 6 {
		t.Errorf("two seconds after it bit, the daggercat is %.1f blocks off and runs for %.1f s more: wanted it to back off", far, c.flee)
	}
	// It comes again.
	g.health = maxHealth
	hold(t, g, seconds(15))
	if g.health == maxHealth {
		t.Error("fifteen seconds on, the daggercat has not sprung again")
	}

	// A player who steps well aside as it springs is missed.
	g = hunting(t, 0.5, giant(render.Daggercat, 0.5, 14.5))
	c = &g.fauna.all[0]
	for range seconds(20) {
		if hold(t, g, 1); c.leap > 0 {
			g.player.Pos.X += 3.5
			break
		}
	}
	hold(t, g, seconds(springTime+0.3))
	if g.health != maxHealth || c.leap != 0 {
		t.Errorf("a player who stepped aside lost %v points, and the daggercat is still in the air for %.2f s", maxHealth-g.health, c.leap)
	}
}

func TestADaggercatLiesStillInTheDarkAndWillNotComeAtAFire(t *testing.T) {
	g := hunting(t, 0.5, giant(render.Daggercat, 0.5, 12.5))
	g.days = midnight
	hold(t, g, seconds(15))
	if c := g.fauna.all[0]; c.keen > 0 || c.Pos.Distance(geometry.Vector3{X: 0.5, Y: 6, Z: 12.5}) > 1 || g.health != maxHealth {
		t.Errorf("at midnight a daggercat twelve blocks off is keen for %.1f s, has moved %.1f, and the player has lost %v points", c.keen, c.Pos.Distance(geometry.Vector3{X: 0.5, Y: 6, Z: 12.5}), maxHealth-g.health)
	}

	g = hunting(t, 0.5, giant(render.Daggercat, 0.5, 20.5))
	g.world.Set(2, 6, 0, world.Fire) // two blocks from the player
	nearest, sprang := math.Inf(1), false
	for range seconds(25) {
		hold(t, g, 1)
		c := &g.fauna.all[0]
		nearest, sprang = min(nearest, c.Pos.Distance(g.player.Pos)), sprang || c.leap > 0
	}
	if b := breeds[render.Daggercat]; sprang || nearest < b.springs || g.health != maxHealth {
		t.Errorf("a daggercat came to %.1f blocks from a player by a fire, sprang: %v, and took %v points: wanted it to keep out of its spring", nearest, sprang, maxHealth-g.health)
	}
}

func TestTheHuntersOfTheSteppeAreWhatTheirRowsSay(t *testing.T) {
	howler, cat, claw := breeds[render.Howler], breeds[render.Daggercat], breeds[render.Quickclaw]
	if howler.group != [2]int{4, 6} || !howler.visits || howler.bears != claw.bears || howler.from != firstHunt || howler.heralds < 5 {
		t.Errorf("a howler: packs of %v, visits %v, bears light of %v, from day %v, heard %v s before", howler.group, howler.visits, howler.bears, howler.from, howler.heralds)
	}
	if cat.visits || !cat.byDay || cat.springs != 8 || cat.bite != 10 || cat.group != [2]int{1, 1} || cat.most != 1 {
		t.Errorf("a daggercat: visits %v, by day %v, springs from %v, bites for %v, groups of %v, at the most %d", cat.visits, cat.byDay, cat.springs, cat.bite, cat.group, cat.most)
	}
	// A daggercat lives on the steppe, one at the most, and howlers are not there until
	// they come.
	found := 0
	for seed := int64(1); seed <= 5; seed++ {
		w, at := onSteppe(t, seed)
		f := newFauna(w, at)
		if n := f.count(render.Daggercat); n > 1 || f.count(render.Howler) != 0 {
			t.Errorf("seed %d: %d daggercats and %d howlers stand on the steppe as a world is made", seed, n, f.count(render.Howler))
		}
		for _, c := range only(f.all, render.Daggercat) {
			found++
			if land := world.BiomeAt(seed, int(math.Floor(c.Pos.X)), int(math.Floor(c.Pos.Z))); land != world.Tundra {
				t.Errorf("seed %d: a daggercat stands in land %d", seed, land)
			}
		}
	}
	if found < 2 {
		t.Errorf("%d daggercats were found on the steppe of five worlds", found)
	}
}

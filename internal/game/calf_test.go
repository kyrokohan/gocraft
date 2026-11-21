package game

import (
	"math"
	"testing"

	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/sound"
	"github.com/kyrokohan/skyspear/internal/world"
)

// family is a game on the plain with a family of snowtusks: two grown ones, and their
// calf at 0.5, 33.5. The player stands at the z given, facing it, with berries in hand.
func family(t *testing.T, z float64) *Game {
	t.Helper()
	bull, cow, calf := giant(render.Snowtusk, -3.5, 30.5), giant(render.Snowtusk, 4.5, 30.5), giant(render.Snowtusk, 0.5, 33.5)
	bull.herd, cow.herd, calf.herd, calf.young = 1, 1, 1, true
	g := among(t, z, bull, cow, calf)
	holding(g, world.Berries, 10)
	return g
}

// calm reports whether no grown snowtusk of a family warns or charges, and the player is
// whole.
func calm(g *Game) bool {
	for _, c := range g.fauna.all[:2] {
		if c.lower || c.ire > 0 {
			return false
		}
	}
	return heard(g)[sound.Trumpet] == 0 && g.health == maxHealth
}

func TestACrouchedPlayerWithFoodReachesASnowtuskCalf(t *testing.T) {
	g := family(t, 31)
	g.player.Crouched = true
	for range seconds(4) {
		if hold(t, g, 1); !calm(g) {
			t.Fatalf("a snowtusk turned on a crouched player with berries in hand 2.5 blocks from its calf")
		}
	}
	// Stood up, the player is a threat again.
	g.player.Crouched = false
	hold(t, g, seconds(3))
	if calm(g) {
		t.Error("a player who stood up 2.5 blocks from a snowtusk calf was not warned or charged")
	}
	// And so is one with nothing the calf eats in hand, crouched or not.
	g = family(t, 31)
	g.player.Crouched = true
	holding(g, world.Meat, 1)
	hold(t, g, seconds(3))
	if calm(g) {
		t.Error("a crouched player with meat in hand 2.5 blocks from a snowtusk calf was not warned or charged")
	}
}

func TestFiveMealsWinASnowtuskCalfAndNoMoreThanThreeADay(t *testing.T) {
	g := family(t, 31.5)
	g.player.Crouched = true
	calf := &g.fauna.all[2]
	b := breeds[render.Snowtusk]
	if b.meals != 5 || b.daily != 3 {
		t.Fatalf("a snowtusk calf is won by %d meals, %d a day", b.meals, b.daily)
	}
	feed := func() {
		hold(t, g, 1, 'e')
		hold(t, g, seconds(1))
	}
	for range 4 {
		feed()
	}
	if calf.fed != 3 || calf.tame || g.items[g.selected].Count != 7 {
		t.Fatalf("four meals held out in a day: the calf ate %d, is tame: %v; %d berries are left", calf.fed, calf.tame, g.items[g.selected].Count)
	}
	// The next day it takes two more, and is the player's: it leaves its family.
	g.days++
	feed()
	if calf.tame {
		t.Fatal("a snowtusk calf was won by four meals")
	}
	feed()
	if !calf.tame || !calf.young || calf.herd != 0 || calf.fed != 5 {
		t.Fatalf("five meals over two days: the calf is tame %v, young %v, of herd %d, fed %d", calf.tame, calf.young, calf.herd, calf.fed)
	}
	if !calm(g) {
		t.Error("a family turned on the player who won its calf")
	}
	g.player.Crouched = false
	g.player.Pos.Z -= 20
	hold(t, g, seconds(3))
	if !calf.walking || calf.Pos.Z > 33.4 {
		t.Errorf("a calf that is the player's did not follow it: walking %v, at z %.1f", calf.walking, calf.Pos.Z)
	}
	// A grown one eats nothing from the hand.
	g = family(t, 26.5)
	g.player.Pos.X, g.player.Crouched = -3.5, true // before the bull
	hold(t, g, 1, 'e')
	if g.fauna.all[0].fed+g.fauna.all[1].fed != 0 {
		t.Error("a grown snowtusk ate from the player's hand")
	}
}

func TestOnASnowtusksBackTheRiderKeepsWarmInTheWinter(t *testing.T) {
	for _, rides := range []bool{true, false} {
		g := winterAt(t, 0.5)
		c := raised(render.Snowtusk, g.player.Pos.X, g.player.Pos.Z, false)
		c.Pos, c.ridden = g.player.Pos, rides
		g.fauna.all = append(g.fauna.all, c)
		hold(t, g, seconds(30))
		if kept := g.warmth == maxWarmth; kept != rides {
			t.Errorf("thirty seconds of the Long Winter on a snowtusk's back (%v) left %.1f of %v warmth", rides, g.warmth, maxWarmth)
		}
	}
}

func TestARiddenSnowtuskIsSlowGoesThroughLeavesAndTramplesHunters(t *testing.T) {
	g, m := mountedOn(t, render.Snowtusk, -60.5)
	hold(t, g, seconds(2), 'w')
	from := m.Pos.Z
	hold(t, g, seconds(1), 'w')
	if went := m.Pos.Z - from; math.Abs(went-2.5) > 0.2 || !onBack(g, m) {
		t.Errorf("a snowtusk ridden with W held for three seconds went %.1f blocks in the last of them: wanted 2.5", went)
	}
	// Leaves do not stop it.
	for x := -6; x <= 6; x++ {
		for y := 6; y < 14; y++ {
			g.world.Set(x, y, int(m.Pos.Z)+4, world.Leaves)
		}
	}
	hold(t, g, seconds(4), 'w')
	if m.Pos.Z < from+9 {
		t.Errorf("a ridden snowtusk was stopped by a wall of leaves at z %.1f", m.Pos.Z)
	}

	// A hunter in its way is trampled and thrown aside; the dread is not.
	g, m = mountedOn(t, render.Snowtusk, 10.5)
	g.days = midnight
	g.fauna.all = append(g.fauna.all, claw(0.5, 16.5), dread(0.5, 40.5))
	hold(t, g, seconds(4), 'w')
	q, d := only(g.fauna.all, render.Quickclaw), only(g.fauna.all, render.Dread)
	if len(q) == 1 && q[0].health == breeds[render.Quickclaw].health {
		t.Error("a quickclaw in the way of a ridden snowtusk was not trampled")
	}
	if len(d) != 1 || d[0].health != breeds[render.Dread].health {
		t.Error("a ridden snowtusk trampled the dread")
	}
}

func TestASaveHoldsTheMealsOfADay(t *testing.T) {
	c := giant(render.Snowtusk, 0.5, 3.5)
	c.young, c.herd, c.fed, c.mealDay, c.mealsToday = true, 1, 4, 2, 2
	f := &fauna{all: []beast{c}}
	back := faunaFrom(world.New(), world.Meta{Beasts: f.saved()}, geometry.Vector3{})
	if got := back.all[0]; got.fed != 4 || got.mealDay != 2 || got.mealsToday != 2 {
		t.Errorf("a calf fed 4 times, twice on day 2, came back fed %d, %d times on day %d", got.fed, got.mealsToday, got.mealDay)
	}
}

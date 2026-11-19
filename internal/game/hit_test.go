package game

import (
	"math"
	"math/rand/v2"
	"path/filepath"
	"testing"

	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/world"
)

// midnight is the days of play to the middle of the second night: the first is quiet,
// and no hunter comes in it.
const midnight = 1.7

// withMossback puts a mossback on the test floor three blocks ahead of the player, and
// turns the player to look at it.
func withMossback(t *testing.T, g *Game) *beast {
	t.Helper()
	g.fauna.all = []beast{{Pos: geometry.Vector3{X: 20.5, Y: 6, Z: 23.5}, timer: math.Inf(1), rng: rand.New(rand.NewPCG(1, 2))}}
	g.player.Pitch = -0.35
	hold(t, g, 1)
	if !g.preyed || g.prey != (prey{i: 0}) {
		t.Fatalf("aimed at %+v (%v), wanted the mossback", g.prey, g.preyed)
	}
	return &g.fauna.all[0]
}

func TestAHitKnocksAMossbackAwayAndItRuns(t *testing.T) {
	g := onFloor(t)
	c := withMossback(t, g)

	hold(t, g, 1, 'q')
	if c.flee <= 0 || c.hurt <= 0 || !c.Pose().Hurt {
		t.Fatalf("after a hit the mossback runs for %v s and shows it for %v s", c.flee, c.hurt)
	}
	if g.world.At(20, 5, 24) != world.Stone {
		t.Error("the floor behind the mossback was broken")
	}
	hold(t, g, 30)
	if c.Pos.Z < 26 {
		t.Errorf("a second after the hit the mossback is at %v, wanted it well away", c.Pos)
	}
}

func TestAMossbackCannotBeKilled(t *testing.T) {
	g := inSurvival(t)
	g.items[0] = world.Stack{Item: world.IronBlade, Count: 1}
	c := withMossback(t, g)
	for range 20 {
		c.Pos = geometry.Vector3{X: 20.5, Y: 6, Z: 23.5}
		hold(t, g, seconds(hitWait)+1, 'q')
	}
	if len(g.fauna.all) != 1 {
		t.Errorf("after 20 hits there are %d mossbacks", len(g.fauna.all))
	}
}

func TestShearingAMossbackGivesMossUntilItGrowsBack(t *testing.T) {
	g := inSurvival(t)
	c := withMossback(t, g)

	hold(t, g, 1, 'e')
	if len(g.drops) != 1 || g.drops[0].Stack != (world.Stack{Item: world.Moss, Count: mossShorn}) || !c.Pose().Shorn {
		t.Fatalf("after shearing, %+v lies about and the mossback is shorn %v", g.drops, c.Pose().Shorn)
	}
	hold(t, g, 1)
	hold(t, g, 1, 'e')
	if len(g.drops) != 1 {
		t.Errorf("a shorn mossback gave more moss: %+v", g.drops)
	}

	c.regrow = frame / 2
	hold(t, g, 1)
	hold(t, g, 1, 'e')
	if len(g.drops) != 2 {
		t.Errorf("its moss grew back and it gave none: %+v", g.drops)
	}
}

func TestTheHerdIsSavedAndComesBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "world")
	g := savedGame(t, path)
	startNew(t, g, true)
	g.fauna.all = []beast{
		{Pos: geometry.Vector3{X: 3.5, Y: 70, Z: 4.5}, Yaw: 1, regrow: 100, rng: rand.New(rand.NewPCG(1, 2))},
		{Pos: geometry.Vector3{X: -6.5, Y: 71, Z: 2.5}, Yaw: -2, rng: rand.New(rand.NewPCG(1, 3))},
	}
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}

	back := savedGame(t, path)
	if len(back.fauna.all) != 2 {
		t.Fatalf("came back with %d mossbacks, wanted 2", len(back.fauna.all))
	}
	for i, c := range back.fauna.all {
		if want := g.fauna.all[i]; c.Pos != want.Pos || c.Yaw != want.Yaw || c.regrow != want.regrow || c.rng == nil {
			t.Errorf("mossback %d came back as %+v, wanted %+v", i, c, want)
		}
	}
}

func TestAHeldKeyHitsNoFasterThanHitWait(t *testing.T) {
	g := inSurvival(t)
	g.days = midnight // so that the quickclaw does not leave
	g.fauna.timer = math.Inf(1)
	g.fauna.all = []beast{claw(20.5, 22.5)}
	g.fauna.all[0].health = 100
	hold(t, g, seconds(1), 'q')
	if hits := int(100 - g.fauna.all[0].health); hits < 2 || hits > 4 {
		t.Errorf("a second of holding Q hit %d times", hits)
	}
}

// A blade is the weapon: a hit with one of iron takes six points.
func TestAnIronBladeKillsAQuickclawInTwoHits(t *testing.T) {
	g := inSurvival(t)
	g.days = midnight
	g.fauna.timer = math.Inf(1)
	g.fauna.all = []beast{claw(20.5, 22.5)}
	g.items[0] = world.Stack{Item: world.IronBlade, Count: 1}

	g.strike(prey{i: 0})
	if c := g.fauna.all[0]; c.health != breeds[render.Quickclaw].health-world.IronBlade.Damage() || g.items[0].Wear != 1 {
		t.Fatalf("after a hit the quickclaw has %v health and the blade %d wear", c.health, g.items[0].Wear)
	}
	g.strike(prey{i: 0})
	if !g.fauna.all[0].dead {
		t.Fatalf("after two hits the quickclaw has %v health and lives", g.fauna.all[0].health)
	}
}

func TestAKnifeIsMadeAtAKnappingStone(t *testing.T) {
	g := inSurvival(t)
	g.world.Set(22, 6, 22, world.KnappingStone)
	g.items[0] = world.Stack{Item: world.Flint, Count: 1}
	g.items[1] = world.Stack{Item: world.Stick, Count: 1}
	g.items[2] = world.Stack{Item: world.Cord, Count: 1}

	if r, ok := recipeFor(g, world.FlintKnife); !ok || !g.craft(r) {
		t.Fatal("a flint knife could not be made")
	}
	if g.items[0] != (world.Stack{Item: world.FlintKnife, Count: 1}) {
		t.Errorf("made %+v", g.items[:3])
	}
}

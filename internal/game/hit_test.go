package game

import (
	"math"
	"math/rand/v2"
	"path/filepath"
	"testing"

	"glonk.dev/engine"
	"glonk.dev/geometry"

	"github.com/kyrokohan/gocraft/internal/world"
)

// midnight is the days of play to the middle of the first night.
const midnight = 0.7

// withMossback puts a mossback on the test floor three blocks ahead of the player, and
// turns the player to look at it.
func withMossback(t *testing.T, g *Game) *Creature {
	t.Helper()
	g.herd.all = []Creature{{Pos: geometry.Vector3{X: 20.5, Y: 6, Z: 23.5}, timer: math.Inf(1), rng: rand.New(rand.NewPCG(1, 2))}}
	g.player.Pitch = -0.35
	hold(t, g, 1)
	if !g.preyed || g.prey != (prey{i: 0}) {
		t.Fatalf("aimed at %+v (%v), wanted the mossback", g.prey, g.preyed)
	}
	return &g.herd.all[0]
}

// withGloam puts a gloam on the test floor three blocks ahead of the player at midnight,
// with no others to come, and turns the player to look at it.
func withGloam(t *testing.T, g *Game) *gloam {
	t.Helper()
	g.days, g.gloamWait = midnight, math.Inf(1)
	g.gloams = []gloam{{pos: geometry.Vector3{X: 20.5, Y: 6, Z: 23.5}, health: gloamHealth}}
	g.player.Pitch = -0.21
	hold(t, g, 1)
	if !g.preyed || g.prey != (prey{gloam: true}) {
		t.Fatalf("aimed at %+v (%v), wanted the gloam", g.prey, g.preyed)
	}
	return &g.gloams[0]
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
	if len(g.herd.all) != 1 {
		t.Errorf("after 20 hits there are %d mossbacks", len(g.herd.all))
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
	g.herd.all = []Creature{
		{Pos: geometry.Vector3{X: 3.5, Y: 70, Z: 4.5}, Yaw: 1, regrow: 100, rng: rand.New(rand.NewPCG(1, 2))},
		{Pos: geometry.Vector3{X: -6.5, Y: 71, Z: 2.5}, Yaw: -2, rng: rand.New(rand.NewPCG(1, 3))},
	}
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}

	back := savedGame(t, path)
	if len(back.herd.all) != 2 {
		t.Fatalf("came back with %d mossbacks, wanted 2", len(back.herd.all))
	}
	for i, c := range back.herd.all {
		if want := g.herd.all[i]; c.Pos != want.Pos || c.Yaw != want.Yaw || c.regrow != want.regrow || c.rng == nil {
			t.Errorf("mossback %d came back as %+v, wanted %+v", i, c, want)
		}
	}
}

func TestGloamsGatherInTheDarkInSurvivalOnly(t *testing.T) {
	g := inSurvival(t)
	g.days = midnight
	for range seconds(20 * gloamLook) {
		if hold(t, g, 1); len(g.gloams) > 0 {
			break
		}
	}
	if len(g.gloams) != 1 {
		t.Fatalf("after a long night there are %d gloams", len(g.gloams))
	}
	m := g.gloams[0]
	along := geometry.Vector3{X: m.pos.X - g.player.Pos.X, Z: m.pos.Z - g.player.Pos.Z}.Length()
	if along < gloamNear-1 || along > gloamFar+1 || m.pos.Y != 6 || m.fade < 0.9 {
		t.Errorf("a gloam gathered %v blocks away at %v, faded %v", along, m.pos, m.fade)
	}

	for _, tc := range []struct {
		name     string
		g        *Game
		survival bool
		days     float64
	}{
		{"in creative at night", onFloor(t), false, midnight},
		{"in survival by day", inSurvival(t), true, 0},
	} {
		tc.g.days = tc.days
		hold(t, tc.g, seconds(10*gloamLook))
		if len(tc.g.gloams) != 0 {
			t.Errorf("%s: %d gloams gathered", tc.name, len(tc.g.gloams))
		}
	}
}

func TestAGloamFollowsAndHurtsOnTouch(t *testing.T) {
	g := inSurvival(t)
	withGloam(t, g)
	hold(t, g, seconds(3))
	if g.health >= maxHealth {
		t.Fatalf("the gloam is at %v and the player at %v is unhurt", g.gloams[0].pos, g.player.Pos)
	}

	g.hurt(maxHealth)
	press(g, engine.KeyEnter) // Wake up
	if len(g.gloams) != 0 {
		t.Errorf("the player woke with %d gloams about", len(g.gloams))
	}
}

func TestAGloamFadesInTorchlight(t *testing.T) {
	g := inSurvival(t)
	withGloam(t, g)
	g.world.Set(21, 6, 23, world.Torch)

	hold(t, g, seconds(gloamFadeTime/2))
	if len(g.gloams) != 1 || g.gloams[0].fade == 0 {
		t.Fatalf("by the torch the gloams are %+v", g.gloams)
	}
	hold(t, g, seconds(gloamFadeTime))
	if len(g.gloams) != 0 {
		t.Errorf("a gloam is still there by a torch: %+v", g.gloams)
	}
}

func TestAnIronBladeUndoesAGloamInTwoHits(t *testing.T) {
	g := inSurvival(t)
	g.items[0] = world.Stack{Item: world.IronBlade, Count: 1}
	m := withGloam(t, g)

	hold(t, g, 1, 'q')
	if m.health != gloamHealth-world.IronBlade.Damage() || g.items[0].Wear != 1 {
		t.Fatalf("after a hit the gloam has %v health and the blade %d wear", m.health, g.items[0].Wear)
	}
	m.pos = geometry.Vector3{X: 20.5, Y: 6, Z: 23.5}
	hold(t, g, seconds(hitWait)+1)
	hold(t, g, 1, 'q')
	if !m.dying {
		t.Fatalf("after two hits the gloam has %v health", m.health)
	}
	hold(t, g, seconds(gloamDieTime)+1)
	if len(g.gloams) != 0 {
		t.Errorf("the gloam is still there: %+v", g.gloams)
	}
}

func TestAHeldKeyHitsNoFasterThanHitWait(t *testing.T) {
	g := inSurvival(t)
	m := withGloam(t, g)
	m.health = 100
	hold(t, g, seconds(1), 'q')
	if hits := int(100 - m.health); hits < 2 || hits > 4 {
		t.Errorf("a second of holding Q hit %d times", hits)
	}
}

func TestABladeIsMadeAndMossCooksToACake(t *testing.T) {
	g := inSurvival(t)
	g.world.Set(22, 6, 22, world.Workbench)
	g.items[0] = world.Stack{Item: world.Cobble.Item(), Count: 2}
	g.items[1] = world.Stack{Item: world.Stick, Count: 1}

	if r, ok := recipeFor(g, world.StoneBlade); !ok || !g.craft(r) {
		t.Fatal("a stone blade could not be made")
	}
	if g.items[0] != (world.Stack{Item: world.StoneBlade, Count: 1}) {
		t.Errorf("made %+v", g.items[:2])
	}

	k := &world.KilnStore{In: world.Stack{Item: world.Moss, Count: 3}, Fuel: world.Stack{Item: world.Coal, Count: 1}}
	stepKiln(k, cookTime+0.5)
	if k.Out != (world.Stack{Item: world.MossCake, Count: 1}) || k.In.Count != 0 {
		t.Errorf("the kiln holds %+v and made %+v, wanted the moss cooked to a cake", k.In, k.Out)
	}
}

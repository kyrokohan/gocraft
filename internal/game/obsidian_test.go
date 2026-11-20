package game

import (
	"testing"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/world"
)

func TestToolsOfObsidianAreKnappedAndAStrikerIsMadeByHand(t *testing.T) {
	g := inSurvival(t)
	g.items[0] = world.Stack{Item: world.Obsidian, Count: 5}
	g.items[1] = world.Stack{Item: world.Stick, Count: 6}
	g.items[2] = world.Stack{Item: world.Cord, Count: 3}
	g.items[3] = world.Stack{Item: world.Flint, Count: 1}
	g.items[4] = world.Stack{Item: world.Firestone, Count: 1}

	for _, item := range []world.Item{world.ObsidianKnife, world.ObsidianPick, world.ObsidianSpear} {
		if _, ok := recipeFor(g, item); ok {
			t.Errorf("%s can be made away from a knapping stone", item.Name())
		}
	}
	if r, ok := recipeFor(g, world.Striker); !ok || r.at != byHand || !g.craft(r) {
		t.Fatal("a striker could not be made by hand of a flint and a firestone")
	}

	g.world.Set(22, 6, 22, world.KnappingStone)
	for _, item := range []world.Item{world.ObsidianKnife, world.ObsidianPick, world.ObsidianSpear} {
		r, ok := recipeFor(g, item)
		if !ok || r.at != atBench || !g.craft(r) {
			t.Fatalf("%s could not be knapped of 5 obsidian, 6 sticks and 3 cords", item.Name())
		}
	}
	for item, want := range map[world.Item]int{
		world.ObsidianKnife: 1, world.ObsidianPick: 1, world.ObsidianSpear: 1, world.Striker: 1,
		world.Obsidian: 0, world.Stick: 0, world.Cord: 0, world.Flint: 0, world.Firestone: 0,
	} {
		if n := g.items.count([]world.Item{item}); n != want {
			t.Errorf("the player carries %d of %s, wanted %d", n, item.Name(), want)
		}
	}
}

func TestAStrikerLightsAHearthAtOnceAndNotOneWithNoFuel(t *testing.T) {
	g, at := atHearth(t, world.Striker, world.Stack{Item: world.Stick, Count: 2})
	hold(t, g, 1, 'e')
	hold(t, g, 2)
	if k := g.hearth(at); k.Burn == 0 || k.Fuel.Count != 1 || g.world.At(at.X, at.Y, at.Z) != world.HearthLit || g.screen != playing {
		t.Fatalf("one press of E with a striker: fire %v, fuel %+v, block %v, screen %d", k.Burn, k.Fuel, g.world.At(at.X, at.Y, at.Z), g.screen)
	}
	if g.items[0].Wear != 1 {
		t.Errorf("the striker has %d uses on it, wanted 1", g.items[0].Wear)
	}

	g, at = atHearth(t, world.Striker, world.Stack{})
	hold(t, g, 1, 'e')
	if g.hearth(at).Burn != 0 || g.screen != inBag || g.items[0].Wear != 0 {
		t.Errorf("a striker on a hearth with no fuel: fire %v, screen %d, wear %d: wanted it opened to be fed", g.hearth(at).Burn, g.screen, g.items[0].Wear)
	}
}

func TestAStrikerLightsTheTorchesCarriedWithNoFireNear(t *testing.T) {
	g := inSurvival(t)
	g.items[0] = world.Stack{Item: world.Striker, Count: 1}
	g.items[5] = world.Stack{Item: world.UnlitTorch, Count: 3}
	g.items[9] = world.Stack{Item: world.UnlitTorch, Count: 2}
	g.player.Pitch = 0.6 // at the sky: nothing is aimed at
	hold(t, g, 1, 'e')
	if g.items[5] != (world.Stack{Item: world.Torch.Item(), Count: 3}) || g.items[9].Item != world.UnlitTorch {
		t.Errorf("E with a striker: the torches are %+v and %+v, wanted the first of them lit", g.items[5], g.items[9])
	}
	if g.items[0].Wear != 1 || g.notice == "" {
		t.Errorf("the striker has %d uses on it, and the player was told %q", g.items[0].Wear, g.notice)
	}

	// With none to light it does nothing, and is not worn.
	g = inSurvival(t)
	g.items[0] = world.Stack{Item: world.Striker, Count: 1}
	g.player.Pitch = 0.6
	hold(t, g, 2, 'e')
	if g.items[0] != (world.Stack{Item: world.Striker, Count: 1}) {
		t.Errorf("a striker with no torch to light is now %+v", g.items[0])
	}

	// A last use, and it is gone.
	g = inSurvival(t)
	g.items[0] = world.Stack{Item: world.Striker, Count: 1, Wear: world.Striker.Uses() - 1}
	g.items[1] = world.Stack{Item: world.UnlitTorch, Count: 1}
	g.player.Pitch = 0.6
	hold(t, g, 1, 'e')
	if g.items[0].Count != 0 || g.items[1].Item != world.Torch.Item() {
		t.Errorf("a striker on its last fire: %+v left, and the torch is %+v", g.items[0], g.items[1])
	}
}

func TestAnObsidianSpearIsThrownAndAnObsidianKnifeCarves(t *testing.T) {
	g := thrower(t, world.ObsidianSpear, 0)
	one := quarry(7)
	one.young, one.Pos.X, one.Pos.Z = false, 0.5, 7.5
	g.fauna.all = []beast{one}
	hold(t, g, 1, 'r')
	if len(g.missiles) != 1 || g.items[0].Count != 0 {
		t.Fatalf("R with an obsidian spear: %d in the air, %+v in hand", len(g.missiles), g.items[0])
	}
	flies(t, g)
	if lost := 100 - g.fauna.all[0].health; lost != 7 {
		t.Errorf("a thrown obsidian spear took %v points, wanted 7", lost)
	}

	// A kill with an obsidian knife is carved, as with a flint one, in fewer hits.
	hits := func(with world.Item) (n int, carved bool) {
		g := among(t, -30.5, giant(render.Shieldbrow, 0.5, 20.5))
		holding(g, with, 1)
		for ; !g.fauna.all[0].dead && n < 50; n++ {
			g.strike(prey{i: 0})
		}
		return n, g.fauna.all[0].carved
	}
	glass, carved := hits(world.ObsidianKnife)
	flint, _ := hits(world.FlintKnife)
	if !carved || glass >= flint {
		t.Errorf("a shieldbrow died of %d hits of an obsidian knife and %d of a flint one, carved %v: wanted fewer, and carved", glass, flint, carved)
	}
}

func TestAnObsidianPickBreaksOutCrystalAndWearsFast(t *testing.T) {
	g := digging(t, world.CrystalOre)
	g.items[0] = world.Stack{Item: world.ObsidianPick, Count: 1, Wear: world.ObsidianPick.Uses() - 1}
	hold(t, g, seconds(3), 'q')
	if g.world.At(20, 5, 21) != world.Air {
		t.Fatal("an obsidian pick did not break crystal in three seconds")
	}
	if n := shards(g); n != 1 {
		t.Errorf("the crystal left %d shards", n)
	}
	if g.items.count([]world.Item{world.ObsidianPick}) != 0 {
		t.Error("an obsidian pick on its last use is still in hand")
	}

	g = digging(t, world.CrystalOre)
	g.items[0] = world.Stack{Item: world.FlintPick, Count: 1}
	hold(t, g, seconds(12), 'q')
	if n := shards(g); n != 0 || g.world.At(20, 5, 21) != world.Air {
		t.Errorf("a flint pick broke out %d crystal shards, and left %v there", n, g.world.At(20, 5, 21))
	}
}

// shards counts the crystal shards a game holds: lying on the ground, and carried.
func shards(g *Game) int {
	n := g.items.count([]world.Item{world.CrystalShard})
	for _, d := range g.drops {
		if d.Stack.Item == world.CrystalShard {
			n += d.Stack.Count
		}
	}
	return n
}

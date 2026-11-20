package game

import (
	"testing"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/world"
)

var skyIronTools = []world.Item{world.SkyIronKnife, world.SkyIronPick, world.SkyIronAxe, world.SkyIronSpear}

func TestToolsOfSkyIronAreHammeredAtAKnappingStone(t *testing.T) {
	g := inSurvival(t)
	g.items[0] = world.Stack{Item: world.SkyIron, Count: 8}
	g.items[1] = world.Stack{Item: world.Stick, Count: 8}
	g.items[2] = world.Stack{Item: world.Cord, Count: 4}

	for _, item := range skyIronTools {
		if _, ok := recipeFor(g, item); ok {
			t.Errorf("%s can be made away from a knapping stone", item.Name())
		}
	}
	g.world.Set(22, 6, 22, world.KnappingStone)
	for _, item := range skyIronTools {
		r, ok := recipeFor(g, item)
		if !ok || r.at != atBench || !g.craft(r) {
			t.Fatalf("%s could not be made of 8 sky iron, 8 sticks and 4 cords at a knapping stone", item.Name())
		}
	}
	for item, want := range map[world.Item]int{
		world.SkyIronKnife: 1, world.SkyIronPick: 1, world.SkyIronAxe: 1, world.SkyIronSpear: 1,
		world.SkyIron: 0, world.Stick: 0, world.Cord: 0,
	} {
		if n := g.items.count([]world.Item{item}); n != want {
			t.Errorf("the player carries %d of %s, wanted %d", n, item.Name(), want)
		}
	}
	// One lump is not enough for a pick.
	g.items[5] = world.Stack{Item: world.SkyIron, Count: 2}
	g.items[6] = world.Stack{Item: world.Stick, Count: 2}
	g.items[7] = world.Stack{Item: world.Cord, Count: 1}
	if r, ok := recipeFor(g, world.SkyIronPick); ok && g.craft(r) {
		t.Error("a sky iron pick was made of two lumps, wanted three")
	}
}

func TestASkyIronSpearIsThrownForEightAndASkyIronKnifeCarves(t *testing.T) {
	g := thrower(t, world.SkyIronSpear, 0)
	one := quarry(7)
	one.young, one.Pos.X, one.Pos.Z = false, 0.5, 7.5
	g.fauna.all = []beast{one}
	hold(t, g, 1, 'r')
	if len(g.missiles) != 1 || g.items[0].Count != 0 {
		t.Fatalf("R with a sky iron spear: %d in the air, %+v in hand", len(g.missiles), g.items[0])
	}
	flies(t, g)
	if lost := 100 - g.fauna.all[0].health; lost != 8 {
		t.Errorf("a thrown sky iron spear took %v points, wanted 8", lost)
	}

	// A kill with a sky iron knife is carved, in fewer hits than with one of obsidian.
	hits := func(with world.Item) (n int, carved bool) {
		g := among(t, -30.5, giant(render.Shieldbrow, 0.5, 20.5))
		holding(g, with, 1)
		for ; !g.fauna.all[0].dead && n < 50; n++ {
			g.strike(prey{i: 0})
		}
		return n, g.fauna.all[0].carved
	}
	iron, carved := hits(world.SkyIronKnife)
	glass, _ := hits(world.ObsidianKnife)
	if !carved || iron >= glass {
		t.Errorf("a shieldbrow died of %d hits of a sky iron knife and %d of an obsidian one, carved %v: wanted fewer, and carved", iron, glass, carved)
	}
}

func TestASkyIronPickBreaksOutACoreFasterThanAFlintOne(t *testing.T) {
	lumps := func(with world.Item, time float64) int {
		g := digging(t, world.SkyIronCore)
		g.items[0] = world.Stack{Item: with, Count: 1}
		hold(t, g, seconds(time), 'q')
		if g.world.At(20, 5, 21) != world.Air {
			return -1
		}
		n := g.items.count([]world.Item{world.SkyIron})
		for _, d := range g.drops {
			if d.Item == world.SkyIron {
				n += d.Count
			}
		}
		return n
	}
	if n := lumps(world.SkyIronPick, 2.5); n < 2 || n > 4 {
		t.Errorf("a sky iron pick broke a core in two and a half seconds and got %d lumps: wanted the core broken, and two to four", n)
	}
	if n := lumps(world.FlintPick, 2.5); n != -1 {
		t.Errorf("a flint pick broke a core in two and a half seconds, and got %d lumps", n)
	}
	if n := lumps(world.FlintPick, 4.5); n < 2 || n > 4 {
		t.Errorf("a flint pick broke a core in four and a half seconds and got %d lumps: wanted two to four", n)
	}
}

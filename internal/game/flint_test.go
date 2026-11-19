package game

import (
	"math"
	"math/rand/v2"
	"testing"

	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/world"
)

// quarry is a bugler on the test floor, a way ahead of the player, that stands still.
func quarry(ahead float64) beast {
	return beast{
		kind: render.Bugler, young: true, health: 100, timer: math.Inf(1),
		Pos: geometry.Vector3{X: 20.5, Y: 6, Z: 20.5 + ahead}, rng: rand.New(rand.NewPCG(1, 2)),
	}
}

func TestAClubKnocksABeastFartherThanAHand(t *testing.T) {
	thrown := func(with world.Item) float64 {
		g := inSurvival(t)
		g.fauna.timer = math.Inf(1)
		g.fauna.all = []beast{quarry(2)}
		holding(g, with, 1)
		g.strike(prey{i: 0})
		g.fauna.all[0].flee, g.fauna.all[0].walking = 0, false // only the knock moves it
		hold(t, g, seconds(1))
		return g.fauna.all[0].Pos.Z - 22.5
	}
	hand, club := thrown(world.NoItem), thrown(world.Club)
	if hand < 0.3 || club < 1.5*hand {
		t.Errorf("a hand knocked the beast %.2f back and a club %.2f, wanted the club half as far again or more", hand, club)
	}
}

func TestASpearReachesABeastABlockFartherThanAHand(t *testing.T) {
	for _, tc := range []struct {
		with  world.Item
		ahead float64
		aimed bool
	}{
		{world.NoItem, reach - 1, true},
		{world.NoItem, reach + 0.4, false},
		{world.Spear, reach + 0.4, true},
		{world.Spear, reach + 1.6, false},
	} {
		g := inSurvival(t)
		g.fauna.timer = math.Inf(1)
		one := quarry(tc.ahead)
		one.young = false
		one.Pos.Z += 0.9 // the near face of its box is then the distance ahead
		g.fauna.all = []beast{one}
		holding(g, tc.with, 1)
		hold(t, g, 1)
		if g.preyed != tc.aimed {
			t.Errorf("with %q a beast %.1f ahead is aimed at: %v, wanted %v", tc.with.Name(), tc.ahead, g.preyed, tc.aimed)
		}
	}

	// A spear reaches no farther for a block.
	g := lookingDown(t)
	wallAt(g, 26) // its face is 5.5 ahead
	g.player.Pitch = 0
	holding(g, world.Spear, 1)
	hold(t, g, 1)
	if g.aimed {
		t.Errorf("with a spear a wall 5.5 ahead is aimed at: %+v", g.target)
	}
}

func TestAKillWithAKnifeLeavesMoreThanOneWithout(t *testing.T) {
	left := func(with world.Item, kind render.Kind) (meat, hides int) {
		g := inSurvival(t)
		g.fauna.timer = math.Inf(1)
		one := quarry(2)
		one.kind, one.young, one.health = kind, false, 1
		g.fauna.all = []beast{one}
		holding(g, with, 1)
		g.strike(prey{i: 0})
		hold(t, g, int(fallTime*30)+3)
		for _, d := range g.drops {
			switch d.Item {
			case world.Meat:
				meat += d.Count
			case world.Hide:
				hides += d.Count
			}
		}
		return meat, hides
	}
	b := breeds[render.Bugler]
	if meat, hides := left(world.Club, render.Bugler); meat != b.meat || hides != b.hide || b.hide != 1 {
		t.Errorf("a bugler killed with a club left %d meat and %d hides, wanted %d and %d", meat, hides, b.meat, b.hide)
	}
	if meat, hides := left(world.FlintKnife, render.Bugler); meat != b.meat+1 || hides != 1 {
		t.Errorf("a bugler killed with a knife left %d meat and %d hides, wanted %d and 1", meat, hides, b.meat+1)
	}
	if meat, hides := left(world.Club, render.Nipper); meat != 1 || hides != 0 {
		t.Errorf("a nipper killed with a club left %d meat and %d hides, wanted 1 and none", meat, hides)
	}
	if meat, hides := left(world.FlintKnife, render.Nipper); meat != 2 || hides != 1 {
		t.Errorf("a nipper killed with a knife left %d meat and %d hides, wanted 2 and 1", meat, hides)
	}
}

func TestAFurBedIsMadeOfHidesAndMoss(t *testing.T) {
	g := inSurvival(t)
	g.items[0] = world.Stack{Item: world.Hide, Count: 3}
	g.items[1] = world.Stack{Item: world.Moss, Count: 6}
	if r, ok := recipeFor(g, world.FurBed.Item()); !ok || !g.craft(r) {
		t.Fatal("a fur bed could not be made of three hides and six moss by hand")
	}
	if world.FurBed.Info().Name != "fur bed" || !world.FurBed.Info().Mat {
		t.Errorf("the bed is %+v", world.FurBed.Info())
	}
}

// What a world saved before holds is still known, though no more of it can be made.
func TestTheOldThingsAreStillKnown(t *testing.T) {
	for _, old := range []world.Item{
		world.Planks.Item(), world.WoodPick, world.StoneAxe, world.CopperBlade, world.IronShovel,
		world.CopperBar, world.IronBar, world.BerryLoaf, world.MossCake, world.IronOre.Item(),
	} {
		if !old.Known() || old.Name() == "" {
			t.Errorf("item %d is no longer known", old)
		}
	}
}

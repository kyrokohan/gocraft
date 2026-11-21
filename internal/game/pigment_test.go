package game

import (
	"testing"

	"github.com/kyrokohan/skyspear/internal/world"
)

func TestOchreCoalAndChalkAreGroundIntoFourPaintsAtAKnappingStone(t *testing.T) {
	g := inSurvival(t)
	g.items[0] = world.Stack{Item: world.Ochre, Count: 1}
	g.items[1] = world.Stack{Item: world.Coal, Count: 1}
	g.items[2] = world.Stack{Item: world.Chalk, Count: 1}
	paints := map[world.Item]world.Item{world.RedPaint: world.Ochre, world.BlackPaint: world.Coal, world.WhitePaint: world.Chalk}
	for paint := range paints {
		if _, ok := recipeFor(g, paint); ok {
			t.Errorf("%s can be ground away from a knapping stone", paint.Name())
		}
	}
	g.world.Set(22, 6, 22, world.KnappingStone)
	for paint, of := range paints {
		r, ok := recipeFor(g, paint)
		if !ok || r.at != atBench || !g.craft(r) {
			t.Fatalf("%s could not be ground of one %s at a knapping stone", paint.Name(), of.Name())
		}
		if n, left := g.items.count([]world.Item{paint}), g.items.count([]world.Item{of}); n != 4 || left != 0 {
			t.Errorf("one %s ground into %d of %s, and %d are left", of.Name(), n, paint.Name(), left)
		}
	}
}

func TestAHearthCooksALogIntoCoal(t *testing.T) {
	for _, log := range logs {
		k := &world.HearthStore{In: world.Stack{Item: log, Count: 1}, Fuel: world.Stack{Item: world.Coal, Count: 1}}
		kindle(k)
		stepHearth(k, cookTime+0.1)
		if k.Out != (world.Stack{Item: world.Coal, Count: 1}) {
			t.Errorf("of a %s the hearth made %+v, wanted one coal", log.Name(), k.Out)
		}
	}
}

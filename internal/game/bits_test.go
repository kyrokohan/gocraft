package game

import (
	"testing"

	"github.com/kyrokohan/gocraft/internal/world"
)

func TestABrokenBlockThrowsBitsThatFallAndAreGone(t *testing.T) {
	g := lookingDown(t)
	hold(t, g, 1, 'q')
	if len(g.bits) != bitsABreak || g.bits[0].item != world.Stone.Item() {
		t.Fatalf("%d bits of %v, wanted %d of stone", len(g.bits), g.bits[0].item, bitsABreak)
	}

	hold(t, g, seconds(0.35))
	for _, b := range g.bits {
		at := g.world.At(int(b.pos.X), int(b.pos.Y-bitSize/2), int(b.pos.Z))
		if at.Info().Solid || b.pos.Y < 5 {
			t.Errorf("a bit is at %v, in %s or under the floor", b.pos, at.Info().Name)
		}
	}
	hold(t, g, seconds(bitLife))
	if len(g.bits) != 0 {
		t.Errorf("%d bits are still about", len(g.bits))
	}
}

func TestAPlantLeavesNoBitsAndThereAreNeverTooMany(t *testing.T) {
	g := lookingDown(t)
	g.shatter(20, 6, 21, world.TallGrass)
	if len(g.bits) != 0 {
		t.Errorf("tall grass threw %d bits", len(g.bits))
	}
	for i := range 2 * maxBits / bitsABreak {
		g.shatter(i, 5, 0, world.Dirt)
	}
	if len(g.bits) != maxBits {
		t.Errorf("%d bits, wanted no more than %d", len(g.bits), maxBits)
	}
}

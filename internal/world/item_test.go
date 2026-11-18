package world

import (
	"testing"

	"github.com/kyrokohan/gocraft/internal/tile"
)

func TestABlockIsTheItemOfItsNumber(t *testing.T) {
	for b := Grass; b < blockCount; b++ {
		got, ok := b.Item().Block()
		if !ok || got != b || !b.Item().Known() || b.Item().Name() != b.Info().Name || b.Item().Icon() != b.Info().Top {
			t.Errorf("%v as an item is %v, %v, named %q", b.Info().Name, got, ok, b.Item().Name())
		}
	}
	if _, ok := Berries.Block(); ok || !Berries.Known() || Berries.Name() != "berries" || Berries.Icon() != tile.Berries {
		t.Error("berries are a block, or unknown, or not named or drawn as berries")
	}
	for _, unknown := range []Item{NoItem, Item(blockCount), itemEnd, 9999} {
		if unknown.Known() || unknown.Name() != "" {
			t.Errorf("item %d is known as %q", unknown, unknown.Name())
		}
	}
}

func TestOnlyFoodCanBeEaten(t *testing.T) {
	if Berries.Food() <= 0 || Berries.Food() > MaxFood {
		t.Errorf("berries give %d food", Berries.Food())
	}
	if Dirt.Item().Food() != 0 || Bush.Item().Food() != 0 || itemEnd.Food() != 0 {
		t.Error("something that is not food can be eaten")
	}
}

func TestABrokenBlockDropsWhatItIsMadeOf(t *testing.T) {
	for _, tc := range []struct {
		block Block
		item  Item
		count int
	}{
		{Dirt, Dirt.Item(), 1},
		{Grass, Dirt.Item(), 1},
		{Stone, Cobble.Item(), 1},
		{Log, Log.Item(), 1},
		{Torch, Torch.Item(), 1},
		{Bush, Berries, 2},
		{Leaves, NoItem, 0},
		{TallGrass, NoItem, 0},
	} {
		if item, count := tc.block.Drop(); item != tc.item || count != tc.count {
			t.Errorf("%v drops %d of %q, wanted %d of %q", tc.block.Info().Name, count, item.Name(), tc.count, tc.item.Name())
		}
	}
}

// Every block takes a time to dig: none for a plant, a little for leaves, and more
// for stone than for dirt.
func TestBlocksTakeTimeToDig(t *testing.T) {
	for b := Grass; b < blockCount; b++ {
		info := b.Info()
		switch {
		case b == Water || info.Fixed:
		case info.Plant && info.Dig != 0:
			t.Errorf("%v is a plant and takes %v s to dig", info.Name, info.Dig)
		case info.Solid && !(info.Dig > 0 && info.Dig <= 5):
			t.Errorf("%v takes %v s to dig", info.Name, info.Dig)
		}
	}
	if Stone.Info().Dig <= Dirt.Info().Dig || Leaves.Info().Dig >= Dirt.Info().Dig {
		t.Error("stone is no harder than dirt, or leaves no softer")
	}
}

func TestAStackIsEmptyOrHoldsWhatASlotCan(t *testing.T) {
	for s, want := range map[Stack]bool{
		{}:                           true,
		{Dirt.Item(), 1}:             true,
		{Berries, StackSize}:         true,
		{Dirt.Item(), 0}:             false,
		{Dirt.Item(), StackSize + 1}: false,
		{NoItem, 3}:                  false,
		{Item(blockCount), 1}:        false,
		{Dirt.Item(), -1}:            false,
	} {
		if s.valid() != want {
			t.Errorf("%+v valid: %v, wanted %v", s, s.valid(), want)
		}
	}
}

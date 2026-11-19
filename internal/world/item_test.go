package world

import (
	"testing"

	"github.com/kyrokohan/skyspear/internal/tile"
)

func TestABlockIsTheItemOfItsNumber(t *testing.T) {
	for b := Turf; b < blockCount; b++ {
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
		{Turf, Dirt.Item(), 1},
		{Stone, Cobble.Item(), 1},
		{Log, Log.Item(), 1},
		{Torch, Torch.Item(), 1},
		{Bush, Berries, 2},
		{Leaves, NoItem, 0},
		{Fern, Fiber, 1},
		{TallFern, Fiber, 2},
		{FlintSeam, Flint, 1},
	} {
		if item, count := tc.block.Drop(); item != tc.item || count != tc.count {
			t.Errorf("%v drops %d of %q, wanted %d of %q", tc.block.Info().Name, count, item.Name(), tc.count, tc.item.Name())
		}
	}
}

// Every block takes a time to dig: none for a plant, a little for leaves, and more
// for stone than for dirt.
func TestBlocksTakeTimeToDig(t *testing.T) {
	for b := Turf; b < blockCount; b++ {
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
		{}:                                true,
		{Item: Dirt.Item(), Count: 1}:     true,
		{Item: Berries, Count: StackSize}: true,
		{Item: Dirt.Item(), Count: 0}:     false,
		{Item: Dirt.Item(), Count: StackSize + 1}: false,
		{Item: NoItem, Count: 3}:                  false,
		{Item: Item(blockCount), Count: 1}:        false,
		{Item: Dirt.Item(), Count: -1}:            false,
	} {
		if s.valid() != want {
			t.Errorf("%+v valid: %v, wanted %v", s, s.valid(), want)
		}
	}
}

func TestAToolIsOneToASlotAndWearsOut(t *testing.T) {
	for _, tc := range []struct {
		stack Stack
		valid bool
	}{
		{Stack{Item: StonePick, Count: 1}, true},
		{Stack{Item: StonePick, Count: 1, Wear: StonePick.Uses() - 1}, true},
		{Stack{Item: StonePick, Count: 1, Wear: StonePick.Uses()}, false},
		{Stack{Item: StonePick, Count: 2}, false},
		{Stack{Item: StonePick, Count: 1, Wear: -1}, false},
		{Stack{Item: Stick, Count: 1, Wear: 1}, false},
	} {
		if tc.stack.valid() != tc.valid {
			t.Errorf("%+v valid: %v, wanted %v", tc.stack, tc.stack.valid(), tc.valid)
		}
	}
	if WoodPick.Uses() >= IronPick.Uses() || Stick.Uses() != 0 || Stick.StackSize() != StackSize {
		t.Error("an iron tool wears out no slower than a wood one, or a stick wears")
	}
}

func TestTheRightToolDigsFasterAndSomeBlocksNeedOne(t *testing.T) {
	for _, tc := range []struct {
		block  Block
		hand   Item
		faster float64 // than a hand that can dig it
		yields bool
	}{
		{Dirt, NoItem, 1, true},
		{Dirt, WoodShovel, 2, true},
		{Dirt, IronShovel, 6, true},
		{Dirt, IronPick, 1, true},   // the wrong kind is no help
		{Log, StoneAxe, 3, true},    // and the right kind is
		{Stone, NoItem, 1, true},    // by hand stone breaks to rubble, slowly
		{Stone, FlintPick, 3, true}, // and a pick is quicker
		{CoalOre, NoItem, 1, false}, // coal needs a flint pick
		{CoalOre, FlintPick, 3, true},
		{FlintSeam, NoItem, 1, true},
		{Log, HandAxe, 3, true},
		{Dirt, DiggingStick, 2, true},
		{IronOre, WoodPick, 1, false},
		{IronOre, StonePick, 3, true},
		{CrystalOre, StonePick, 1, false},
		{CrystalOre, CopperPick, 4, true},
		{Fern, NoItem, 1, true},
	} {
		seconds, yields := tc.block.DigWith(tc.hand)
		want := tc.block.Info().Dig / tc.faster
		if !yields {
			want = tc.block.Info().Dig * 3
		}
		if seconds != want || yields != tc.yields {
			t.Errorf("%v with %q: %v s, yields %v; wanted %v s, %v", tc.block.Info().Name, tc.hand.Name(), seconds, yields, want, tc.yields)
		}
	}
}

func TestOresLeaveWhatIsInThem(t *testing.T) {
	if item, n := CoalOre.Drop(); item != Coal || n != 1 {
		t.Errorf("coal ore drops %d %q", n, item.Name())
	}
	if item, n := CrystalOre.Drop(); item != CrystalShard || n != 1 {
		t.Errorf("crystal ore drops %d %q", n, item.Name())
	}
	if item, _ := IronOre.Drop(); item != IronOre.Item() {
		t.Errorf("iron ore drops %q, wanted itself for the kiln", item.Name())
	}
}

func TestSomeLeavesHoldAFruitAndAlwaysTheSameOnes(t *testing.T) {
	fruit := 0
	for x := range 100 {
		for z := range 10 {
			item, n := Leaves.DropAt(x, 40, z)
			if again, _ := Leaves.DropAt(x, 40, z); again != item {
				t.Fatal("the same leaves gave a fruit once and not again")
			}
			if item == Fruit && n == 1 {
				fruit++
			} else if item != NoItem {
				t.Fatalf("leaves dropped %q", item.Name())
			}
		}
	}
	if fruit < 60 || fruit > 140 {
		t.Errorf("%d of 1000 leaves hold a fruit, wanted about 100", fruit)
	}
	if item, _ := Dirt.DropAt(3, 4, 5); item != Dirt.Item() {
		t.Error("DropAt changed what dirt drops")
	}
}

func TestABladeHitsHarderTheBetterItIs(t *testing.T) {
	if NoItem.Damage() != 1 || Dirt.Item().Damage() != 1 || WoodPick.Damage() != 2 {
		t.Errorf("a hand hits %v, dirt %v and a pick %v: wanted 1, 1 and 2",
			NoItem.Damage(), Dirt.Item().Damage(), WoodPick.Damage())
	}
	last := WoodPick.Damage()
	for _, b := range []Item{WoodBlade, StoneBlade, CopperBlade, IronBlade} {
		if b.Damage() <= last || b.StackSize() != 1 || b.Uses() == 0 {
			t.Errorf("%s hits %v after %v, stacks %d, lasts %d", b.Name(), b.Damage(), last, b.StackSize(), b.Uses())
		}
		last = b.Damage()
	}
	if s, _ := Stone.DigWith(IronBlade); s != Stone.Info().Dig {
		t.Errorf("a blade digs stone in %v s, as a hand does in %v", s, Stone.Info().Dig)
	}
}

func TestFlintIsFoundInGravelNowAndThenAndAlwaysInTheSamePlaces(t *testing.T) {
	flint := 0
	for x := range 100 {
		for z := range 10 {
			item, n := Gravel.DropAt(x, 40, z)
			if again, _ := Gravel.DropAt(x, 40, z); again != item {
				t.Fatalf("the gravel at %d,40,%d left two different things", x, z)
			}
			switch item {
			case Flint:
				flint++
			case Gravel.Item():
			default:
				t.Fatalf("gravel left %d of %q", n, item.Name())
			}
		}
	}
	if flint < 60 || flint > 200 {
		t.Errorf("%d of 1,000 blocks of gravel held a flint, wanted about one in eight", flint)
	}
}

// The tools of the age: flint lasts longer bound to a handle than held in the hand, a
// club and a spear are for hitting, and a knife is for the kill.
func TestTheToolsOfFlintAndWood(t *testing.T) {
	for _, tc := range []struct {
		item Item
		kind Tool
		tier int
	}{
		{HandAxe, Axe, TierFlint}, {FlintPick, Pick, TierFlint}, {FlintKnife, Blade, TierFlint},
		{DiggingStick, Shovel, TierWood}, {Club, Blade, TierWood}, {Spear, Blade, TierWood},
	} {
		kind, tier := tc.item.Tool()
		if kind != tc.kind || tier != tc.tier || tc.item.StackSize() != 1 || tc.item.Uses() == 0 || tc.item.Name() == "" {
			t.Errorf("%q is tool %d of tier %d, %d to a slot, lasting %d", tc.item.Name(), kind, tier, tc.item.StackSize(), tc.item.Uses())
		}
	}
	if HandAxe.Uses() >= FlintPick.Uses() {
		t.Errorf("a hand axe lasts %d and a pick on a handle %d: wanted the handle to last longer", HandAxe.Uses(), FlintPick.Uses())
	}
	if Club.Damage() <= NoItem.Damage() || Club.Knock() <= NoItem.Knock() || NoItem.Knock() != 1 {
		t.Errorf("a club hits %v and knocks %v times as far as a hand", Club.Damage(), Club.Knock())
	}
	if Spear.Reach() < 1 || Club.Reach() != 0 || NoItem.Reach() != 0 {
		t.Errorf("a spear reaches %v farther, a club %v", Spear.Reach(), Club.Reach())
	}
	if !FlintKnife.Carves() || Club.Carves() || NoItem.Carves() {
		t.Error("only a knife carves a kill")
	}
	for _, thing := range []Item{Fiber, Cord, Flint, Hide} {
		if thing.Name() == "" || thing.StackSize() != StackSize || thing.Food() != 0 {
			t.Errorf("%q stacks %d and feeds %d", thing.Name(), thing.StackSize(), thing.Food())
		}
	}
}

package game

import (
	"testing"

	"github.com/kyrokohan/skyspear/internal/world"
)

func TestThingsPickedUpStackAndFillTheHotbarFirst(t *testing.T) {
	var inv inventory
	dirt, sand := world.Dirt.Item(), world.Sand.Item()

	if left := inv.add(world.Stack{Item: dirt, Count: 10}); left != 0 || inv[0] != (world.Stack{Item: dirt, Count: 10}) {
		t.Fatalf("10 dirt: %d left, slot 0 holds %+v", left, inv[0])
	}
	inv.add(world.Stack{Item: sand, Count: 1})
	inv.add(world.Stack{Item: dirt, Count: 60}) // 54 more fit with the first 10, and 6 go in the next empty slot
	if inv[0].Count != world.StackSize || inv[1] != (world.Stack{Item: sand, Count: 1}) || inv[2] != (world.Stack{Item: dirt, Count: 6}) {
		t.Errorf("after 70 dirt and a sand the first slots hold %+v", inv[:3])
	}
}

func TestWhatDoesNotFitIsLeftOver(t *testing.T) {
	var inv inventory
	for i := range inv {
		inv[i] = world.Stack{Item: world.Stone.Item(), Count: world.StackSize}
	}
	inv[slots-1].Count = 60

	if left := inv.add(world.Stack{Item: world.Stone.Item(), Count: 10}); left != 6 || inv[slots-1].Count != world.StackSize {
		t.Errorf("10 stone into room for 4: %d left, the last slot holds %d", left, inv[slots-1].Count)
	}
	if left := inv.add(world.Stack{Item: world.Dirt.Item(), Count: 3}); left != 3 {
		t.Errorf("3 dirt into a full inventory: %d left, wanted all 3", left)
	}
}

func TestTakingTheLastOneEmptiesTheSlot(t *testing.T) {
	inv := inventory{{Item: world.Dirt.Item(), Count: 2}}

	inv.takeOne(0)
	if inv[0].Count != 1 {
		t.Fatalf("one taken from 2 leaves %+v", inv[0])
	}
	inv.takeOne(0)
	if inv[0] != (world.Stack{}) {
		t.Errorf("the last one taken leaves %+v, wanted an empty slot", inv[0])
	}
}

func TestACreativeWorldStartsWithTheHotbarFull(t *testing.T) {
	inv := creativeStart()
	for i, b := range hotbar {
		if inv[i].Item != b.Item() || inv[i].Count != 1 {
			t.Errorf("slot %d holds %+v, wanted a %v", i, inv[i], b.Info().Name)
		}
	}
	for _, s := range inv[hotbarSlots:] {
		if s != (world.Stack{}) {
			t.Errorf("the pack holds %+v, wanted it empty", s)
		}
	}
}

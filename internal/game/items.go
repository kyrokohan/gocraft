package game

import "github.com/kyrokohan/gocraft/internal/world"

// What the player carries: the hotbar, on the keys 1 to 9, and the pack behind it.
const (
	hotbarSlots = 9
	packSlots   = 27
	slots       = hotbarSlots + packSlots
)

// inventory is the player's slots: the hotbar's first, and then the pack's.
type inventory [slots]world.Stack

// hotbar is what a creative world starts with on the hotbar.
var hotbar = [hotbarSlots]world.Block{
	world.Grass, world.Dirt, world.Stone, world.Sand, world.Log,
	world.Torch, world.Planks, world.Brick, world.Glass,
}

// creativeStart is the inventory of a creative world before anything is changed in it.
func creativeStart() (inv inventory) {
	for i, b := range hotbar {
		inv[i] = world.Stack{Item: b.Item(), Count: 1}
	}
	return inv
}

// add puts n of an item into the inventory, and returns how many did not fit. They go
// on the stacks of that item that have room first, and then into the first empty slots,
// the hotbar before the pack, so what is picked up is at hand.
func (inv *inventory) add(item world.Item, n int) (left int) {
	for i := range inv {
		if s := &inv[i]; s.Item == item && s.Count < world.StackSize {
			put := min(n, world.StackSize-s.Count)
			s.Count += put
			n -= put
		}
	}
	for i := range inv {
		if s := &inv[i]; n > 0 && s.Count == 0 {
			put := min(n, world.StackSize)
			*s = world.Stack{Item: item, Count: put}
			n -= put
		}
	}
	return n
}

// takeOne takes one thing from a slot.
func (inv *inventory) takeOne(slot int) {
	if s := &inv[slot]; s.Count > 1 {
		s.Count--
	} else {
		*s = world.Stack{}
	}
}

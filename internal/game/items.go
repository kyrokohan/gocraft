package game

import (
	"slices"

	"github.com/kyrokohan/skyspear/internal/world"
)

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
	world.Turf, world.Dirt, world.Stone, world.Sand, world.Log,
	world.Torch, world.Planks, world.Brick, world.Glass,
}

// creativeStart is the inventory of a creative world before anything is changed in it.
func creativeStart() (inv inventory) {
	for i, b := range hotbar {
		inv[i] = world.Stack{Item: b.Item(), Count: 1}
	}
	return inv
}

// add puts a stack into the inventory, and returns how many of it did not fit. They go
// on the stacks of that item that have room first, and then into the first empty slots,
// the hotbar before the pack, so what is picked up is at hand. A tool is one to a slot,
// and keeps its wear.
func (inv *inventory) add(s world.Stack) (left int) {
	return addTo(inv[:], s)
}

// count returns how many the inventory holds of any of some items.
func (inv *inventory) count(items []world.Item) (n int) {
	for _, s := range inv {
		if slices.Contains(items, s.Item) {
			n += s.Count
		}
	}
	return n
}

// use takes n of some items out of the inventory, of the first item of them that it has
// first, and from the pack before the hotbar. It has them: count says so.
func (inv *inventory) use(items []world.Item, n int) {
	for _, item := range items {
		for i := len(inv) - 1; i >= 0 && n > 0; i-- {
			if s := &inv[i]; s.Item == item {
				took := min(n, s.Count)
				if s.Count -= took; s.Count == 0 {
					*s = world.Stack{}
				}
				n -= took
			}
		}
	}
}

// takeOne takes one thing from a slot.
func (inv *inventory) takeOne(slot int) {
	if s := &inv[slot]; s.Count > 1 {
		s.Count--
	} else {
		*s = world.Stack{}
	}
}

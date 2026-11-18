package world

import "github.com/kyrokohan/gocraft/internal/tile"

// Item is a kind of thing the player can carry. From 1 to 255 an item is the block of
// that number, and from 256 up it is a thing that is not a block. The numbers are
// written into saves, so a kind keeps its number: add new ones at the end.
type Item uint16

// NoItem is nothing: what an empty slot holds.
const NoItem Item = 0

const (
	Berries Item = 256 + iota

	itemEnd
)

// StackSize is the most of one item that one slot holds.
const StackSize = 64

// thing is what is known about an item that is not a block.
type thing struct {
	name string
	icon tile.ID
	food int // how much eating one gives back, out of MaxFood
}

var things = [itemEnd - 256]thing{
	Berries - 256: {name: "berries", icon: tile.Berries, food: 4},
}

// MaxFood is how full the player can be.
const MaxFood = 20

// Item returns the item that is the block.
func (b Block) Item() Item {
	return Item(b)
}

// Block returns the block that the item is, and false for an item that is not a block.
func (i Item) Block() (Block, bool) {
	if i == NoItem || i >= 256 {
		return Air, false
	}
	return Block(i), true
}

// Known reports whether i is a kind of item this version of the game has.
func (i Item) Known() bool {
	if b, ok := i.Block(); ok {
		return b.Known()
	}
	return i >= 256 && i < itemEnd
}

// Name is what the player is shown.
func (i Item) Name() string {
	if b, ok := i.Block(); ok {
		return b.Info().Name
	}
	if i.Known() {
		return things[i-256].name
	}
	return ""
}

// Icon is the tile the item is shown by: a block's top.
func (i Item) Icon() tile.ID {
	if b, ok := i.Block(); ok {
		return b.Info().Top
	}
	if i.Known() {
		return things[i-256].icon
	}
	return 0
}

// Food is how much eating one of the item gives back, out of MaxFood: 0 for an item
// that cannot be eaten.
func (i Item) Food() int {
	if i >= 256 && i.Known() {
		return things[i-256].food
	}
	return 0
}

// Drop returns what a block leaves when it is broken, and how many: mostly the block
// itself, and nothing for leaves and grass that crumble away.
func (b Block) Drop() (Item, int) {
	switch b {
	case Air, Water, Leaves, PineNeedles, PaleLeaves, TallGrass, DryBush:
		return NoItem, 0
	case Grass:
		return Dirt.Item(), 1 // the turf does not come away
	case Stone:
		return Cobble.Item(), 1 // it breaks to rubble
	case Bush:
		return Berries, 2
	}
	return b.Item(), 1
}

// Stack is a number of one item, as a slot holds them. The zero Stack is an empty slot.
type Stack struct {
	Item  Item `json:"item"`
	Count int  `json:"count"`
}

// valid reports whether a stack is empty, or a known item and a number a slot can hold.
func (s Stack) valid() bool {
	return s == Stack{} || s.Item.Known() && s.Count >= 1 && s.Count <= StackSize
}

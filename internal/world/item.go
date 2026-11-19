package world

import "github.com/kyrokohan/skyspear/internal/tile"

// Item is a kind of thing the player can carry. From 1 to 255 an item is the block of
// that number, and from 256 up it is a thing that is not a block. The numbers are
// written into saves, so a kind keeps its number: add new ones at the end.
type Item uint16

// NoItem is nothing: what an empty slot holds.
const NoItem Item = 0

const (
	Berries Item = 256 + iota
	Stick
	Coal
	CopperBar
	IronBar
	CrystalShard
	Fruit
	BerryLoaf

	WoodPick
	WoodAxe
	WoodShovel
	StonePick
	StoneAxe
	StoneShovel
	CopperPick
	CopperAxe
	CopperShovel
	IronPick
	IronAxe
	IronShovel

	WoodBlade
	StoneBlade
	CopperBlade
	IronBlade
	Moss
	MossCake

	Meat // raw, from a beast that died

	itemEnd
)

// StackSize is the most of one item that one slot holds, but for a tool, which is one
// to a slot.
const StackSize = 64

// Tool is a kind of tool: what kind of block it digs faster.
type Tool uint8

const (
	NoTool Tool = iota
	Pick        // stone, ore and brick
	Axe         // wood
	Shovel      // earth, sand and snow
	Blade       // nothing: it is for hitting
)

// The tiers of tools, from the first to the best, as Item.Tool gives them. A block that
// needs a tool needs one of its tier or better.
const (
	TierWood = 1 + iota
	TierStone
	TierCopper
	TierIron
)

// toolSpeed is how many times faster a tool of each tier digs the blocks of its kind
// than a hand does, and toolUses how many blocks it breaks before it is worn out.
var (
	toolSpeed = [...]float64{1, 2, 3, 4, 6}
	toolUses  = [...]int{0, 60, 130, 200, 250}
)

// thing is what is known about an item that is not a block.
type thing struct {
	name string
	icon tile.ID
	food int // how much eating one gives back, out of MaxFood
	tool Tool
	tier int
}

var things = [itemEnd - 256]thing{
	Berries - 256:      {name: "berries", icon: tile.Berries, food: 4},
	Stick - 256:        {name: "stick", icon: tile.Stick},
	Coal - 256:         {name: "coal", icon: tile.Coal},
	CopperBar - 256:    {name: "copper bar", icon: tile.CopperBar},
	IronBar - 256:      {name: "iron bar", icon: tile.IronBar},
	CrystalShard - 256: {name: "crystal shard", icon: tile.CrystalShard},
	Fruit - 256:        {name: "fruit", icon: tile.Fruit, food: 5},
	BerryLoaf - 256:    {name: "berry loaf", icon: tile.BerryLoaf, food: 14},

	WoodPick - 256:     {name: "wood pick", icon: tile.WoodPick, tool: Pick, tier: TierWood},
	WoodAxe - 256:      {name: "wood axe", icon: tile.WoodAxe, tool: Axe, tier: TierWood},
	WoodShovel - 256:   {name: "wood shovel", icon: tile.WoodShovel, tool: Shovel, tier: TierWood},
	StonePick - 256:    {name: "stone pick", icon: tile.StonePick, tool: Pick, tier: TierStone},
	StoneAxe - 256:     {name: "stone axe", icon: tile.StoneAxe, tool: Axe, tier: TierStone},
	StoneShovel - 256:  {name: "stone shovel", icon: tile.StoneShovel, tool: Shovel, tier: TierStone},
	CopperPick - 256:   {name: "copper pick", icon: tile.CopperPick, tool: Pick, tier: TierCopper},
	CopperAxe - 256:    {name: "copper axe", icon: tile.CopperAxe, tool: Axe, tier: TierCopper},
	CopperShovel - 256: {name: "copper shovel", icon: tile.CopperShovel, tool: Shovel, tier: TierCopper},
	IronPick - 256:     {name: "iron pick", icon: tile.IronPick, tool: Pick, tier: TierIron},
	IronAxe - 256:      {name: "iron axe", icon: tile.IronAxe, tool: Axe, tier: TierIron},
	IronShovel - 256:   {name: "iron shovel", icon: tile.IronShovel, tool: Shovel, tier: TierIron},

	WoodBlade - 256:   {name: "wood blade", icon: tile.WoodBlade, tool: Blade, tier: TierWood},
	StoneBlade - 256:  {name: "stone blade", icon: tile.StoneBlade, tool: Blade, tier: TierStone},
	CopperBlade - 256: {name: "copper blade", icon: tile.CopperBlade, tool: Blade, tier: TierCopper},
	IronBlade - 256:   {name: "iron blade", icon: tile.IronBlade, tool: Blade, tier: TierIron},
	Moss - 256:        {name: "moss", icon: tile.Moss},
	MossCake - 256:    {name: "moss cake", icon: tile.MossCake, food: 8},

	Meat - 256: {name: "raw meat", icon: tile.Meat, food: 5},
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

// thing returns what is known of an item that is not a block, and the zero thing for
// any other.
func (i Item) thing() thing {
	if i >= 256 && i < itemEnd {
		return things[i-256]
	}
	return thing{}
}

// Name is what the player is shown.
func (i Item) Name() string {
	if b, ok := i.Block(); ok {
		return b.Info().Name
	}
	return i.thing().name
}

// Icon is the tile the item is shown by: a block's top.
func (i Item) Icon() tile.ID {
	if b, ok := i.Block(); ok {
		return b.Info().Top
	}
	return i.thing().icon
}

// Food is how much eating one of the item gives back, out of MaxFood: 0 for an item
// that cannot be eaten.
func (i Item) Food() int {
	return i.thing().food
}

// Tool returns the kind of tool an item is and its tier, or NoTool.
func (i Item) Tool() (Tool, int) {
	t := i.thing()
	return t.tool, t.tier
}

// Uses is how many blocks a tool breaks before it is worn out, and 0 for an item that
// is not a tool.
func (i Item) Uses() int {
	_, tier := i.Tool()
	return toolUses[tier]
}

// bladeDamage is how hard a blade of each tier hits, in points of health.
var bladeDamage = [...]float64{1, 3, 4, 5, 6}

// Damage is how hard a hit with the item in hand is, in points of a creature's health:
// 1 for a hand or a thing that is not a tool, 2 for a tool, and a blade's by its tier.
func (i Item) Damage() float64 {
	switch kind, tier := i.Tool(); kind {
	case NoTool:
		return 1
	case Blade:
		return bladeDamage[tier]
	}
	return 2
}

// StackSize is the most of the item one slot holds.
func (i Item) StackSize() int {
	if t, _ := i.Tool(); t != NoTool {
		return 1
	}
	return StackSize
}

// DigWith returns how many seconds the block takes to dig with an item in hand (NoItem
// for a bare hand), and whether it leaves anything when it breaks. A tool of the block's
// kind digs it faster. A block that needs a tool takes three times as long without one
// of its kind and tier, and leaves nothing.
func (b Block) DigWith(hand Item) (seconds float64, yields bool) {
	info := b.Info()
	kind, tier := hand.Tool()
	if kind != info.Tool {
		tier = 0
	}
	if tier < info.Needs {
		return info.Dig * 3, false
	}
	return info.Dig / toolSpeed[tier], true
}

// Drop returns what a block leaves when it is broken, and how many: mostly the block
// itself, and nothing for leaves and grass that crumble away.
func (b Block) Drop() (Item, int) {
	switch b {
	case Air, Water, Leaves, PineNeedles, PaleLeaves, Fronds, FanLeaves, Fern, TallFern, PalmCrown, DryBush:
		return NoItem, 0
	case Turf:
		return Dirt.Item(), 1 // the turf does not come away
	case Stone:
		return Cobble.Item(), 1 // it breaks to rubble
	case Bush:
		return Berries, 2
	case CoalOre:
		return Coal, 1
	case CrystalOre:
		return CrystalShard, 1
	}
	return b.Item(), 1
}

// fruitChance is the part of the leaves that hold a fruit.
const fruitChance = 0.1

// DropAt is Drop for the block in a cell. Leaves now and then hold a fruit: which ones
// is a hash of the cell, so that it does not change.
func (b Block) DropAt(x, y, z int) (Item, int) {
	if b == Leaves || b == PineNeedles || b == PaleLeaves || b == FanLeaves {
		h := uint32(x)*0x9E3779B1 ^ uint32(y)*0x85EBCA77 ^ uint32(z)*0xC2B2AE3D
		h ^= h >> 15
		h *= 0x2C1B3C6D
		h ^= h >> 13
		if float64(h)/(1<<32) < fruitChance {
			return Fruit, 1
		}
	}
	return b.Drop()
}

// Stack is a number of one item, as a slot holds them. The zero Stack is an empty slot.
type Stack struct {
	Item  Item `json:"item"`
	Count int  `json:"count"`

	// Wear is how many blocks a tool has broken, out of its Uses.
	Wear int `json:"wear,omitempty"`
}

// valid reports whether a stack is empty, or a known item and a number a slot can hold,
// with no more wear than a tool can take.
func (s Stack) valid() bool {
	if s == (Stack{}) {
		return true
	}
	return s.Item.Known() && s.Count >= 1 && s.Count <= s.Item.StackSize() &&
		s.Wear >= 0 && (s.Wear == 0 || s.Wear < s.Item.Uses())
}

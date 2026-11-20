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

	// The things of the stone age.
	Fiber // from a fern
	Cord  // twisted from fiber
	Flint
	Hide
	HandAxe
	FlintPick
	FlintKnife
	DiggingStick
	Club
	Spear

	// The things of fire.
	CookedMeat
	PalmSeeds // of a stump palm: poison until they are roasted
	RoastSeeds
	Resin      // from a pine or a redwood log, now and then
	UnlitTorch // a stick with resin on it: lit at a fire, it is the torch block's item
	FireDrill
	HardSpear // a spear with its tip hardened in a fire
	SoftPot   // a pot shaped of clay, before it is baked

	// What is thrown, besides a spear.
	Pebble
	Sling

	// What a fire mountain gives.
	Obsidian  // a piece of it, broken out of a vein
	Firestone // a piece of it
	ObsidianKnife
	ObsidianPick
	ObsidianSpear
	Striker // a flint and a firestone: it strikes fire at once

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

// The tiers of tools, from the first to the best, as Item.Tool gives them: what they are
// made of. A block that needs a tool needs one of its tier or better. Obsidian is on
// the fire mountain only, and iron only falls from the sky.
const (
	TierWood = 1 + iota
	TierFlint
	TierObsidian
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

	// poison is how many points of health eating one takes.
	poison float64

	tool Tool
	tier int

	// What a tool or a weapon has of its own, where it is not what its tier gives: how
	// many uses it lasts, how hard it hits, how much farther than a hand it reaches a
	// beast, how many times as far it knocks one back, and whether it carves a kill.
	uses   int
	hits   float64
	reach  float64
	knock  float64
	carves bool

	// thrown is true for a thing that R throws as a spear is thrown.
	thrown bool
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
	StonePick - 256:    {name: "stone pick", icon: tile.StonePick, tool: Pick, tier: TierFlint},
	StoneAxe - 256:     {name: "stone axe", icon: tile.StoneAxe, tool: Axe, tier: TierFlint},
	StoneShovel - 256:  {name: "stone shovel", icon: tile.StoneShovel, tool: Shovel, tier: TierFlint},
	CopperPick - 256:   {name: "copper pick", icon: tile.CopperPick, tool: Pick, tier: TierObsidian},
	CopperAxe - 256:    {name: "copper axe", icon: tile.CopperAxe, tool: Axe, tier: TierObsidian},
	CopperShovel - 256: {name: "copper shovel", icon: tile.CopperShovel, tool: Shovel, tier: TierObsidian},
	IronPick - 256:     {name: "iron pick", icon: tile.IronPick, tool: Pick, tier: TierIron},
	IronAxe - 256:      {name: "iron axe", icon: tile.IronAxe, tool: Axe, tier: TierIron},
	IronShovel - 256:   {name: "iron shovel", icon: tile.IronShovel, tool: Shovel, tier: TierIron},

	WoodBlade - 256:   {name: "wood blade", icon: tile.WoodBlade, tool: Blade, tier: TierWood},
	StoneBlade - 256:  {name: "stone blade", icon: tile.StoneBlade, tool: Blade, tier: TierFlint},
	CopperBlade - 256: {name: "copper blade", icon: tile.CopperBlade, tool: Blade, tier: TierObsidian},
	IronBlade - 256:   {name: "iron blade", icon: tile.IronBlade, tool: Blade, tier: TierIron},
	Moss - 256:        {name: "moss", icon: tile.Moss},
	MossCake - 256:    {name: "moss cake", icon: tile.MossCake, food: 8},

	Meat - 256: {name: "raw meat", icon: tile.Meat, food: 5},

	Fiber - 256: {name: "fiber", icon: tile.Fiber},
	Cord - 256:  {name: "cord", icon: tile.Cord},
	Flint - 256: {name: "flint", icon: tile.Flint},
	Hide - 256:  {name: "hide", icon: tile.Hide},

	// Flint held in the hand wears sooner than flint bound to a handle.
	HandAxe - 256:      {name: "hand axe", icon: tile.HandAxe, tool: Axe, tier: TierFlint, uses: 60},
	FlintPick - 256:    {name: "flint pick", icon: tile.FlintPick, tool: Pick, tier: TierFlint},
	FlintKnife - 256:   {name: "flint knife", icon: tile.FlintKnife, tool: Blade, tier: TierFlint, carves: true},
	DiggingStick - 256: {name: "digging stick", icon: tile.DiggingStick, tool: Shovel, tier: TierWood},
	Club - 256:         {name: "club", icon: tile.Club, tool: Blade, tier: TierWood, hits: 4, knock: 1.8},
	Spear - 256:        {name: "spear", icon: tile.Spear, tool: Blade, tier: TierWood, hits: 3, reach: 1, thrown: true},

	CookedMeat - 256: {name: "cooked meat", icon: tile.CookedMeat, food: 12},
	PalmSeeds - 256:  {name: "palm seeds", icon: tile.PalmSeeds, food: 2, poison: 4},
	RoastSeeds - 256: {name: "roast seeds", icon: tile.RoastSeeds, food: 6},
	Resin - 256:      {name: "resin", icon: tile.Resin},
	UnlitTorch - 256: {name: "unlit torch", icon: tile.UnlitTorch},

	// A fire drill is no tool for digging, but it wears as one does: a use is a fire lit.
	FireDrill - 256: {name: "fire drill", icon: tile.FireDrill, uses: 12},
	HardSpear - 256: {name: "hardened spear", icon: tile.HardSpear, tool: Blade, tier: TierWood, hits: 4, reach: 1, uses: 150, thrown: true},
	SoftPot - 256:   {name: "soft pot", icon: tile.SoftPot},

	// A sling wears as a tool does: a use is a pebble thrown.
	Pebble - 256: {name: "pebble", icon: tile.Pebble},
	Sling - 256:  {name: "sling", icon: tile.Sling, uses: 80},

	Obsidian - 256:  {name: "obsidian", icon: tile.Obsidian},
	Firestone - 256: {name: "firestone", icon: tile.Firestone},

	// Obsidian takes the sharpest edge there is, and is the first to break: its tools
	// hit hardest and dig fastest, and are worn out in a third of the time flint is.
	ObsidianKnife - 256: {name: "obsidian knife", icon: tile.ObsidianKnife, tool: Blade, tier: TierObsidian, hits: 5, carves: true, uses: obsidianUses},
	ObsidianPick - 256:  {name: "obsidian pick", icon: tile.ObsidianPick, tool: Pick, tier: TierObsidian, uses: obsidianUses},
	ObsidianSpear - 256: {name: "obsidian spear", icon: tile.ObsidianSpear, tool: Blade, tier: TierObsidian, hits: 5, reach: 1, uses: obsidianUses, thrown: true},

	// A striker is no tool for digging, but it wears as one does: a use is a fire lit.
	Striker - 256: {name: "striker", icon: tile.Striker, uses: 30},
}

// obsidianUses is how many uses a tool of obsidian lasts.
const obsidianUses = 40

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

// Poison is how many points of health eating one of the item takes: 0 for what is good
// to eat.
func (i Item) Poison() float64 {
	return i.thing().poison
}

// Tool returns the kind of tool an item is and its tier, or NoTool.
func (i Item) Tool() (Tool, int) {
	t := i.thing()
	return t.tool, t.tier
}

// Uses is how many blocks a tool breaks before it is worn out, and 0 for an item that
// does not wear.
func (i Item) Uses() int {
	if uses := i.thing().uses; uses > 0 {
		return uses
	}
	_, tier := i.Tool()
	return toolUses[tier]
}

// Reach is how much farther than a hand the item reaches a beast, in blocks.
func (i Item) Reach() float64 {
	return i.thing().reach
}

// Knock is how many times as far as a hand a hit with the item knocks a beast back.
func (i Item) Knock() float64 {
	if knock := i.thing().knock; knock > 0 {
		return knock
	}
	return 1
}

// Thrown reports whether R throws the item as a spear is thrown.
func (i Item) Thrown() bool {
	return i.thing().thrown
}

// Carves reports whether a kill made with the item is carved: a knife's is, and leaves
// more.
func (i Item) Carves() bool {
	return i.thing().carves
}

// bladeDamage is how hard a blade of each tier hits, in points of health.
var bladeDamage = [...]float64{1, 3, 4, 5, 6}

// Damage is how hard a hit with the item in hand is, in points of a creature's health:
// 1 for a hand or a thing that is not a tool, 2 for a tool, and a blade's by its tier.
func (i Item) Damage() float64 {
	if hits := i.thing().hits; hits > 0 {
		return hits
	}
	switch kind, tier := i.Tool(); kind {
	case NoTool:
		return 1
	case Blade:
		return bladeDamage[tier]
	}
	return 2
}

// StackSize is the most of the item one slot holds: one of a thing that wears, as each
// has its own wear.
func (i Item) StackSize() int {
	if i.Uses() > 0 {
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
	case Air, Water, Leaves, PineNeedles, PaleLeaves, Fronds, FanLeaves, DryBush, Fire:
		return NoItem, 0
	case PalmCrown:
		return PalmSeeds, 2
	case CharredLog:
		return Coal, 1 // what is left of the wood burns well
	case HearthLit:
		return Hearth.Item(), 1 // broken, it goes out
	case Fern:
		return Fiber, 1
	case TallFern:
		return Fiber, 2
	case FlintSeam:
		return Flint, 1
	case ObsidianRock:
		return Obsidian, 1
	case FirestoneRock:
		return Firestone, 1
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

// fruitChance is the part of the leaves that hold a fruit, flintChance the part of the
// gravel that holds a flint, and resinChance the part of the pine and redwood logs that
// hold resin.
const (
	fruitChance = 0.1
	flintChance = 0.12
	resinChance = 0.2
)

// cellChance is a number from 0 to 1 that is a hash of a cell: what the block there
// holds by chance goes by it, so that it does not change.
func cellChance(x, y, z int) float64 {
	h := uint32(x)*0x9E3779B1 ^ uint32(y)*0x85EBCA77 ^ uint32(z)*0xC2B2AE3D
	h ^= h >> 15
	h *= 0x2C1B3C6D
	h ^= h >> 13
	return float64(h) / (1 << 32)
}

// DropAt is Drop for the block in a cell. Leaves now and then hold a fruit, and gravel
// a flint.
func (b Block) DropAt(x, y, z int) (Item, int) {
	chance := cellChance(x, y, z)
	switch {
	case (b == Leaves || b == PineNeedles || b == PaleLeaves || b == FanLeaves) && chance < fruitChance:
		return Fruit, 1
	case b == Gravel && chance < flintChance:
		return Flint, 1
	}
	return b.Drop()
}

// AlsoAt returns what the block in a cell leaves besides what DropAt says, and how many:
// a pine or a redwood log now and then holds a piece of resin.
func (b Block) AlsoAt(x, y, z int) (Item, int) {
	if (b == PineLog || b == RedwoodLog) && cellChance(x, y, z) < resinChance {
		return Resin, 1
	}
	return NoItem, 0
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

// Package tile names the block pictures in the tile strip (internal/assets/img/blocks.png),
// in the order they are drawn there. The generator, the loader and the block table all
// use these names, so the order is written down once.
package tile

// Size is the width and height of a tile in pixels.
const Size = 16

// ID is the place of a tile in the strip, counted from the left.
type ID uint8

const (
	TurfTop ID = iota
	TurfSide
	Dirt
	Stone
	Sand
	LogSide
	LogTop
	Leaves
	Planks
	Brick
	Glass
	Water
	Snow
	Deepstone

	Gravel
	Clay
	SandstoneTop
	SandstoneSide
	Cobble
	CoalOre
	FlintSeam
	IronOre
	CrystalOre
	PineLogSide
	PineLogTop
	PineNeedles
	PaleLogSide
	PaleLogTop
	PaleLeaves
	StumpPalmSide
	StumpPalmTop
	Ice
	Fern
	RedFlower
	YellowFlower
	Bush
	Horsetail
	DryBush
	Mushroom
	Torch
	KnappingTop
	KnappingSide
	HearthTop
	HearthSide
	PotTop
	PotSide
	FurBed

	TallFern
	PalmCrown
	FernTrunkSide
	FernTrunkTop
	Fronds
	FanLeaves
	RedwoodSide
	RedwoodTop
	Bone
	RedrockTop
	RedrockSide
	Fire
	HearthLitTop
	HearthLitSide
	CharredSide
	CharredTop

	// Pictures that are on no block.

	Berries
	// Cracks is laid over a block as it is broken. How red a pixel of it is says how far
	// the breaking must have gone for that pixel to show.
	Cracks
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
	Meat
	Fiber
	Cord
	Flint
	Hide
	HandAxe
	FlintPick
	FlintKnife
	DiggingStick
	Club
	Spear
	CookedMeat
	PalmSeeds
	RoastSeeds
	Resin
	UnlitTorch
	FireDrill
	HardSpear
	SoftPot

	// Count is the number of tiles in the strip.
	Count
)

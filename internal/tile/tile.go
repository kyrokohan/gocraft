// Package tile names the block pictures in the tile strip (internal/assets/img/blocks.png),
// in the order they are drawn there. The generator, the loader and the block table all
// use these names, so the order is written down once.
package tile

// Size is the width and height of a tile in pixels.
const Size = 16

// ID is the place of a tile in the strip, counted from the left.
type ID uint8

const (
	GrassTop ID = iota
	GrassSide
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
	CopperOre
	IronOre
	CrystalOre
	PineLogSide
	PineLogTop
	PineNeedles
	PaleLogSide
	PaleLogTop
	PaleLeaves
	CactusSide
	CactusTop
	Ice
	TallGrass
	RedFlower
	YellowFlower
	Bush
	Reed
	DryBush
	Mushroom

	// Count is the number of tiles in the strip.
	Count
)

// Package world is the grid of blocks: what each block is, the terrain a seed makes,
// rays through the grid, and saving it.
package world

import "github.com/kyrokohan/gocraft/internal/tile"

// Block is the kind of block in one cell of the world. The numbers are written into
// save files, so a kind keeps its number: add new ones at the end.
type Block uint8

const (
	Air Block = iota
	Grass
	Dirt
	Stone
	Sand
	Log
	Leaves
	Planks
	Brick
	Glass
	Water
	Snow
	Deepstone

	Gravel
	Clay
	Sandstone
	Cobble
	CoalOre
	CopperOre
	IronOre
	CrystalOre
	PineLog
	PineNeedles
	PaleLog
	PaleLeaves
	Cactus
	Ice
	TallGrass
	RedFlower
	YellowFlower
	Bush
	Reed
	DryBush
	Mushroom

	Torch

	blockCount
)

// Info is what the game needs to know about a kind of block.
type Info struct {
	// Name is what the player is shown.
	Name string

	// Top, Side and Bottom are the tiles drawn on its faces.
	Top, Side, Bottom tile.ID

	// Solid blocks stop the player. Air, water and plants do not.
	Solid bool

	// Plant blocks are not cubes: they are drawn as two pictures that cross in the
	// middle of the cell, and they need a block under them to stand on.
	Plant bool

	// Fixed blocks cannot be broken.
	Fixed bool

	// Shade is how many levels of light the block takes from light that passes through
	// it, from 0 to MaxLight: 0 for air and glass, a little for leaves and water, and
	// MaxLight for a block that lets none through.
	Shade uint8

	// Glow is the level of light the block gives off, from 0 to MaxLight.
	Glow uint8

	// Dig is how many seconds of digging by hand break the block. 0 is at once.
	Dig float64
}

var infos = [256]Info{
	Air:       {Name: "air"},
	Grass:     dig(solid("grass", tile.GrassTop, tile.GrassSide, tile.Dirt), 0.6),
	Dirt:      dig(cube("dirt", tile.Dirt), 0.5),
	Stone:     dig(cube("stone", tile.Stone), 2),
	Sand:      dig(cube("sand", tile.Sand), 0.5),
	Log:       dig(solid("log", tile.LogTop, tile.LogSide, tile.LogTop), 1.5),
	Leaves:    dig(leaves("leaves", tile.Leaves), 0.2),
	Planks:    dig(cube("planks", tile.Planks), 1.2),
	Brick:     dig(cube("brick", tile.Brick), 2),
	Glass:     {Name: "glass", Top: tile.Glass, Side: tile.Glass, Bottom: tile.Glass, Solid: true, Dig: 0.3},
	Water:     {Name: "water", Top: tile.Water, Side: tile.Water, Bottom: tile.Water, Shade: 1},
	Snow:      dig(cube("snow", tile.Snow), 0.3),
	Deepstone: fixed(cube("deepstone", tile.Deepstone)),

	Gravel:      dig(cube("gravel", tile.Gravel), 0.6),
	Clay:        dig(cube("clay", tile.Clay), 0.6),
	Sandstone:   dig(solid("sandstone", tile.SandstoneTop, tile.SandstoneSide, tile.SandstoneTop), 1.2),
	Cobble:      dig(cube("cobble", tile.Cobble), 2),
	CoalOre:     dig(cube("coal ore", tile.CoalOre), 2.5),
	CopperOre:   dig(cube("copper ore", tile.CopperOre), 3),
	IronOre:     dig(cube("iron ore", tile.IronOre), 3),
	CrystalOre:  dig(glowing(cube("crystal ore", tile.CrystalOre), 8), 3.5),
	PineLog:     dig(solid("pine log", tile.PineLogTop, tile.PineLogSide, tile.PineLogTop), 1.5),
	PineNeedles: dig(leaves("pine needles", tile.PineNeedles), 0.2),
	PaleLog:     dig(solid("pale log", tile.PaleLogTop, tile.PaleLogSide, tile.PaleLogTop), 1.5),
	PaleLeaves:  dig(leaves("pale leaves", tile.PaleLeaves), 0.2),
	Cactus:      dig(solid("cactus", tile.CactusTop, tile.CactusSide, tile.CactusTop), 0.4),
	Ice:         dig(cube("ice", tile.Ice), 0.5),

	TallGrass:    plant("tall grass", tile.TallGrass),
	RedFlower:    plant("red flower", tile.RedFlower),
	YellowFlower: plant("yellow flower", tile.YellowFlower),
	Bush:         plant("bush", tile.Bush),
	Reed:         plant("reed", tile.Reed),
	DryBush:      plant("dry bush", tile.DryBush),
	Mushroom:     plant("mushroom", tile.Mushroom),

	// A torch stands on the ground as a plant does, and is drawn as one.
	Torch: glowing(plant("torch", tile.Torch), 14),
}

// solid is a block that stops the player and lets no light through, with a tile for
// its top, its sides and its bottom.
func solid(name string, top, side, bottom tile.ID) Info {
	return Info{Name: name, Top: top, Side: side, Bottom: bottom, Solid: true, Shade: MaxLight}
}

// cube is a solid block with one tile on every face.
func cube(name string, t tile.ID) Info {
	return solid(name, t, t, t)
}

// leaves are a solid block that light comes through, a little dimmed.
func leaves(name string, t tile.ID) Info {
	info := cube(name, t)
	info.Shade = 1
	return info
}

func fixed(info Info) Info {
	info.Fixed = true
	return info
}

func glowing(info Info, level uint8) Info {
	info.Glow = level
	return info
}

func dig(info Info, seconds float64) Info {
	info.Dig = seconds
	return info
}

// plant is a plant with one picture.
func plant(name string, t tile.ID) Info {
	return Info{Name: name, Top: t, Side: t, Bottom: t, Plant: true}
}

// Info returns what is known about the kind of block. A number that is no kind of block
// is like air.
func (b Block) Info() Info {
	return infos[b]
}

// Known reports whether b is a kind of block this version of the game has.
func (b Block) Known() bool {
	return b < blockCount
}

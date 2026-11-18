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
}

var infos = [256]Info{
	Air:       {Name: "air"},
	Grass:     solid("grass", tile.GrassTop, tile.GrassSide, tile.Dirt),
	Dirt:      cube("dirt", tile.Dirt),
	Stone:     cube("stone", tile.Stone),
	Sand:      cube("sand", tile.Sand),
	Log:       solid("log", tile.LogTop, tile.LogSide, tile.LogTop),
	Leaves:    leaves("leaves", tile.Leaves),
	Planks:    cube("planks", tile.Planks),
	Brick:     cube("brick", tile.Brick),
	Glass:     {Name: "glass", Top: tile.Glass, Side: tile.Glass, Bottom: tile.Glass, Solid: true},
	Water:     {Name: "water", Top: tile.Water, Side: tile.Water, Bottom: tile.Water, Shade: 1},
	Snow:      cube("snow", tile.Snow),
	Deepstone: fixed(cube("deepstone", tile.Deepstone)),

	Gravel:      cube("gravel", tile.Gravel),
	Clay:        cube("clay", tile.Clay),
	Sandstone:   solid("sandstone", tile.SandstoneTop, tile.SandstoneSide, tile.SandstoneTop),
	Cobble:      cube("cobble", tile.Cobble),
	CoalOre:     cube("coal ore", tile.CoalOre),
	CopperOre:   cube("copper ore", tile.CopperOre),
	IronOre:     cube("iron ore", tile.IronOre),
	CrystalOre:  glowing(cube("crystal ore", tile.CrystalOre), 6),
	PineLog:     solid("pine log", tile.PineLogTop, tile.PineLogSide, tile.PineLogTop),
	PineNeedles: leaves("pine needles", tile.PineNeedles),
	PaleLog:     solid("pale log", tile.PaleLogTop, tile.PaleLogSide, tile.PaleLogTop),
	PaleLeaves:  leaves("pale leaves", tile.PaleLeaves),
	Cactus:      solid("cactus", tile.CactusTop, tile.CactusSide, tile.CactusTop),
	Ice:         cube("ice", tile.Ice),

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

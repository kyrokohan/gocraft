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
}

var infos = [256]Info{
	Air:       {Name: "air"},
	Grass:     {Name: "grass", Top: tile.GrassTop, Side: tile.GrassSide, Bottom: tile.Dirt, Solid: true},
	Dirt:      {Name: "dirt", Top: tile.Dirt, Side: tile.Dirt, Bottom: tile.Dirt, Solid: true},
	Stone:     {Name: "stone", Top: tile.Stone, Side: tile.Stone, Bottom: tile.Stone, Solid: true},
	Sand:      {Name: "sand", Top: tile.Sand, Side: tile.Sand, Bottom: tile.Sand, Solid: true},
	Log:       {Name: "log", Top: tile.LogTop, Side: tile.LogSide, Bottom: tile.LogTop, Solid: true},
	Leaves:    {Name: "leaves", Top: tile.Leaves, Side: tile.Leaves, Bottom: tile.Leaves, Solid: true},
	Planks:    {Name: "planks", Top: tile.Planks, Side: tile.Planks, Bottom: tile.Planks, Solid: true},
	Brick:     {Name: "brick", Top: tile.Brick, Side: tile.Brick, Bottom: tile.Brick, Solid: true},
	Glass:     {Name: "glass", Top: tile.Glass, Side: tile.Glass, Bottom: tile.Glass, Solid: true},
	Water:     {Name: "water", Top: tile.Water, Side: tile.Water, Bottom: tile.Water},
	Snow:      {Name: "snow", Top: tile.Snow, Side: tile.Snow, Bottom: tile.Snow, Solid: true},
	Deepstone: {Name: "deepstone", Top: tile.Deepstone, Side: tile.Deepstone, Bottom: tile.Deepstone, Solid: true, Fixed: true},

	Gravel:      cube("gravel", tile.Gravel),
	Clay:        cube("clay", tile.Clay),
	Sandstone:   {Name: "sandstone", Top: tile.SandstoneTop, Side: tile.SandstoneSide, Bottom: tile.SandstoneTop, Solid: true},
	Cobble:      cube("cobble", tile.Cobble),
	CoalOre:     cube("coal ore", tile.CoalOre),
	CopperOre:   cube("copper ore", tile.CopperOre),
	IronOre:     cube("iron ore", tile.IronOre),
	CrystalOre:  cube("crystal ore", tile.CrystalOre),
	PineLog:     {Name: "pine log", Top: tile.PineLogTop, Side: tile.PineLogSide, Bottom: tile.PineLogTop, Solid: true},
	PineNeedles: cube("pine needles", tile.PineNeedles),
	PaleLog:     {Name: "pale log", Top: tile.PaleLogTop, Side: tile.PaleLogSide, Bottom: tile.PaleLogTop, Solid: true},
	PaleLeaves:  cube("pale leaves", tile.PaleLeaves),
	Cactus:      {Name: "cactus", Top: tile.CactusTop, Side: tile.CactusSide, Bottom: tile.CactusTop, Solid: true},
	Ice:         cube("ice", tile.Ice),

	TallGrass:    plant("tall grass", tile.TallGrass),
	RedFlower:    plant("red flower", tile.RedFlower),
	YellowFlower: plant("yellow flower", tile.YellowFlower),
	Bush:         plant("bush", tile.Bush),
	Reed:         plant("reed", tile.Reed),
	DryBush:      plant("dry bush", tile.DryBush),
	Mushroom:     plant("mushroom", tile.Mushroom),
}

// cube is a solid block with one tile on every face.
func cube(name string, t tile.ID) Info {
	return Info{Name: name, Top: t, Side: t, Bottom: t, Solid: true}
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

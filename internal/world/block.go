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

	blockCount
)

// Info is what the game needs to know about a kind of block.
type Info struct {
	// Name is what the player is shown.
	Name string

	// Top, Side and Bottom are the tiles drawn on its faces.
	Top, Side, Bottom tile.ID

	// Solid blocks stop the player. Air and water do not.
	Solid bool

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

// Package world is the grid of blocks: what each block is, the terrain a seed makes,
// rays through the grid, and saving it.
package world

import "github.com/kyrokohan/skyspear/internal/tile"

// Block is the kind of block in one cell of the world. The numbers are written into
// save files, so a kind keeps its number: add new ones at the end.
type Block uint8

const (
	Air Block = iota
	Turf
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
	StumpPalm
	Ice
	Fern
	RedFlower
	YellowFlower
	Bush
	Horsetail
	DryBush
	Mushroom

	Torch
	Workbench
	Kiln
	Crate
	Bedroll

	TallFern
	PalmCrown
	FernTrunk
	Fronds
	FanLeaves
	RedwoodLog
	Bone
	Redrock

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

	// Mat blocks are not cubes either: they lie flat on the block under them, drawn as
	// their Top a little above the floor of the cell, and they are walked over.
	Mat bool

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

	// Tool is the kind of tool that digs the block faster, and Needs the tier of that
	// tool the block needs to leave anything when it breaks: 0 for none.
	Tool  Tool
	Needs int
}

var infos = [256]Info{
	Air:       {Name: "air"},
	Turf:      byShovel(solid("mossy earth", tile.TurfTop, tile.TurfSide, tile.Dirt), 0.6),
	Dirt:      byShovel(cube("dirt", tile.Dirt), 0.5),
	Stone:     byPick(cube("stone", tile.Stone), 2, TierWood),
	Sand:      byShovel(cube("sand", tile.Sand), 0.5),
	Log:       byAxe(solid("log", tile.LogTop, tile.LogSide, tile.LogTop), 1.5),
	Leaves:    dig(leaves("leaves", tile.Leaves), 0.2),
	Planks:    byAxe(cube("planks", tile.Planks), 1.2),
	Brick:     byPick(cube("brick", tile.Brick), 2, TierWood),
	Glass:     byPick(Info{Name: "glass", Top: tile.Glass, Side: tile.Glass, Bottom: tile.Glass, Solid: true}, 0.3, 0),
	Water:     {Name: "water", Top: tile.Water, Side: tile.Water, Bottom: tile.Water, Shade: 1},
	Snow:      byShovel(cube("snow", tile.Snow), 0.3),
	Deepstone: fixed(cube("deepstone", tile.Deepstone)),

	Gravel:      byShovel(cube("gravel", tile.Gravel), 0.6),
	Clay:        byShovel(cube("clay", tile.Clay), 0.6),
	Sandstone:   byPick(solid("sandstone", tile.SandstoneTop, tile.SandstoneSide, tile.SandstoneTop), 1.2, TierWood),
	Cobble:      byPick(cube("cobble", tile.Cobble), 2, TierWood),
	CoalOre:     byPick(cube("coal ore", tile.CoalOre), 2.5, TierWood),
	CopperOre:   byPick(cube("copper ore", tile.CopperOre), 3, TierStone),
	IronOre:     byPick(cube("iron ore", tile.IronOre), 3, TierStone),
	CrystalOre:  byPick(glowing(cube("crystal ore", tile.CrystalOre), 8), 3.5, TierCopper),
	PineLog:     byAxe(solid("pine log", tile.PineLogTop, tile.PineLogSide, tile.PineLogTop), 1.5),
	PineNeedles: dig(leaves("pine needles", tile.PineNeedles), 0.2),
	PaleLog:     byAxe(solid("pale log", tile.PaleLogTop, tile.PaleLogSide, tile.PaleLogTop), 1.5),
	PaleLeaves:  dig(leaves("pale leaves", tile.PaleLeaves), 0.2),
	StumpPalm:   byAxe(solid("stump palm", tile.StumpPalmTop, tile.StumpPalmSide, tile.StumpPalmTop), 0.4),
	Ice:         byPick(cube("ice", tile.Ice), 0.5, 0),

	Fern:         plant("fern", tile.Fern),
	RedFlower:    plant("red flower", tile.RedFlower),
	YellowFlower: plant("yellow flower", tile.YellowFlower),
	Bush:         plant("bush", tile.Bush),
	Horsetail:    plant("horsetail", tile.Horsetail),
	DryBush:      plant("dry bush", tile.DryBush),
	Mushroom:     plant("mushroom", tile.Mushroom),

	// A torch stands on the ground as a plant does, and is drawn as one.
	Torch: glowing(plant("torch", tile.Torch), 14),

	// Standing near a workbench or a kiln lets the player make more.
	Workbench: byAxe(solid("workbench", tile.WorkbenchTop, tile.WorkbenchSide, tile.Planks), 1.5),
	Kiln:      byPick(solid("kiln", tile.KilnTop, tile.KilnSide, tile.Cobble), 2.5, TierWood),

	// A crate keeps things, and a bedroll is where the player sleeps and wakes.
	Crate:   byAxe(solid("crate", tile.CrateTop, tile.CrateSide, tile.CrateTop), 1.2),
	Bedroll: dig(Info{Name: "bedroll", Top: tile.Bedroll, Side: tile.Bedroll, Bottom: tile.Bedroll, Mat: true}, 0.3),

	// What grows in the old lands. A tall fern fills its cell, and a stump palm's crown
	// stands on its trunk as a plant does on the ground.
	TallFern:   plant("tall fern", tile.TallFern),
	PalmCrown:  plant("stump palm crown", tile.PalmCrown),
	FernTrunk:  byAxe(solid("tree fern trunk", tile.FernTrunkTop, tile.FernTrunkSide, tile.FernTrunkTop), 1.2),
	Fronds:     dig(leaves("fronds", tile.Fronds), 0.2),
	FanLeaves:  dig(leaves("fan leaves", tile.FanLeaves), 0.2),
	RedwoodLog: byAxe(solid("redwood log", tile.RedwoodTop, tile.RedwoodSide, tile.RedwoodTop), 2),

	// Bone is what the great skeletons are made of, and red rock the ground of the
	// badlands they lie in.
	Bone:    byPick(cube("bone", tile.Bone), 1.5, 0),
	Redrock: byPick(solid("red rock", tile.RedrockTop, tile.RedrockSide, tile.RedrockTop), 1.6, TierWood),
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

// byPick, byAxe and byShovel are blocks dug faster with that tool. A block dug with a pick
// may need one of a tier to leave anything.
func byPick(info Info, seconds float64, needs int) Info {
	info.Dig, info.Tool, info.Needs = seconds, Pick, needs
	return info
}

func byAxe(info Info, seconds float64) Info {
	info.Dig, info.Tool = seconds, Axe
	return info
}

func byShovel(info Info, seconds float64) Info {
	info.Dig, info.Tool = seconds, Shovel
	return info
}

// plant is a plant with one picture.
func plant(name string, t tile.ID) Info {
	return Info{Name: name, Top: t, Side: t, Bottom: t, Plant: true}
}

// Stands reports whether the block needs one under it, and goes when that one does: a
// plant or a mat.
func (i Info) Stands() bool {
	return i.Plant || i.Mat
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

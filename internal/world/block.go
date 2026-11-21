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
	FlintSeam
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
	KnappingStone
	Hearth
	ClayPot
	FurBed

	TallFern
	PalmCrown
	FernTrunk
	Fronds
	FanLeaves
	RedwoodLog
	Bone
	Redrock

	Fire
	HearthLit
	CharredLog

	PitCover

	Ash
	Basalt
	Lava
	ObsidianRock
	FirestoneRock

	SinterRust
	SinterGold
	SinterGreen

	HotCore
	SkyIronCore

	Peat
	ScaleTrunk
	ScaleCrown

	Tar

	OchreSeam
	ChalkSeam

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
	Stone:     byPick(cube("stone", tile.Stone), 5, 0), // by hand it breaks to rubble, slowly
	Sand:      byShovel(cube("sand", tile.Sand), 0.5),
	Log:       byAxe(solid("log", tile.LogTop, tile.LogSide, tile.LogTop), 1.5),
	Leaves:    dig(leaves("leaves", tile.Leaves), 0.2),
	Planks:    byAxe(cube("planks", tile.Planks), 1.2),
	Brick:     byPick(cube("brick", tile.Brick), 2, 0),
	Glass:     byPick(Info{Name: "glass", Top: tile.Glass, Side: tile.Glass, Bottom: tile.Glass, Solid: true}, 0.3, 0),
	Water:     {Name: "water", Top: tile.Water, Side: tile.Water, Bottom: tile.Water, Shade: 1},
	Snow:      byShovel(cube("snow", tile.Snow), 0.3),
	Deepstone: fixed(cube("deepstone", tile.Deepstone)),

	Gravel:      byShovel(cube("gravel", tile.Gravel), 0.6),
	Clay:        byShovel(cube("clay", tile.Clay), 0.6),
	Sandstone:   byPick(solid("sandstone", tile.SandstoneTop, tile.SandstoneSide, tile.SandstoneTop), 1.2, TierWood),
	Cobble:      byPick(cube("cobble", tile.Cobble), 3, 0),
	CoalOre:     byPick(cube("coal ore", tile.CoalOre), 2.5, TierFlint),
	FlintSeam:   byPick(cube("flint seam", tile.FlintSeam), 3, 0),
	IronOre:     byPick(cube("iron ore", tile.IronOre), 3, TierFlint),
	CrystalOre:  byPick(glowing(cube("crystal ore", tile.CrystalOre), 8), 3.5, TierObsidian),
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

	// A torch stands on the ground as a plant does, and is drawn as one. Set down, it
	// burns for ever.
	Torch: glowing(plant("torch", tile.Torch), 14),

	// Standing near a knapping stone lets the player make more. A hearth is a ring of
	// stones that cooks while it burns: cold, it is this block, and lit, HearthLit.
	KnappingStone: byPick(solid("knapping stone", tile.KnappingTop, tile.KnappingSide, tile.Stone), 2, 0),
	Hearth:        byPick(solid("hearth", tile.HearthTop, tile.HearthSide, tile.Cobble), 2.5, 0),

	// A clay pot keeps things, and a fur bed is where the player sleeps and wakes.
	ClayPot: dig(solid("clay pot", tile.PotTop, tile.PotSide, tile.PotTop), 0.5),
	FurBed:  dig(Info{Name: "fur bed", Top: tile.FurBed, Side: tile.FurBed, Bottom: tile.FurBed, Mat: true}, 0.3),

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

	// Fire burns for ever where lightning struck, on the charred wood that is left of
	// the tree. It stands on what it burns as a plant stands on the ground, and is
	// walked into, which hurts. No hand breaks it: it goes when what it burns does.
	Fire:       fixed(glowing(plant("fire", tile.Fire), MaxLight)),
	HearthLit:  glowing(byPick(solid("hearth", tile.HearthLitTop, tile.HearthLitSide, tile.Cobble), 2.5, 0), 14),
	CharredLog: byAxe(solid("charred log", tile.CharredTop, tile.CharredSide, tile.CharredTop), 1),

	// A pit cover is branches and ferns laid over a hole, to look like the ground. It
	// holds a person and a small beast, and a heavy beast breaks through it.
	PitCover: dig(solid("pit cover", tile.PitCoverTop, tile.PitCoverSide, tile.PitCoverSide), 0.4),

	// What a fire mountain is made of. Ash is soft, and dark rock as hard as stone. Lava
	// is no solid thing: a body sinks in it as in water, and burns. It glows as fire
	// does, lets light through, and cannot be broken or taken.
	Ash:    byShovel(cube("ash", tile.Ash), 0.4),
	Basalt: byPick(cube("dark rock", tile.Basalt), 5, 0),
	Lava:   fixed(glowing(Info{Name: "lava", Top: tile.Lava, Side: tile.Lava, Bottom: tile.Lava}, MaxLight)),

	// What only a fire mountain has. Obsidian is glass the mountain made, in veins on its
	// cone: a flint pick breaks it out. Firestone is a yellow stone by the crater, which
	// strikes fire from a flint: it breaks by hand, as a flint seam does.
	ObsidianRock:  byPick(cube("obsidian", tile.ObsidianRock), 4, TierFlint),
	FirestoneRock: byPick(cube("firestone", tile.FirestoneRock), 3, 0),

	// The stone a hot spring leaves round itself, in rings: rust nearest the water,
	// then gold, then the green of what grows in the warm.
	SinterRust:  byPick(cube("rust sinter", tile.SinterRust), 2, 0),
	SinterGold:  byPick(cube("gold sinter", tile.SinterGold), 2, 0),
	SinterGreen: byPick(cube("green sinter", tile.SinterGreen), 2, 0),

	// What a fallen star leaves at the bottom of its crater: a core of sky iron. At first
	// it glows, burns what stands on it, and no hand or tool breaks it. Then it goes
	// dark, and a flint pick breaks it out, slowly.
	HotCore:     fixed(glowing(cube("glowing sky iron", tile.HotCore), 12)),
	SkyIronCore: byPick(cube("sky iron core", tile.SkyIronCore), 12, TierFlint),

	// What the coal swamp is made of. Peat is its ground, black and wet: it burns. A
	// scale tree is a bare trunk, all diamonds where old leaves grew, with a crown of
	// long leaves at the top of it.
	Peat:       byShovel(cube("peat", tile.Peat), 0.7),
	ScaleTrunk: byAxe(solid("scale tree trunk", tile.ScaleTrunkTop, tile.ScaleTrunkSide, tile.ScaleTrunkTop), 1.6),
	ScaleCrown: dig(leaves("scale tree crown", tile.ScaleCrown), 0.2),

	// Tar is no solid thing: what walks into it sinks, slowly, and is held. It is black,
	// lets no light through, and comes away in the hand.
	Tar: dig(Info{Name: "tar", Top: tile.Tar, Side: tile.Tar, Bottom: tile.Tar, Shade: MaxLight}, 1.2),

	// What paint is ground from. Ochre is a red earth that lies in beds in the rock of
	// the badlands, and chalk a soft white rock with flint in it. Both break by hand.
	OchreSeam: byPick(cube("ochre", tile.OchreSeam), 1.2, 0),
	ChalkSeam: byPick(cube("chalk", tile.ChalkSeam), 1.0, 0),
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

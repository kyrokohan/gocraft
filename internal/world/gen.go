package world

import "math"

// Layers of the terrain, counted as the layer of a column's top block.
const (
	rockLine = 76  // from here up the ground is bare stone
	snowLine = 86  // and from here up it is snow
	maxTop   = 112 // no ground is higher, which leaves room for a tree
)

// treeSquare is the side of the squares of a grid over the ground, each of which holds
// one tree or none, so that no two trees stand in each other.
const treeSquare = 7

// treeReach is how far from its foot the widest tree reaches, which is a redwood's
// crown, and so how far outside itself a chunk looks for trees that reach into it.
const treeReach = 5

// highStand is how high the country must stand, of 1, to be highland.
const highStand = 0.57

// Biome is a kind of country: what the ground is made of and what grows on it.
type Biome uint8

// The lands are old ones. There is no grass in them: the ground is moss, and ferns grow
// where grass would.
const (
	Sea      Biome = iota // ground under the water
	Shore                 // sand at the water's edge
	Prairie               // moss and ferns, with horsetails, stump palms and few trees
	Jungle                // tree ferns and tall ferns under a high roof of leaves
	Badlands              // dry red rock over sandstone, where the great skeletons lie
	Tundra                // snow, with pines
	Crags                 // bare stone, high up, with snow higher still
	Highland              // high country of umbrella pines, fan-leaf trees and redwoods
	Ashlands              // ash and dark rock round a fire mountain, where nothing grows
	Swamp                 // the coal swamp: peat and black pools at the water's edge, under scale trees

	biomeCount
)

// lattice returns a number from 0 up to 1 fixed by a point of a grid and a seed.
func lattice(x, z int, seed uint32) float64 {
	h := uint32(x)*0x9E3779B1 ^ uint32(z)*0x85EBCA77 ^ seed*0xC2B2AE3D
	h ^= h >> 15
	h *= 0x2C1B3C6D
	h ^= h >> 12
	h *= 0x297A2D39
	h ^= h >> 15
	return float64(h) / (1 << 32)
}

// smooth returns noise from 0 to 1 that changes gently from one whole number to the next.
func smooth(x, z float64, seed uint32) float64 {
	x0, z0 := math.Floor(x), math.Floor(z)
	tx, tz := x-x0, z-z0
	tx, tz = tx*tx*(3-2*tx), tz*tz*(3-2*tz)
	i, j := int(x0), int(z0)
	near := lattice(i, j, seed)*(1-tx) + lattice(i+1, j, seed)*tx
	far := lattice(i, j+1, seed)*(1-tx) + lattice(i+1, j+1, seed)*tx
	return near*(1-tz) + far*tz
}

// hills returns noise from 0 to 1 with detail at four sizes, each half the size and half
// the strength of the one before.
func hills(x, z float64, seed uint32) float64 {
	sum, strength, total := 0.0, 1.0, 0.0
	for octave := range uint32(4) {
		sum += strength * smooth(x, z, seed+octave*101)
		total += strength
		x, z, strength = x*2, z*2, strength/2
	}
	return sum / total
}

// ramp returns 0 below lo, 1 above hi, and a smooth rise between.
func ramp(lo, hi, v float64) float64 {
	t := min(max((v-lo)/(hi-lo), 0), 1)
	return t * t * (3 - 2*t)
}

// floorDiv divides and rounds down, where Go rounds toward 0: the square of a grid that
// a coordinate is in, on either side of 0.
func floorDiv(a, b int) int {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

// Generate returns the world a seed makes, with the chunks round its origin loaded.
// The same seed always makes the same world.
func Generate(seed int64) *World {
	return newWorld(seed, Terrain(seed), "", 0, 0)
}

// Terrain returns the Generator of the world a seed makes: seas and land, flat country
// and high, warm and cold, wet and dry, each with what grows there.
func Terrain(seed int64) Generator {
	s := uint32(seed) ^ uint32(seed>>32)
	below := new(under) // kept between chunks: it is big, and a chunk is made many times a second
	return func(cx, cz int, blocks []Block) {
		terrain{seed: s, cx: cx, cz: cz, blocks: blocks, below: below}.build()
	}
}

// land is what the seed says of one column of the world: how high its ground is and
// what country it is in. It is worked out from the seed and the place alone.
type land struct {
	top   int // the layer of the top block of the ground
	biome Biome
	cold  bool // cold enough for ice to lie on water

	// cone is true on the cone of a fire mountain, and lava is the layer the lava of
	// its lake stands to, in its crater: 0 anywhere else. crater is how far the column
	// is from the middle of the mountain, on its cone.
	cone   bool
	lava   int
	crater float64
}

// The coal swamp is low wet country that the water has soaked flat: where the land is
// wetter than swampWet and stands lower than swampStand, by the slow noises that say so,
// the ground is drawn down to a step or two over the sea. It is soaked the more the
// farther inside both, so that the country round it comes down to it and does not end
// in a cliff.
const (
	swampWet   = 0.58
	swampStand = 0.54
	swampEdge  = 0.03 // how far inside the wet, and the low, the ground is all the way down
)

// landAt returns the land of the column at x, z.
func landAt(x, z int, seed uint32) land {
	fx, fz := float64(x), float64(z)

	// Four slow noises: how far above the sea the country stands, how rough it is,
	// how warm and how wet. And one quick one, for the lie of the ground itself.
	stand := hills(fx/300, fz/300, seed+1)
	rough := ramp(0.44, 0.72, hills(fx/170, fz/170, seed+2))
	warmth := hills(fx/380, fz/380, seed+3)
	wetness := hills(fx/330, fz/330, seed+4)
	warm := warmth + (lattice(x, z, seed+6)-0.5)*0.02
	wet := wetness + (lattice(x, z, seed+7)-0.5)*0.02
	lie := hills(fx/52, fz/52, seed+5) - 0.5

	dry := warm > 0.60 && wet < 0.46
	swell := 6 + 50*rough
	if dry {
		swell = 4 + 14*rough // dunes, not mountains
	}
	h := float64(SeaLevel) + (stand-0.44)*90 + lie*swell
	if dry {
		// The badlands rise in mesas: flat tops in two steps, with steep sides that
		// show the beds of the rock. They are lower toward the edge of the dry country.
		inside := ramp(0.60, 0.625, warmth) * (1 - ramp(0.435, 0.46, wetness))
		mesa := smooth(fx/36, fz/36, seed+8)
		h += inside * (9*ramp(0.58, 0.62, mesa) + 7*ramp(0.72, 0.75, mesa))
	}
	soak := 0.0
	if !dry && warm >= 0.38 && h >= SeaLevel {
		soak = ramp(swampWet, swampWet+swampEdge, wetness) * (1 - ramp(swampStand-swampEdge, swampStand, stand))
		level := float64(SeaLevel) + 1.5 + 0.9*(smooth(fx/11, fz/11, seed+43)-0.5)
		h = level + (h-level)*(1-soak)
	}
	// A fire mountain stands on whatever land it is in.
	m, far, ashen := mountainNear(x, z, seed)
	if ashen {
		h = m.shape(h, far)
	}
	l := land{top: min(max(int(math.Floor(h)), 2), maxTop), cold: warm < 0.38}
	if ashen {
		l.cold, l.cone, l.crater = false, far < mountFoot, far
		if far < craterWide && l.top < m.lava() {
			l.lava = m.lava()
		}
	}

	switch {
	case l.top < SeaLevel:
		l.biome = Sea
	case ashen && (l.cone || l.top > SeaLevel+1):
		l.biome = Ashlands // to the top of its cone, however high
	case soak > 0.8 && l.top <= SeaLevel+2:
		// The coal swamp: all but flat, a step or two over the sea, and pitted with
		// pools that stand to the level of the sea.
		l.biome = Swamp
		switch pool := smooth(fx/7, fz/7, seed+41); {
		case pool > 0.74:
			l.top = SeaLevel - 2
		case pool > 0.6:
			l.top = SeaLevel - 1
		}
	case l.top <= SeaLevel+1:
		l.biome = Shore
	case dry:
		l.biome = Badlands // red rock to the top of its mesas, however high
	case l.top >= rockLine:
		l.biome = Crags
	case l.cold:
		l.biome = Tundra
	case stand > highStand:
		l.biome = Highland
	case wet > 0.55:
		l.biome = Jungle
	default:
		l.biome = Prairie
	}
	return l
}

// BiomeAt returns the kind of country the column at x, z is in, in the world of a seed.
func BiomeAt(seed int64, x, z int) Biome {
	return landAt(x, z, uint32(seed)^uint32(seed>>32)).biome
}

// terrain is one chunk being made. Everything it puts down is worked out from the seed
// and the place alone, never from what is in a chunk beside it, so a chunk comes out
// the same whenever and in whatever order it is made, and a tree that stands on the
// line between two chunks is built half by each.
type terrain struct {
	seed   uint32
	cx, cz int
	blocks []Block
	below  *under // the caves and ores under this chunk
}

// put sets the block at a place in the world, if the place is in this chunk. Leaves
// and the like only take the place of air.
func (t terrain) put(x, y, z int, b Block, onlyAir bool) {
	lx, lz := x-t.cx*ChunkSize, z-t.cz*ChunkSize
	if uint(lx) >= ChunkSize || uint(lz) >= ChunkSize || uint(y) >= Height {
		return
	}
	at := (y*ChunkSize+lz)*ChunkSize + lx
	if onlyAir && t.blocks[at] != Air {
		return
	}
	t.blocks[at] = b
}

func (t terrain) build() {
	x0, z0 := t.cx*ChunkSize, t.cz*ChunkSize

	// The land of every column first: how high the highest is says how much of what
	// lies under the ground has to be worked out.
	var lands [ChunkSize * ChunkSize]land
	top := 0
	for i := range lands {
		lands[i] = landAt(x0+i%ChunkSize, z0+i/ChunkSize, t.seed)
		top = max(top, lands[i].top)
	}
	t.below.fill(t.cx, t.cz, top, t.seed)

	for i, l := range lands {
		x, z := x0+i%ChunkSize, z0+i/ChunkSize
		t.column(x, z, l)
		t.grow(x, z, l)
	}

	t.landmarks()
	t.mountains()

	// The trees whose leaves can reach into this chunk: those rooted in it or within
	// treeReach of it. A tree takes the place of whatever small thing grew at its foot.
	for gz := floorDiv(z0-treeReach, treeSquare); gz <= floorDiv(z0+ChunkSize-1+treeReach, treeSquare); gz++ {
		for gx := floorDiv(x0-treeReach, treeSquare); gx <= floorDiv(x0+ChunkSize-1+treeReach, treeSquare); gx++ {
			x := gx*treeSquare + 2 + int(lattice(gx, gz, t.seed+7002)*3)
			z := gz*treeSquare + 2 + int(lattice(gx, gz, t.seed+7003)*3)
			t.plant(x, z, landAt(x, z, t.seed), lattice(gx, gz, t.seed+7001), lattice(gx, gz, t.seed+7004))
		}
	}
}

// column builds the ground of one column, and the water over it.
func (t terrain) column(x, z int, l land) {
	// What the top block is, what the few blocks under it are, and how many of those.
	surface, under, deep := Turf, Dirt, 3
	switch l.biome {
	case Sea:
		surface, under = Sand, Sand
		switch patch := smooth(float64(x)/9, float64(z)/9, t.seed+31); {
		case patch > 0.68:
			surface, under = Gravel, Gravel
		case patch < 0.24:
			surface, under = Clay, Clay
		}
	case Shore:
		surface, under = Sand, Sand
	case Badlands:
		surface, under, deep = Redrock, Sandstone, 6
		if smooth(float64(x)/11, float64(z)/11, t.seed+32) > 0.68 {
			surface = Sand // a dry wash
		}
	case Tundra:
		surface = Snow
	case Swamp:
		surface, under = Peat, Peat
	case Crags:
		surface, under = Stone, Stone
		if l.top >= snowLine {
			surface = Snow
		}
	case Ashlands:
		// Ash lies on the flat ground and in drifts on the cone, whose rock is bare
		// where it is steep.
		surface, under, deep = Ash, Basalt, 5
		if l.cone && smooth(float64(x)/7, float64(z)/7, t.seed+33) < 0.62 {
			surface = Basalt
		}
		// What only the mountain has. Obsidian runs in veins on the cone, more of them
		// toward the top. Firestone lies in a ring at the rim of the crater and down its
		// inner wall, in patches.
		vein := smooth(float64(x)/4, float64(z)/4, t.seed+34)
		switch {
		case !l.cone || l.lava > 0:
		case math.Abs(l.crater-craterWide) < firestoneRing && vein < 0.42:
			surface = FirestoneRock
		case vein > 0.84-0.1*ramp(mountFoot, craterWide, l.crater):
			surface = ObsidianRock
		}
	}

	lx, lz := x-t.cx*ChunkSize, z-t.cz*ChunkSize
	t.put(x, 0, z, Deepstone, false)
	for y := 1; y <= l.top; y++ {
		if l.lava == 0 && t.below.hole(lx, y, lz, l) {
			continue // a cave: but there is none under the lake of a crater
		}
		b := Stone
		switch {
		case y == l.top:
			b = surface
		case l.biome == Badlands && y >= l.top-2:
			b = surface // red rock, or the sand of a wash, over the sandstone
		case l.biome == Swamp && y == l.top-2 && smooth(float64(x)/5, float64(z)/5, t.seed+42) > 0.62:
			b = CoalOre // a seam of coal lies near the top of the peat
		case y >= l.top-deep:
			b = under
		default:
			b = t.below.ore(x, y, z, lx, lz, t.seed)
		}
		t.put(x, y, z, b, false)
	}

	for y := l.top + 1; y <= SeaLevel; y++ {
		t.put(x, y, z, Water, false)
	}
	for y := l.top + 1; y <= l.lava; y++ {
		t.put(x, y, z, Lava, false)
	}
	if l.cold && l.top < SeaLevel {
		t.put(x, SeaLevel, z, Ice, false)
	}
}

// grow puts the small plant, if any, that stands on a column.
func (t terrain) grow(x, z int, l land) {
	if t.below.hole(x-t.cx*ChunkSize, l.top, z-t.cz*ChunkSize, l) {
		return // the ground here is the mouth of a cave
	}
	chance := lattice(x, z, t.seed+8001)
	b, tall := Air, 1
	switch l.biome {
	case Prairie:
		// Tall ferns gather in thickets and horsetails in stands, with low ferns
		// everywhere between them.
		switch patch := smooth(float64(x)/14, float64(z)/14, t.seed+8003); {
		case patch > 0.70 && chance < 0.55:
			b = TallFern
		case patch < 0.24 && chance < 0.30:
			b, tall = Horsetail, 2+int(chance*10)%2
		case chance < 0.16:
			b = Fern
		case chance < 0.175:
			b = TallFern
		case chance < 0.185:
			b = Bush
		}
	case Jungle:
		switch {
		case chance < 0.22:
			b = Fern
		case chance < 0.36:
			b = TallFern
		case chance < 0.385:
			b = Bush
		case chance < 0.40:
			// The first flowers. Those of one color grow together.
			b = RedFlower
			if smooth(float64(x)/18, float64(z)/18, t.seed+8002) < 0.5 {
				b = YellowFlower
			}
		case chance < 0.41:
			b = Mushroom
		}
	case Highland:
		switch {
		case chance < 0.08:
			b = Fern
		case chance < 0.10:
			b = Bush
		case chance < 0.11:
			b = Mushroom
		}
	case Badlands:
		if chance < 0.012 {
			b = DryBush
		}
	case Tundra:
		if chance < 0.07 {
			b = DryBush // a tussock of dry grass, through the snow
		}
	case Swamp:
		// Giant horsetails in stands, and ferns between them: but nothing in a pool.
		switch {
		case l.top < SeaLevel:
		case chance < 0.14:
			b, tall = Horsetail, 2+int(chance*100)%2
		case chance < 0.22:
			b = TallFern
		case chance < 0.3:
			b = Fern
		case chance < 0.32:
			b = Mushroom
		}
	case Ashlands:
		// What is left of the trees that stood here: a charred trunk, now and then, on
		// the flat ground round the cone.
		if chance < 0.012 && !l.cone {
			b, tall = CharredLog, 1+int(chance*1000)%3
		}
	case Shore:
		// Horsetails stand at the water's edge: on sand a step above the water, beside
		// a column that is under it.
		if chance < 0.3 && l.top == SeaLevel+1 && t.byWater(x, z) {
			b, tall = Horsetail, 1+int(chance*10)%2
		}
	}
	for i := range tall {
		if b != Air {
			t.put(x, l.top+1+i, z, b, true)
		}
	}
}

// byWater reports whether a column beside the one at x, z is under water.
func (t terrain) byWater(x, z int) bool {
	for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		if n := landAt(x+d[0], z+d[1], t.seed); n.top < SeaLevel && !n.cold {
			return true
		}
	}
	return false
}

// plant stands the tree, if any, that grows on the column at x, z. chance says whether
// one does and kind what it is like, each a number from 0 to 1 of the tree's own.
func (t terrain) plant(x, z int, l land, chance, kind float64) {
	// The tree may stand in the chunk beside this one, so the ground at its foot is
	// asked about by its place in the world, not by this chunk's grid.
	if holeAt(x, l.top, z, l, t.seed) {
		return // the ground here is the mouth of a cave
	}
	y := l.top + 1
	switch l.biome {
	case Jungle:
		switch {
		case chance < 0.45:
			t.treeFern(x, y, z, 3+int(kind*4))
		case chance < 0.62:
			t.broadTree(x, y, z, 6+int(kind*4)) // the roof of the jungle
		case chance < 0.72:
			t.stumpPalm(x, y, z, 1+int(kind*2))
		}
	case Highland:
		switch {
		case chance < 0.40:
			t.umbrellaPine(x, y, z, 9+int(kind*6))
		case chance < 0.52:
			t.fanTree(x, y, z, 5+int(kind*3))
		case chance < 0.56:
			t.redwood(x, y, z, 22+int(kind*11))
		}
	case Prairie:
		switch {
		case chance < 0.08:
			t.stumpPalm(x, y, z, 1+int(kind*2))
		case chance < 0.11:
			t.fanTree(x, y, z, 5+int(kind*3))
		}
	case Tundra:
		// The trees of the steppe are few and small: the wind keeps them so.
		if chance < 0.14 {
			t.pine(x, y, z, 3+int(kind*2))
		}
	case Swamp:
		if chance < 0.5 && l.top >= SeaLevel {
			t.scaleTree(x, y, z, 8+int(kind*5))
		}
	}
}

// crown puts one layer of a tree's leaves round its trunk: a square reach blocks from
// the trunk each way, with its corners there or not by chance, or never.
func (t terrain) crown(x, y, z, reach int, leaves Block, corners bool) {
	for dz := -reach; dz <= reach; dz++ {
		for dx := -reach; dx <= reach; dx++ {
			if abs(dx) == reach && abs(dz) == reach && reach > 0 &&
				(!corners || lattice(x+dx+y*31, z+dz, t.seed+7005) < 0.5) {
				continue
			}
			t.put(x+dx, y, z+dz, leaves, true)
		}
	}
}

// disc puts one flat round layer of a tree's leaves, reach blocks from its trunk.
func (t terrain) disc(x, y, z, reach int, leaves Block) {
	for dz := -reach; dz <= reach; dz++ {
		for dx := -reach; dx <= reach; dx++ {
			if dx*dx+dz*dz <= reach*reach+reach {
				t.put(x+dx, y, z+dz, leaves, true)
			}
		}
	}
}

// trunk stands n logs one on another, the lowest at (x, y, z).
func (t terrain) trunk(x, y, z, n int, log Block) {
	for i := range n {
		t.put(x, y+i, z, log, false)
	}
}

// broadTree stands a broad-leaved tree with its lowest log at (x, y, z): a trunk under
// a wide round crown.
func (t terrain) broadTree(x, y, z, trunk int) {
	t.trunk(x, y, z, trunk, Log)
	top := y + trunk
	t.crown(x, top-2, z, 2, Leaves, true)
	t.crown(x, top-1, z, 2, Leaves, true)
	t.crown(x, top, z, 1, Leaves, true)
	t.crown(x, top+1, z, 1, Leaves, false)
}

// pine stands a pine: a tall trunk with rings of needles that narrow to a point.
func (t terrain) pine(x, y, z, trunk int) {
	t.trunk(x, y, z, trunk, PineLog)
	top := y + trunk
	t.crown(x, top, z, 0, PineNeedles, false) // the tip
	for down, reach := range []int{1, 2, 1, 2, 2} {
		if at := top - 1 - down; at >= y+2 { // the foot of the trunk is bare
			t.crown(x, at, z, reach, PineNeedles, false)
		}
	}
}

// scaleTree stands a scale tree: a bare trunk with a small crown of long leaves at the
// top of it, and nothing below.
func (t terrain) scaleTree(x, y, z, trunk int) {
	t.trunk(x, y, z, trunk, ScaleTrunk)
	top := y + trunk
	t.disc(x, top-1, z, 2, ScaleCrown)
	t.crown(x, top, z, 1, ScaleCrown, false)
}

// umbrellaPine stands an umbrella pine: a tall bare trunk that spreads its needles flat
// and wide at the top, with one more ring of them lower down on a tall one.
func (t terrain) umbrellaPine(x, y, z, trunk int) {
	t.trunk(x, y, z, trunk, PineLog)
	top := y + trunk
	t.disc(x, top, z, 1, PineNeedles)
	t.disc(x, top-1, z, 3, PineNeedles)
	t.disc(x, top-2, z, 2, PineNeedles)
	if trunk >= 11 {
		t.disc(x, top-5, z, 2, PineNeedles)
	}
}

// fanTree stands a fan-leaf tree: a pale trunk in a tall crown of gold leaves, shaped
// like an egg.
func (t terrain) fanTree(x, y, z, trunk int) {
	t.trunk(x, y, z, trunk, PaleLog)
	top := y + trunk
	for i, reach := range []int{1, 2, 2, 2, 1} {
		t.crown(x, top-3+i, z, reach, FanLeaves, true)
	}
	t.crown(x, top+2, z, 0, FanLeaves, false)
}

// redwood stands a giant: a trunk two blocks thick, bare for more than half its height,
// under a crown that narrows to a point in steps. Its foot is the four columns from
// (x, z) to (x+1, z+1), and it does not stand where one of them is the mouth of a cave.
func (t terrain) redwood(x, y, z, trunk int) {
	feet := [4][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}}
	var grounds [4]int
	for i, d := range feet {
		l := landAt(x+d[0], z+d[1], t.seed)
		if holeAt(x+d[0], l.top, z+d[1], l, t.seed) {
			return
		}
		grounds[i] = l.top
	}
	for i, d := range feet {
		// Each quarter of the trunk goes down to its own ground.
		for at := min(y, grounds[i]+1); at < y+trunk; at++ {
			t.put(x+d[0], at, z+d[1], RedwoodLog, false)
		}
	}

	top := y + trunk + 1
	first := y + trunk*11/20
	for at := first; at <= top; at++ {
		reach := 1 + 3*(top-at)/(top-first)
		if (at-first)%2 == 1 && reach > 1 {
			reach-- // every other layer is drawn in, so the crown stands in steps
		}
		// A round layer about the middle of the four logs, which is no block's middle.
		edge := (float64(reach) + 0.5) * (float64(reach) + 0.5)
		for dz := -reach; dz <= reach+1; dz++ {
			for dx := -reach; dx <= reach+1; dx++ {
				fx, fz := float64(dx)-0.5, float64(dz)-0.5
				if fx*fx+fz*fz <= edge {
					t.put(x+dx, at, z+dz, PineNeedles, true)
				}
			}
		}
	}
}

// treeFern stands a tree fern: a dark trunk with a star of fronds on its top, whose
// tips hang down.
func (t terrain) treeFern(x, y, z, trunk int) {
	t.trunk(x, y, z, trunk, FernTrunk)
	top := y + trunk
	t.crown(x, top, z, 1, Fronds, false)
	for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		t.put(x+2*d[0], top, z+2*d[1], Fronds, true)
		t.put(x+3*d[0], top-1, z+3*d[1], Fronds, true)
	}
	for _, d := range [4][2]int{{1, 1}, {-1, 1}, {1, -1}, {-1, -1}} {
		t.put(x+d[0], top, z+d[1], Fronds, true)
		t.put(x+2*d[0], top-1, z+2*d[1], Fronds, true)
	}
}

// stumpPalm stands a stump palm: a short armored trunk with a crown of stiff leaves
// standing on it.
func (t terrain) stumpPalm(x, y, z, trunk int) {
	t.trunk(x, y, z, trunk, StumpPalm)
	t.put(x, y+trunk, z, PalmCrown, false)
}

package world

import "math"

// A camp is where other people live: a fire on a ring of stones, and two or three huts of
// hide round it, each with its door toward the fire. The seed puts camps down as it does
// landmarks: the ground is cut into squares, each holding one camp or none, well inside
// it, and only in the green lands. One camp always lies within a walk of where a world
// begins, so the first people are found soon.
const (
	campSquare = 256 // the side of a square
	campReach  = 10  // nothing of a camp lies farther than this from its fire, along x or z
	campChance = 0.6 // the part of the squares that may hold a camp that do
	campNear   = 120 // the first camp lies this far from where a world begins, up to campFar
	campFar    = 200
	campApart  = 96 // and no other camp lies nearer to it than this, along x or z
	firstTries = 48 // the places that are tried for the first camp, until one will hold it
	campClear  = 20 // how high over its ground a camp is cleared of what grew or lay there
	hutOut     = 6  // how far from the fire a hut's middle stands
	hutBlocks  = 24 // the blocks of tent in a hut: two rings of walls, a roof and its top
)

type camp struct {
	x, y, z int     // the ground under its fire
	huts    int     // how many huts stand round it
	turn    float64 // the way from the fire to the first hut, in radians
}

// grassLand reports whether people make camps in a country: the green lands, where the
// beasts they live on graze.
func grassLand(b Biome) bool {
	return b == Prairie || b == Highland || b == Jungle
}

// campAt returns a camp whose fire is on the column at x, z, and whether the ground there
// will hold one: green land, over the water. gx and gz pick its huts.
func campAt(x, z int, seed uint32, gx, gz int) (camp, bool) {
	l := landAt(x, z, seed)
	c := camp{x: x, y: l.top, z: z, huts: 2 + int(lattice(gx, gz, seed+9103)*2), turn: 2 * math.Pi * lattice(gx, gz, seed+9104)}
	return c, grassLand(l.biome) && l.top >= SeaLevel
}

// firstCamp returns the camp within a walk of where a world begins: the first of some
// places, each a new way out and a new distance, whose ground will hold one.
func firstCamp(seed uint32) (camp, bool) {
	start, far := 2*math.Pi*lattice(0, 0, seed+9111), lattice(0, 0, seed+9112)
	for i := range firstTries {
		a := start + float64(i)*2.399963 // the golden angle: each try a way out between those tried
		d := campNear + (campFar-campNear)*math.Mod(far+float64(i)*0.618034, 1)
		if c, ok := campAt(int(math.Round(d*math.Cos(a))), int(math.Round(d*math.Sin(a))), seed, -1, i); ok {
			return c, true
		}
	}
	return camp{}, false
}

// campsNear calls f with every camp whose reach comes within more blocks of the box from
// x0, z0 to x1, z1, the first camp first if it does. A more of -campReach asks for the
// camps whose fires are in the box. first is the first camp, if there is one, which
// keeps the others away.
func campsNear(seed uint32, first camp, hasFirst bool, x0, z0, x1, z1, more int, f func(camp)) {
	pad := campReach + more
	near := func(x, z int) bool {
		return x+pad >= x0 && x-pad <= x1 && z+pad >= z0 && z-pad <= z1
	}
	if hasFirst && near(first.x, first.z) {
		f(first)
	}
	room := float64(campSquare - 2*campReach)
	for gz := floorDiv(z0-pad, campSquare); gz <= floorDiv(z1+pad, campSquare); gz++ {
		for gx := floorDiv(x0-pad, campSquare); gx <= floorDiv(x1+pad, campSquare); gx++ {
			// Where the camp of the square would be is cheap to say, and whether the ground
			// holds it is not: that is asked only of one that is near.
			x := gx*campSquare + campReach + int(lattice(gx, gz, seed+9101)*room)
			z := gz*campSquare + campReach + int(lattice(gx, gz, seed+9102)*room)
			if lattice(gx, gz, seed+9100) >= campChance || !near(x, z) ||
				hasFirst && abs(x-first.x) < campApart && abs(z-first.z) < campApart {
				continue
			}
			if c, ok := campAt(x, z, seed, gx, gz); ok {
				f(c)
			}
		}
	}
}

// camps builds the part in this chunk of every camp that reaches into it, and returns the
// camps near enough that a tree rooted in or by the chunk could reach into one: those
// keep their trees out.
func (t terrain) camps() []camp {
	x0, z0 := t.cx*ChunkSize, t.cz*ChunkSize
	x1, z1 := x0+ChunkSize-1, z0+ChunkSize-1
	var near []camp
	campsNear(t.seed, t.first, t.hasFirst, x0, z0, x1, z1, 2*treeReach, func(c camp) {
		if near = append(near, c); c.x+campReach >= x0 && c.x-campReach <= x1 && c.z+campReach >= z0 && c.z-campReach <= z1 {
			t.camp(c)
		}
	})
	return near
}

// inCamp reports whether a tree rooted at x, z could reach into one of the camps.
func inCamp(x, z int, camps []camp) bool {
	for _, c := range camps {
		if abs(x-c.x) <= campReach+treeReach && abs(z-c.z) <= campReach+treeReach {
			return true
		}
	}
	return false
}

// camp builds a camp's part in this chunk. Its ground is cleared of what grew there, and
// of anything else the seed put there first. The fire burns on the middle of a ring of
// stones laid in the ground, and the huts stand round it.
func (t terrain) camp(c camp) {
	for z := c.z - campReach; z <= c.z+campReach; z++ {
		for x := c.x - campReach; x <= c.x+campReach; x++ {
			lx, lz := x-t.cx*ChunkSize, z-t.cz*ChunkSize
			if uint(lx) >= ChunkSize || uint(lz) >= ChunkSize {
				continue
			}
			ground := landAt(x, z, t.seed).top
			for y := ground + 1; y <= min(ground+campClear, Height-1); y++ {
				if at := (y*ChunkSize+lz)*ChunkSize + lx; t.blocks[at] != Water {
					t.blocks[at] = Air
				}
			}
		}
	}
	for dz := -1; dz <= 1; dz++ {
		for dx := -1; dx <= 1; dx++ {
			t.raise(c.x+dx, c.z+dz, c.y-1)
			t.put(c.x+dx, c.y, c.z+dz, Cobble, false)
		}
	}
	t.put(c.x, c.y+1, c.z, Fire, false)
	for i := range c.huts {
		a := c.turn + 2*math.Pi*float64(i)/float64(c.huts)
		t.hut(c.x+int(math.Round(hutOut*math.Cos(a))), c.z+int(math.Round(hutOut*math.Sin(a))), a+math.Pi)
	}
}

// raise fills the column at x, z with dirt up to y, where its ground lies lower: so
// what a camp lays down stands on something.
func (t terrain) raise(x, z, y int) {
	for up := landAt(x, z, t.seed).top + 1; up <= y; up++ {
		t.put(x, up, z, Dirt, false)
	}
}

// hut builds a hut of hide stretched over poles: three blocks across, walls two high with
// a door in them, and a roof, with its middle at x, z on that column's ground. toFire is
// the way to the fire from it, which its door faces.
func (t terrain) hut(x, z int, toFire float64) {
	g := landAt(x, z, t.seed).top
	var door [2]int
	if dx, dz := math.Cos(toFire), math.Sin(toFire); math.Abs(dx) > math.Abs(dz) {
		door[0] = int(math.Copysign(1, dx))
	} else {
		door[1] = int(math.Copysign(1, dz))
	}
	for dz := -1; dz <= 1; dz++ {
		for dx := -1; dx <= 1; dx++ {
			t.raise(x+dx, z+dz, g)
			wall := (dx != 0 || dz != 0) && [2]int{dx, dz} != door
			for y := g + 1; y <= g+2; y++ {
				b := Air
				if wall {
					b = Tent
				}
				t.put(x+dx, y, z+dz, b, false)
			}
			t.put(x+dx, g+3, z+dz, Tent, false)
		}
	}
	t.put(x, g+4, z, Tent, false)
}

// Camps returns the cells of the fires of the camps within reach blocks of the column at
// x, z, along x and along z, as the seed put them there. A fire may have been put out
// since. A world that is not made from a seed has none.
func (w *World) Camps(x, z, reach float64) []Cell {
	if w.gen == nil {
		return nil
	}
	seed := uint32(w.Seed) ^ uint32(w.Seed>>32)
	first, hasFirst := firstCamp(seed)
	var fires []Cell
	x0, z0 := int(math.Floor(x-reach)), int(math.Floor(z-reach))
	x1, z1 := int(math.Floor(x+reach)), int(math.Floor(z+reach))
	campsNear(seed, first, hasFirst, x0, z0, x1, z1, -campReach, func(c camp) {
		fires = append(fires, Cell{c.x, c.y + 1, c.z})
	})
	return fires
}

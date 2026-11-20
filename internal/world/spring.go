package world

import "math"

// What a fire mountain has round it and in it besides its fire: hot springs on the ash
// at its foot, some with a geyser, and a tube that lava once ran through, into its side.
// Each is placed by the seed from the mountain's own place, so a chunk builds its part of
// one as it builds its part of the mountain.
const (
	springNear  = 50  // the least a spring lies from the middle of its mountain
	springFar   = 60  // and the most: so all of it is on the ash, off the cone
	springReach = 8   // a spring reaches no farther than this from its middle, rings and all
	tubeFrom    = 13  // how far from the middle of the mountain the chamber of its tube is
	tubeTo      = 36  // and how far out the tube runs: to where it comes out of the cone's side
	chamberWide = 4.5 // how far from its middle the chamber reaches
)

// spring is a hot spring: a pool of warm water in rings of colored stone.
type spring struct {
	x, y, z int     // the middle of its pool, at the level of the water
	wide    float64 // how far from its middle the water reaches
	geyser  bool    // a geyser stands in the middle of it
}

// springs returns the hot springs of a mountain: three or four, at even bearings round
// it, the first of them with a geyser, and the others with one or not.
func (m mountain) springs(seed uint32) []spring {
	n := 3 + int(lattice(m.x, m.z, seed+9201)*2)
	turn := 2 * math.Pi * lattice(m.x, m.z, seed+9202)
	out := make([]spring, 0, 4)
	for i := range n {
		bearing := turn + 2*math.Pi*float64(i)/float64(n)
		far := springNear + (springFar-springNear)*lattice(m.x+i, m.z, seed+9203)
		x, z := m.x+int(math.Round(far*math.Cos(bearing))), m.z+int(math.Round(far*math.Sin(bearing)))
		l := landAt(x, z, seed)
		if l.biome != Ashlands {
			continue // the sea came up to the mountain here
		}
		out = append(out, spring{
			x: x, y: l.top, z: z,
			wide:   2.6 + 1.2*lattice(m.x+i, m.z, seed+9204),
			geyser: len(out) == 0 || lattice(m.x+i, m.z, seed+9205) < 0.5,
		})
	}
	return out
}

// tube returns the way the lava tube of a mountain runs out from the mountain's middle,
// one long, and the layer of its floor: a step under the ground where it comes out of
// the cone's side, so that its mouth is open there whatever the land is like.
func (m mountain) tube(seed uint32) (way [2]float64, floor int) {
	bearing := 2 * math.Pi * lattice(m.x, m.z, seed+9211)
	way = [2]float64{math.Cos(bearing), math.Sin(bearing)}
	out := landAt(m.x+int(math.Floor(0.5+tubeTo*way[0])), m.z+int(math.Floor(0.5+tubeTo*way[1])), seed)
	return way, out.top - 1
}

// mountainOf returns the mountain of the square of the grid that a column lies in,
// whether or not the column is on it.
func mountainOf(x, z int, seed uint32) mountain {
	gx, gz := floorDiv(x+mountSquare/2, mountSquare), floorDiv(z+mountSquare/2, mountSquare)
	mx, mz := mountPlace(gx, gz, seed)
	m, _, _ := mountainNear(mx, mz, seed)
	return m
}

// mountains builds the part that lies in this chunk of what the mountain near it has
// besides its cone: its springs, and its tube.
func (t terrain) mountains() {
	x0, z0 := t.cx*ChunkSize, t.cz*ChunkSize
	m := mountainOf(x0+ChunkSize/2, z0+ChunkSize/2, t.seed)
	if math.Hypot(float64(x0+ChunkSize/2-m.x), float64(z0+ChunkSize/2-m.z)) > mountAsh+ChunkSize {
		return
	}
	near := func(x, z, reach int) bool {
		return x+reach >= x0 && x-reach < x0+ChunkSize && z+reach >= z0 && z-reach < z0+ChunkSize
	}
	for _, s := range m.springs(t.seed) {
		if near(s.x, s.z, springReach) {
			t.spring(s)
		}
	}
	if near(m.x, m.z, tubeTo+2) {
		t.tube(m)
	}
}

// spring builds a hot spring. The ground is made level out to its last ring, the pool
// is dug in it, a block deep, so that a person stands in it with the head out, and round
// the water lie three rings of the stone a hot spring leaves, each its own color.
func (t terrain) spring(s spring) {
	for dz := -springReach; dz <= springReach; dz++ {
		for dx := -springReach; dx <= springReach; dx++ {
			x, z := s.x+dx, s.z+dz
			far := math.Hypot(float64(dx), float64(dz))
			if far > s.wide+3.5 {
				continue
			}
			// Level: rock up to the water's layer, and nothing over it.
			for y := landAt(x, z, t.seed).top + 1; y <= s.y; y++ {
				t.put(x, y, z, Basalt, false)
			}
			for y := s.y + 1; y <= s.y+6; y++ {
				t.put(x, y, z, Air, false)
			}
			switch {
			case far <= s.wide:
				t.put(x, s.y, z, Water, false)
				t.put(x, s.y-1, z, SinterRust, false) // the floor of the pool
			case far <= s.wide+1.2:
				t.put(x, s.y, z, SinterRust, false)
			case far <= s.wide+2.4:
				t.put(x, s.y, z, SinterGold, false)
			default:
				t.put(x, s.y, z, SinterGreen, false)
			}
		}
	}
}

// tube builds the lava tube of a mountain: a tunnel two blocks wide and three high that
// runs level into the side of the cone, to a chamber with a pool of lava in its floor
// and obsidian and firestone in its walls.
func (t terrain) tube(m mountain) {
	way, floor := m.tube(t.seed)
	cx, cz := float64(m.x)+0.5+tubeFrom*way[0], float64(m.z)+0.5+tubeFrom*way[1] // the middle of the chamber
	for dz := -tubeTo - 1; dz <= tubeTo+1; dz++ {
		for dx := -tubeTo - 1; dx <= tubeTo+1; dx++ {
			x, z := m.x+dx, m.z+dz
			px, pz := float64(x)+0.5-(float64(m.x)+0.5), float64(z)+0.5-(float64(m.z)+0.5)
			along, across := px*way[0]+pz*way[1], px*way[1]-pz*way[0]
			room := math.Hypot(float64(x)+0.5-cx, float64(z)+0.5-cz)
			switch {
			case room <= chamberWide:
				for y := floor + 1; y <= floor+4; y++ {
					t.put(x, y, z, Air, false)
				}
				t.put(x, floor, z, Basalt, false)
				if room <= 1.6 {
					t.put(x, floor, z, Lava, false) // the pool, with rock under it
					t.put(x, floor-1, z, Basalt, false)
				}
			case along >= tubeFrom && along <= tubeTo && math.Abs(across) <= 1:
				for y := floor + 1; y <= floor+3; y++ {
					t.put(x, y, z, Air, false)
				}
				t.put(x, floor, z, Basalt, false)
				// Lava still hangs in the roof here and there, where there is a roof, and
				// lights the way: out of reach, and at the mouth it is seen from outside.
				if math.Abs(across) <= 0.5 && int(math.Floor(along))%6 == 0 && t.get(x, floor+4, z).Info().Solid {
					t.put(x, floor+4, z, Lava, false)
				}
			case room <= chamberWide+1:
				// The wall of the chamber: what the mountain keeps is in it.
				for y := floor + 1; y <= floor+3; y++ {
					switch h := lattice(x*7+y, z, t.seed+9212); {
					case h < 0.3:
						t.put(x, y, z, ObsidianRock, false)
					case h < 0.42:
						t.put(x, y, z, FirestoneRock, false)
					}
				}
			}
		}
	}
}

// get returns the block at a place in the world as this chunk has it so far, and air for
// a place that is not in this chunk.
func (t terrain) get(x, y, z int) Block {
	lx, lz := x-t.cx*ChunkSize, z-t.cz*ChunkSize
	if uint(lx) >= ChunkSize || uint(lz) >= ChunkSize || uint(y) >= Height {
		return Air
	}
	return t.blocks[(y*ChunkSize+lz)*ChunkSize+lx]
}

// springAt returns the spring whose water the column at x, z is in, if it is in one.
func springAt(x, z int, seed uint32) (spring, bool) {
	m := mountainOf(x, z, seed)
	if far := math.Hypot(float64(x-m.x), float64(z-m.z)); far < springNear-springReach || far > springFar+springReach {
		return spring{}, false
	}
	for _, s := range m.springs(seed) {
		if math.Hypot(float64(x-s.x), float64(z-s.z)) <= s.wide {
			return s, true
		}
	}
	return spring{}, false
}

// Spring reports whether the column at x, z is in the warm water of a hot spring, as
// the seed put it there.
func (w *World) Spring(x, z float64) bool {
	if w.gen == nil {
		return false
	}
	_, ok := springAt(int(math.Floor(x)), int(math.Floor(z)), uint32(w.Seed)^uint32(w.Seed>>32))
	return ok
}

// Geysers returns the cells the geysers within reach blocks of the column at x, z spout
// from: the water in the middle of the pools they stand in. A world that is not made
// from a seed has none.
func (w *World) Geysers(x, z, reach float64) []Cell {
	if w.gen == nil {
		return nil
	}
	seed := uint32(w.Seed) ^ uint32(w.Seed>>32)
	m := mountainOf(int(math.Floor(x)), int(math.Floor(z)), seed)
	if math.Hypot(float64(m.x)-x, float64(m.z)-z) > springFar+springReach+reach {
		return nil
	}
	var vents []Cell
	for _, s := range m.springs(seed) {
		if s.geyser && math.Hypot(float64(s.x)+0.5-x, float64(s.z)+0.5-z) <= reach {
			vents = append(vents, Cell{s.x, s.y, s.z})
		}
	}
	return vents
}

// Tube returns where the lava tube of the fire mountain nearest the column at x, z comes
// out of the cone, at the height of its floor, and the middle of the chamber it runs to.
func (w *World) Tube(x, z float64) (mouth, chamber Cell, ok bool) {
	crater, _, ok := w.Mountain(x, z)
	if !ok {
		return Cell{}, Cell{}, false
	}
	seed := uint32(w.Seed) ^ uint32(w.Seed>>32)
	m, _, _ := mountainNear(crater.X, crater.Z, seed)
	way, floor := m.tube(seed)
	at := func(far float64) Cell {
		return Cell{m.x + int(math.Floor(0.5+far*way[0])), floor + 1, m.z + int(math.Floor(0.5+far*way[1]))}
	}
	return at(tubeTo - 2), at(tubeFrom), true // the mouth: a step inside where the tube ends
}

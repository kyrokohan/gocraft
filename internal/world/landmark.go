package world

// A landmark is a thing the seed puts down that is not of the land it stands in: a
// great skeleton, or a tree that lightning struck. The ground is cut into big squares,
// and each holds one landmark or none, well inside it. So a chunk only asks about the
// squares it lies in, and builds its part of a landmark from the seed alone, as it does
// a tree.
const (
	markSquare = 160 // the side of a square
	markReach  = 17  // a landmark reaches no farther than this from its middle
)

type landmark struct {
	x, z  int  // its middle
	along bool // it lies along x, and not along z
	ribs  int  // how many blocks of its length are ribs
}

// landmarkIn returns the landmark of a square of the grid, and whether it has one.
func landmarkIn(gx, gz int, seed uint32) (landmark, bool) {
	room := float64(markSquare - 2*markReach)
	m := landmark{
		x:     gx*markSquare + markReach + int(lattice(gx, gz, seed+9002)*room),
		z:     gz*markSquare + markReach + int(lattice(gx, gz, seed+9003)*room),
		along: lattice(gx, gz, seed+9004) < 0.5,
		ribs:  12 + 2*int(lattice(gx, gz, seed+9005)*5),
	}
	// Most of the dead lie in the badlands, and a few on the prairie.
	chance := lattice(gx, gz, seed+9001)
	switch landAt(m.x, m.z, seed).biome {
	case Badlands:
		return m, chance < 0.8
	case Prairie:
		return m, chance < 0.3
	}
	return m, false
}

// landmarks builds the part that lies in this chunk of every landmark near it.
func (t terrain) landmarks() {
	x0, z0 := t.cx*ChunkSize, t.cz*ChunkSize
	near := func(x, z, reach int) bool {
		return x+reach >= x0 && x-reach < x0+ChunkSize && z+reach >= z0 && z-reach < z0+ChunkSize
	}
	for gz := floorDiv(z0, markSquare); gz <= floorDiv(z0+ChunkSize-1, markSquare); gz++ {
		for gx := floorDiv(x0, markSquare); gx <= floorDiv(x0+ChunkSize-1, markSquare); gx++ {
			if m, ok := landmarkIn(gx, gz, t.seed); ok && near(m.x, m.z, markReach) {
				t.skeleton(m)
			}
			if s, ok := struckIn(gx, gz, t.seed); ok && near(s.x, s.z, struckReach) {
				t.struckTree(s)
			}
		}
	}
}

// struck is a tree that lightning struck: a charred trunk that still burns at its top,
// with a limb that fell from it burning at its foot. It is where fire is found.
type struck struct {
	x, y, z int // the ground it stands on
	tall    int // how many blocks of trunk are left
	limb    int // which way from the trunk its fallen limb lies, of the four
}

const (
	struckReach  = 2   // the limb, and the burnt ground, lie no farther than this from the trunk
	struckChance = 0.7 // the part of the squares that could hold a struck tree that do
)

// struckIn returns the struck tree of a square of the grid, and whether it has one: a
// square of dry land with no skeleton in it.
func struckIn(gx, gz int, seed uint32) (struck, bool) {
	if _, bones := landmarkIn(gx, gz, seed); bones {
		return struck{}, false
	}
	room := float64(markSquare - 2*markReach)
	s := struck{
		x:    gx*markSquare + markReach + int(lattice(gx, gz, seed+9012)*room),
		z:    gz*markSquare + markReach + int(lattice(gx, gz, seed+9013)*room),
		tall: 6 + int(lattice(gx, gz, seed+9014)*4),
		limb: int(lattice(gx, gz, seed+9015) * 4),
	}
	l := landAt(s.x, s.z, seed)
	s.y = l.top
	return s, l.biome != Sea && l.biome != Shore && l.biome != Ashlands && lattice(gx, gz, seed+9011) < struckChance
}

// fires returns the cells of the two fires of a struck tree, the one on its top first,
// and whether it has the one at its foot: a limb that fell in water has gone out.
func (s struck) fires(seed uint32) (top, foot Cell, limb bool) {
	d := [4][2]int{{struckReach, 0}, {-struckReach, 0}, {0, struckReach}, {0, -struckReach}}[s.limb]
	x, z := s.x+d[0], s.z+d[1]
	ground := landAt(x, z, seed).top
	return Cell{s.x, s.y + s.tall + 1, s.z}, Cell{x, ground + 2, z}, ground >= SeaLevel
}

// struckTree builds a tree that lightning struck. What grew within reach of it is
// burnt away, and the moss of the ground with it.
func (t terrain) struckTree(s struck) {
	for dz := -struckReach; dz <= struckReach; dz++ {
		for dx := -struckReach; dx <= struckReach; dx++ {
			x, z := s.x+dx, s.z+dz
			lx, lz := x-t.cx*ChunkSize, z-t.cz*ChunkSize
			if uint(lx) >= ChunkSize || uint(lz) >= ChunkSize {
				continue
			}
			ground := landAt(x, z, t.seed).top
			for y := ground; y < Height; y++ {
				at := (y*ChunkSize+lz)*ChunkSize + lx
				switch b := t.blocks[at]; {
				case y == ground && b == Turf:
					t.blocks[at] = Dirt
				case y > ground && (b.Info().Stands() || b.Info().Shade == 1 && b != Water):
					t.blocks[at] = Air // a plant, or the leaves of a tree
				}
			}
		}
	}
	for y := s.y + 1; y <= s.y+s.tall; y++ {
		t.put(s.x, y, s.z, CharredLog, false)
	}
	top, foot, limb := s.fires(t.seed)
	t.put(top.X, top.Y, top.Z, Fire, false)
	if limb {
		t.put(foot.X, foot.Y-1, foot.Z, CharredLog, false)
		t.put(foot.X, foot.Y, foot.Z, Fire, false)
	}
}

// Struck returns the cells of the fires of the struck trees whose trunks stand within
// reach blocks of the column at x, z, as the seed put them there: the fire on a tree's
// top, and then the one at its foot. One may have been put out since. A world that is
// not made from a seed has none.
func (w *World) Struck(x, z, reach float64) []Cell {
	if w.gen == nil {
		return nil
	}
	seed := uint32(w.Seed) ^ uint32(w.Seed>>32)
	var fires []Cell
	for gz := floorDiv(int(z-reach), markSquare); gz <= floorDiv(int(z+reach), markSquare); gz++ {
		for gx := floorDiv(int(x-reach), markSquare); gx <= floorDiv(int(x+reach), markSquare); gx++ {
			s, ok := struckIn(gx, gz, seed)
			if !ok || float64(s.x)+0.5-x > reach || x-float64(s.x)-0.5 > reach ||
				float64(s.z)+0.5-z > reach || z-float64(s.z)-0.5 > reach {
				continue
			}
			top, foot, limb := s.fires(seed)
			if fires = append(fires, top); limb {
				fires = append(fires, foot)
			}
		}
	}
	return fires
}

// The parts of a great skeleton, in blocks along it, and how high over the ground at
// its middle the spine of the shortest runs. A longer one stands higher.
const (
	skullLong = 5
	neckLong  = 2
	tailLong  = 6
	spineHigh = 5
)

// skeleton builds the bones of a great beast that died long ago: a skull, a spine held
// up by ribs that stand as arches wide and high enough to walk through, and a tail that
// sinks to the ground. A bone only takes the place of air, so where the ground has risen
// the skeleton lies half buried in it.
func (t terrain) skeleton(m landmark) {
	// a counts along the skeleton from its middle and c across it.
	at := func(a, c int) (x, z int) {
		if m.along {
			return m.x + a, m.z + c
		}
		return m.x + c, m.z + a
	}
	ground := func(a, c int) int {
		x, z := at(a, c)
		return landAt(x, z, t.seed).top
	}
	bone := func(a, y, c int) {
		x, z := at(a, c)
		t.put(x, y, z, Bone, true)
	}

	a := -(skullLong + neckLong + m.ribs + tailLong) / 2
	high := spineHigh + (m.ribs-12)/4
	spine := ground(0, 0) + high

	// The skull lies on its own ground: a hollow box with a narrower, lower snout, and
	// a hole on each side for an eye.
	head := ground(a+2, 0)
	for da := range skullLong {
		for c := -2; c <= 2; c++ {
			for h := 1; h <= 4; h++ {
				outsideSnout := da < 2 && (abs(c) == 2 || h == 4)
				eye := da == 3 && abs(c) == 2 && h == 3
				hollow := da > 0 && da < skullLong-1 && abs(c) < 2 && h > 1 && h < 4
				if !outsideSnout && !eye && !hollow {
					bone(a+da, head+h, c)
				}
			}
		}
	}
	a += skullLong

	// The neck climbs from the top of the skull to the spine.
	for i := range neckLong {
		bone(a+i, head+4+(spine-head-4)*(i+1)/(neckLong+1), 0)
	}
	a += neckLong

	// Under every other block of the spine, a rib on each side: out and down from the
	// spine, then straight down to the ground.
	for i := range m.ribs {
		bone(a+i, spine, 0)
		if i%2 == 1 {
			continue
		}
		for _, side := range [2]int{-1, 1} {
			bone(a+i, spine, side)
			bone(a+i, spine-1, 2*side)
			for y := spine - 2; y > ground(a+i, 3*side) && y > spine-2-8; y-- {
				bone(a+i, y, 3*side)
			}
		}
	}
	a += m.ribs

	for i := range tailLong {
		bone(a+i, spine-(i+1)*(high-1)/tailLong, 0)
	}
}

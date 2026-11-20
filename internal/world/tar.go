package world

import "math"

// A tar pit is a landmark, as a great skeleton is: a round black pool, three deep, level
// with the ground at its middle, with the bones of what it caught standing out of it.
// The seed puts one in some squares of the swamp and of the badlands that hold no
// skeleton.
const (
	tarDeep   = 3   // how deep the tar lies
	tarSwamp  = 0.5 // the part of the squares whose place is in the swamp that hold a pit
	tarBarren = 0.3 // and of those in the badlands
)

type tarPit struct {
	x, y, z int     // the middle of it, at the level of the top of its tar
	wide    float64 // how far from its middle the tar reaches
}

// tarIn returns the tar pit of a square of the grid, and whether it has one.
func tarIn(gx, gz int, seed uint32) (tarPit, bool) {
	if _, bones := landmarkIn(gx, gz, seed); bones {
		return tarPit{}, false
	}
	room := float64(markSquare - 2*markReach)
	p := tarPit{
		x:    gx*markSquare + markReach + int(lattice(gx, gz, seed+9022)*room),
		z:    gz*markSquare + markReach + int(lattice(gx, gz, seed+9023)*room),
		wide: 3 + 2*lattice(gx, gz, seed+9024),
	}
	l := landAt(p.x, p.z, seed)
	p.y = max(l.top, SeaLevel+1) // on the bank, if its middle falls in a pool
	switch chance := lattice(gx, gz, seed+9021); l.biome {
	case Swamp:
		return p, chance < tarSwamp
	case Badlands:
		return p, chance < tarBarren
	}
	return p, false
}

// tarPit builds the part of a tar pit that lies in this chunk: the tar, with nothing
// over it; a bank round it, raised to its level where the ground is lower; a floor
// under it, where there was none; and the bones.
func (t terrain) tarPit(p tarPit) {
	reach := int(p.wide) + 2
	for dz := -reach; dz <= reach; dz++ {
		for dx := -reach; dx <= reach; dx++ {
			x, z := p.x+dx, p.z+dz
			far := math.Hypot(float64(dx), float64(dz))
			l := landAt(x, z, t.seed)
			bank := Redrock
			if l.biome == Swamp || l.biome == Sea || l.biome == Shore {
				bank = Peat
			}
			switch {
			case far <= p.wide:
				for y := p.y + 1; y <= p.y+8; y++ {
					t.put(x, y, z, Air, false)
				}
				for y := p.y - tarDeep + 1; y <= p.y; y++ {
					t.put(x, y, z, Tar, false)
				}
				t.put(x, p.y-tarDeep, z, bank, false) // its floor
				// The bones of what it caught: a rib here and there, standing out of it.
				if h := lattice(x, z, t.seed+9025); h < 0.07 && far < p.wide-0.5 {
					for y := p.y - 1; y <= p.y+1+int(h*40); y++ {
						t.put(x, y, z, Bone, false)
					}
				}
			case far <= p.wide+1.5:
				for y := min(l.top, p.y-tarDeep); y <= p.y; y++ {
					t.put(x, y, z, bank, false) // the bank that holds it
				}
			}
		}
	}
}

package world

import "math"

// starNights is how many nights there are to each one on which a star falls near the
// player.
const starNights = 3

// StarFall reports whether a star falls near the player on a night of the world, counted
// from 1, and at what part of that night, from 0 to 1. It is by the seed alone, so
// nothing of when is kept in a save. On a night that is sure to have one, only the part
// is the seed's. A world that is not made from a seed has none.
func (w *World) StarFall(night int, sure bool) (part float64, ok bool) {
	if w.gen == nil {
		return 0, false
	}
	seed := uint32(w.Seed) ^ uint32(w.Seed>>32)
	return lattice(night, 2, seed+9401), sure || lattice(night, 1, seed+9402) < 1.0/starNights
}

// Cold reports whether the land at the column at x, z is one of the cold lands, where a
// body loses its warmth at any time of the world: the tundra and the crags. A world that
// is not made from a seed has none.
func (w *World) Cold(x, z float64) bool {
	if w.gen == nil {
		return false
	}
	b := landAt(int(math.Floor(x)), int(math.Floor(z)), uint32(w.Seed)^uint32(w.Seed>>32)).biome
	return b == Tundra || b == Crags
}

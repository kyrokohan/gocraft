package main

// The cave person's picture is seven tiles for each look, side by side, which the
// renderer lays on the boxes the body is built from: the face, the hair, the front, the
// back and a side of the hide it wears, an arm and a leg. Every look is the same
// drawing in other colors, so the body keeps its shape whichever is worn.

// look is the colors of one cave person.
type look struct {
	skin, hair, hide, trim rgb
	spots, beard           bool // the hide is spotted; the face has a beard
}

// looks are the cave people a player can be. internal/assets says how many there are.
var looks = []look{
	{skin: rgb{198, 142, 100}, hair: rgb{40, 32, 30}, hide: rgb{124, 88, 56}, trim: rgb{214, 196, 160}},
	{skin: rgb{120, 78, 52}, hair: rgb{30, 26, 26}, hide: rgb{178, 142, 92}, trim: rgb{92, 64, 44}, spots: true},
	{skin: rgb{226, 180, 146}, hair: rgb{170, 74, 38}, hide: rgb{128, 126, 122}, trim: rgb{226, 222, 212}, beard: true},
	{skin: rgb{176, 128, 86}, hair: rgb{76, 50, 34}, hide: rgb{88, 64, 48}, trim: rgb{176, 132, 84}},
	{skin: rgb{94, 62, 44}, hair: rgb{170, 166, 160}, hide: rgb{136, 96, 60}, trim: rgb{232, 226, 210}, beard: true},
	{skin: rgb{210, 160, 118}, hair: rgb{198, 172, 114}, hide: rgb{204, 176, 128}, trim: rgb{110, 76, 50}, spots: true},
}

// personTiles returns the painters of every look's tiles, in the order they are drawn.
func personTiles() []painter {
	var tiles []painter
	for i, l := range looks {
		seed := 400 + 20*i
		tiles = append(tiles, l.face(seed), l.hairTile(seed), l.hideFront(seed), l.hideBack(seed), l.hideSide(seed), l.arm(seed), l.leg(seed))
	}
	return tiles
}

func (l look) skinAt(x, y, seed int) rgb {
	return grain(l.skin, 0.04, x, y, seed)
}

// hairAt is hair that hangs: each strand its own shade from top to bottom.
func (l look) hairAt(x, y, seed int) rgb {
	return grain(l.hair.scale(0.78+0.44*hash(x, 0, seed+1)), 0.06, x, y, seed+2)
}

// furAt is the hide: short strokes of fur, with dark spots on a spotted one.
func (l look) furAt(x, y, seed int) rgb {
	c := l.hide.scale(0.84 + 0.3*hash(x, y/2, seed+3))
	if l.spots && blotch(x, y, 4, seed+4) > 0.7 {
		c = c.scale(0.5)
	}
	return grain(c, 0.05, x, y, seed+5)
}

// face is the front of the head: hair over the brow and down both sides, two eyes, a
// nose and a mouth, and a beard on some.
func (l look) face(seed int) painter {
	return func(x, y int) (rgb, bool) {
		side := min(x, size-1-x) // how far in from the nearer side
		switch {
		case y < 4, side < 2 && y < 11:
			return l.hairAt(x, y, seed), true
		case y == 6 && side >= 3 && side <= 5:
			return l.hair.scale(0.8), true // a brow
		case y == 7 && side == 4, y == 8 && side == 4:
			return rgb{28, 24, 24}, true // an eye
		case y == 7 && side == 3:
			return rgb{236, 232, 224}, true // its white
		case (y == 9 || y == 10) && side == 7:
			return l.skin.scale(0.82), true // the nose
		case y == 12 && side >= 5:
			return rgb{112, 52, 46}, true // the mouth
		case l.beard && y >= 11:
			return l.hairAt(x, y, seed), true
		}
		return l.skinAt(x, y, seed), true
	}
}

func (l look) hairTile(seed int) painter {
	return func(x, y int) (rgb, bool) {
		return l.hairAt(x, y, seed), true
	}
}

// hemmed is the hide with its lower edge trimmed.
func (l look) hemmed(x, y, seed int) rgb {
	if y >= size-2 {
		return grain(l.trim, 0.06, x, y, seed+6)
	}
	return l.furAt(x, y, seed)
}

// hideFront is the chest: the hide hangs from one shoulder and leaves the other bare,
// the one of the arm that holds a tool.
func (l look) hideFront(seed int) painter {
	return func(x, y int) (rgb, bool) {
		switch edge := x - y; {
		case edge > 5:
			return l.skinAt(x, y, seed), true
		case edge >= 4:
			return grain(l.trim, 0.06, x, y, seed+6), true // the stitched edge of the hide
		}
		return l.hemmed(x, y, seed), true
	}
}

// hideBack is the back, with the same shoulder bare.
func (l look) hideBack(seed int) painter {
	return func(x, y int) (rgb, bool) {
		switch edge := x - y; {
		case edge > 8:
			return l.skinAt(x, y, seed), true
		case edge >= 7:
			return grain(l.trim, 0.06, x, y, seed+6), true
		}
		return l.hemmed(x, y, seed), true
	}
}

// hideSide is the side of the body under an arm.
func (l look) hideSide(seed int) painter {
	return func(x, y int) (rgb, bool) {
		if y < 4 {
			return l.skinAt(x, y, seed), true
		}
		return l.hemmed(x, y, seed), true
	}
}

// arm is a bare arm with a band of hide at the wrist, and the hand under it.
func (l look) arm(seed int) painter {
	return func(x, y int) (rgb, bool) {
		switch {
		case y == 10 || y == 11:
			return grain(l.trim, 0.06, x, y, seed+6), true
		case y >= 13:
			return l.skinAt(x, y, seed).scale(0.9), true
		}
		return l.skinAt(x, y, seed), true
	}
}

// leg is a leg from the hip: the ragged skirt of the hide, bare skin, and a foot
// wrapped in fur.
func (l look) leg(seed int) painter {
	return func(x, y int) (rgb, bool) {
		switch {
		case y < 4+int(hash(x, 0, seed+7)*3):
			return l.furAt(x, y, seed), true
		case y == 11:
			return grain(l.trim, 0.06, x, y, seed+6), true // the cord that ties the wrap
		case y > 11:
			return l.furAt(x, y, seed).scale(0.86), true
		}
		return l.skinAt(x, y, seed), true
	}
}

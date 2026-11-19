package main

import "math"

// The things of fire: the fire itself, a hearth cold and lit, what is left of a tree
// that burned, the pot that is baked, and what is cooked, lit and hardened at a fire.

var (
	flameHeart = rgb{255, 250, 222}
	flameMid   = rgb{255, 214, 86}
	flameEdge  = rgb{246, 138, 44}
	flameTip   = rgb{214, 74, 30}
	ashGrey    = rgb{104, 100, 98}
	charBlack  = rgb{36, 32, 32}
	baked      = rgb{184, 104, 66} // clay, once it has been in the fire
)

// tongues is how high the flames of a fire stand in each column of a tile, in pixels.
var tongues = [size]int{2, 5, 4, 8, 11, 9, 13, 16, 14, 15, 10, 12, 7, 8, 4, 2}

// flames returns the color of a fire whose foot is on the row foot of a tile and that
// stands a part high of its full height: white in its heart, low down and in the
// middle, then yellow, then orange, and red at the tips. There is none over a tongue.
func flames(x, y, foot int, high float64) (rgb, bool) {
	top := float64(tongues[x]) * high
	up := float64(foot-y) + 0.5
	if up < 0 || up > top {
		return rgb{}, false
	}
	switch out := up/top + 0.9*math.Abs(float64(x)-7.5)/7.5; {
	case out < 0.42:
		return flameHeart, true
	case out < 0.8:
		return flameMid, true
	case out < 1.2:
		return flameEdge, true
	}
	return flameTip, true
}

// fire is a fire as tall as a block.
func fire(x, y int) (rgb, bool) {
	return flames(x, y, size-1, 1)
}

// hearthRing is the ring of stones of a hearth seen from above: eight stones round a
// hollow. It returns the stone or the gap between two at a pixel, and false in the
// hollow.
func hearthRing(x, y int) (rgb, bool) {
	dx, dy := float64(x)-7.5, float64(y)-7.5
	d := math.Sqrt(dx*dx + dy*dy)
	if d < 4.3 {
		return rgb{}, false
	}
	turn := (math.Atan2(dy, dx)/(2*math.Pi) + 1) * 8
	stoneAt := int(turn) % 8
	if part := turn - math.Floor(turn); d < 7.6 && (part < 0.14 || part > 0.86) || d >= 7.6 && hash(x, y, 580) < 0.5 {
		return grain(rgb{58, 52, 50}, 0.08, x, y, 581), true // the ground between the stones, burnt dark
	}
	c := rgb{140, 138, 142}.scale(0.8 + 0.36*hash(stoneAt, 0, 582))
	if d < 5.2 {
		c = c.scale(0.7) // sooty on the side that faces the fire
	}
	return grain(c, 0.05, x, y, 583), true
}

// hearthTop is a cold hearth from above: ash in the ring, and the black ends of what
// burned there.
func hearthTop(x, y int) (rgb, bool) {
	if c, ok := hearthRing(x, y); ok {
		return c, true
	}
	if d := x - y; (d == 0 || d == 1 || x+y == 15) && hash(x, y, 584) < 0.8 {
		return grain(charBlack, 0.12, x, y, 585), true
	}
	return grain(ashGrey, 0.1, x, y, 586), true
}

// hearthLitTop is a hearth that burns, from above: the fire in the ring, brightest in
// its middle.
func hearthLitTop(x, y int) (rgb, bool) {
	if c, ok := hearthRing(x, y); ok {
		return c.mix(flameEdge, 0.22), true // the stones in the light of it
	}
	dx, dy := float64(x)-7.5, float64(y)-7.5
	switch d := math.Sqrt(dx*dx+dy*dy) + 1.6*blotch(x, y, 4, 587); {
	case d < 2.3:
		return flameHeart, true
	case d < 3.6:
		return flameMid, true
	case d < 4.8:
		return flameEdge, true
	}
	return flameTip, true
}

// hearthStones is a side of a hearth: boulders in two courses, with dark gaps between
// them. It returns whether a pixel is a gap.
func hearthStones(x, y int) (c rgb, gap bool) {
	row := (y + 1) / 6
	at := x + row%2*3
	if y < 3 || (y+1)%6 == 0 || at%6 == 5 {
		return grain(rgb{50, 46, 46}, 0.08, x, y, 588), true
	}
	c = rgb{140, 138, 142}.scale(0.78 + 0.4*hash(at/6, row, 589))
	if (y+1)%6 == 1 || at%6 == 0 {
		c = c.scale(1.14) // the top and the near side of a boulder catch the light
	}
	return grain(c, 0.05, x, y, 590), false
}

// hearthSide is a side of a cold hearth.
func hearthSide(x, y int) (rgb, bool) {
	c, _ := hearthStones(x, y)
	return c, true
}

// hearthLitSide is a side of a hearth that burns: the fire shows through the gaps
// between the stones, and stands over them.
func hearthLitSide(x, y int) (rgb, bool) {
	c, gap := hearthStones(x, y)
	switch {
	case y < 3:
		if f, ok := flames(x, y, 2, 0.2); ok {
			return f, true
		}
		return charBlack, true
	case gap:
		return grain(flameEdge, 0.14, x, y, 591), true
	}
	return c.mix(flameEdge, 0.18), true
}

// charredSide is the side of a tree that burned: black, cracked into squares, and still
// red in a crack here and there.
func charredSide(x, y int) (rgb, bool) {
	if (x%5 == 2 || (y+x/5*2)%6 == 3) && hash(x, y, 592) < 0.85 {
		if hash(x/2, y/2, 593) < 0.12 {
			return grain(rgb{214, 92, 34}, 0.15, x, y, 594), true
		}
		return rgb{16, 14, 14}, true
	}
	return grain(charBlack.scale(0.9+0.5*hash(x/5, (y+x/5*2)/6, 595)), 0.1, x, y, 596), true
}

// charredTop is its top, where it broke off: black, with grey ash in the middle.
func charredTop(x, y int) (rgb, bool) {
	d := max(math.Abs(float64(x)-7.5), math.Abs(float64(y)-7.5))
	switch {
	case d < 3 && hash(x, y, 597) < 0.6:
		return grain(ashGrey, 0.1, x, y, 598), true
	case int(d)%2 == 1:
		return grain(rgb{20, 18, 18}, 0.1, x, y, 599), true
	}
	return grain(charBlack, 0.12, x, y, 600), true
}

// potTop is a clay pot from above: its round mouth, dark inside, in a thick rim.
func potTop(x, y int) (rgb, bool) {
	dx, dy := float64(x)-7.5, float64(y)-7.5
	switch d := math.Sqrt(dx*dx + dy*dy); {
	case d < 3.6:
		return grain(rgb{58, 36, 28}, 0.08, x, y, 601), true
	case d < 5.4:
		return grain(baked.scale(1.16), 0.04, x, y, 602), true
	case d < 6.2:
		return grain(baked.scale(0.74), 0.04, x, y, 603), true // the shadow under the rim
	}
	return grain(baked.scale(0.92), 0.05, x, y, 604), true
}

// potSide is a side of a clay pot: round, so darker toward its edges, with a rim at
// its top and a band scratched round its shoulder before it was baked.
func potSide(x, y int) (rgb, bool) {
	c := baked.scale(1.08 - 0.3*math.Pow(math.Abs(float64(x)-7.5)/7.5, 2))
	switch {
	case y < 2:
		c = c.scale(1.14)
	case y == 2 || y >= 14:
		c = c.scale(0.72)
	case y >= 5 && y <= 8 && (x+abs(y*2-13)/2)%4 == 0:
		c = rgb{232, 208, 170}.scale(1.04 - 0.3*math.Pow(math.Abs(float64(x)-7.5)/7.5, 2))
	case y == 4 || y == 9:
		c = c.scale(0.8)
	}
	return grain(c, 0.04, x, y, 605), true
}

// cookedMeat is a cut of meat browned at a fire, with the marks of the sticks it lay on.
func cookedMeat(x, y int) (rgb, bool) {
	dx, dy := (float64(x)-7.5)/6.5, (float64(y)-8.5)/4.8
	switch d := dx*dx + dy*dy; {
	case x >= 12 && y >= 3 && y <= 6 && x <= 14:
		return rgb{222, 206, 176}, true // the bone
	case d > 1:
		return rgb{}, false
	case d > 0.78:
		return grain(rgb{98, 58, 36}, 0.06, x, y, 606), true // seared at the edge
	case (x+y)%4 == 0:
		return grain(rgb{72, 44, 30}, 0.06, x, y, 607), true
	}
	return grain(rgb{156, 94, 52}, 0.08, x, y, 608), true
}

// seeds returns the painter of a few seeds of a stump palm, of a color, each with a
// mark of another color on it.
func seeds(shell, mark rgb) painter {
	return func(x, y int) (rgb, bool) {
		for i, at := range [][2]float64{{4.5, 5.5}, {10.5, 6.5}, {7, 11}} {
			dx, dy := (float64(x)+0.5-at[0])/2.9, (float64(y)+0.5-at[1])/2.3
			switch d := dx*dx + dy*dy; {
			case d > 1:
			case dx < -0.1 && dx > -0.6 && dy < 0 && dy > -0.6:
				return mark, true
			case d > 0.6:
				return grain(shell.scale(0.72), 0.05, x, y, 609+i), true
			default:
				return grain(shell, 0.06, x, y, 609+i), true
			}
		}
		return rgb{}, false
	}
}

// resin is a drop of resin gone hard: amber, and clear enough to shine.
func resin(x, y int) (rgb, bool) {
	dx, dy := float64(x)-7.5, float64(y)-9.5
	d := math.Sqrt(dx*dx + dy*dy)
	neck := y >= 2 && y < 7 && math.Abs(dx) <= 0.4*float64(y-1)
	switch {
	case d > 4.6 && !neck:
		return rgb{}, false
	case dx > -3 && dx < -1 && dy > -3 && dy < 0:
		return rgb{252, 228, 150}, true // where the light comes through
	case d > 3.5 || neck && d > 4.6 && math.Abs(dx) > 0.4*float64(y-1)-1:
		return grain(rgb{170, 100, 30}, 0.06, x, y, 612), true
	}
	return grain(rgb{228, 158, 50}, 0.06, x, y, 613), true
}

// unlitTorch is a torch that has not been lit: a stick with its head wound in fiber
// and dark with resin.
func unlitTorch(x, y int) (rgb, bool) {
	switch {
	case y >= 7 && (x == 7 || x == 8):
		return grain(rgb{118, 82, 48}, 0.08, x, y, 103), true
	case y >= 2 && y <= 6 && x >= 6 && x <= 9 && !(y == 2 && (x == 6 || x == 9)):
		c := rgb{104, 70, 34}
		if (x+y)%3 == 0 {
			c = rgb{196, 138, 56} // the resin, where it shows between the windings
		}
		return grain(c, 0.08, x, y, 614), true
	}
	return rgb{}, false
}

// fireDrill is a drill for making fire: a spindle that stands in a notch of a board,
// and the bow whose cord turns it.
func fireDrill(x, y int) (rgb, bool) {
	switch {
	case (x == 7 || x == 8) && y >= 11 && y <= 12:
		return charBlack, true // the notch, burnt black
	case y >= 12 && y <= 14 && x >= 2 && x <= 13:
		c := woodLight
		if y == 14 {
			c = woodDark
		}
		return grain(c, 0.07, x, y, 615), true
	case (x == 7 || x == 8) && y >= 1 && y <= 10:
		c := woodLight
		if x == 8 {
			c = woodDark
		}
		if y >= 6 && y <= 7 {
			c = rgb{214, 190, 132} // the cord, a turn round the spindle
		}
		return grain(c, 0.07, x, y, 616), true
	case y == 7 && x >= 2 && x <= 13 || y == 6 && (x == 2 || x == 13):
		return grain(rgb{214, 190, 132}, 0.05, x, y, 617), true // the cord of the bow
	case y == 4 && x >= 3 && x <= 12 || y == 5 && (x == 2 || x == 13 || x == 3 || x == 12):
		return grain(woodDark, 0.07, x, y, 618), true // the bow
	}
	return rgb{}, false
}

// hardSpear is a spear whose tip has been in the fire: black a good way down.
func hardSpear(x, y int) (rgb, bool) {
	c, ok := spear(x, y)
	if a, _ := shaft(x, y, -14, 14, 1.1); ok && a > 5 {
		c = grain(charBlack, 0.1, x, y, 619)
		if a <= 6.5 {
			c = grain(rgb{96, 60, 40}, 0.08, x, y, 620) // scorched, where the black ends
		}
	}
	return c, ok
}

// softPot is a pot shaped of wet clay, not yet baked: grey, and a little slumped.
func softPot(x, y int) (rgb, bool) {
	dx, dy := (float64(x)-7.5)/6, (float64(y)-9.5)/5
	body := dx*dx+dy*dy <= 1
	rim := y >= 2 && y <= 3 && x >= 4 && x <= 11
	neck := y >= 4 && y <= 5 && x >= 5 && x <= 10
	if !body && !rim && !neck {
		return rgb{}, false
	}
	c := rgb{150, 158, 170}.scale(1.1 - 0.34*math.Pow(math.Abs(float64(x)-7.5)/7.5, 1.5))
	switch {
	case rim && y == 2:
		c = c.scale(1.12)
	case neck:
		c = c.scale(0.84)
	case y >= 13:
		c = c.scale(0.8)
	}
	return grain(c, 0.04, x, y, 621), true
}

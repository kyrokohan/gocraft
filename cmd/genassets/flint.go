package main

import "math"

// The things of the stone age: what is gathered, and what is made of flint and wood.

var (
	flintDark = rgb{52, 58, 72}
	flintPale = rgb{222, 214, 194} // the chalky rind of a flint
	woodLight = rgb{150, 110, 66}
	woodDark  = rgb{104, 74, 44}
)

// flintSeam is stone with flints in it: dark glassy lumps, each in a pale rind.
func flintSeam(x, y int) (rgb, bool) {
	switch lump := blotch(x, y, 2, 550); {
	case lump > 0.72:
		c := flintDark
		if hash(x, y, 551) < 0.18 {
			c = c.scale(1.5) // a glint on the glass of it
		}
		return grain(c, 0.08, x, y, 552), true
	case lump > 0.66:
		return grain(flintPale, 0.05, x, y, 553), true
	}
	return stone(x, y)
}

// knappingTop is the top of a knapping stone: a flat boulder, hollowed by work, with
// the flakes struck off lying in the hollow.
func knappingTop(x, y int) (rgb, bool) {
	dx, dy := float64(x)-7.5, float64(y)-7.5
	d := math.Sqrt(dx*dx + dy*dy)
	c, _ := stone(x, y)
	switch {
	case d < 4.5 && hash(x, y, 554) < 0.22:
		return grain(flintDark, 0.1, x, y, 555), true // a flake
	case d < 5:
		c = c.scale(0.82) // the hollow
	case d < 6:
		c = c.scale(1.12) // its rim, worn smooth
	}
	return c, true
}

// knappingSide is its side: a boulder, darker toward the ground, with a few flakes that
// have fallen against its foot.
func knappingSide(x, y int) (rgb, bool) {
	if y >= 13 && hash(x/2, y, 556) < 0.25 {
		return grain(flintDark, 0.1, x, y, 555), true
	}
	c, _ := stone(x, y)
	return c.scale(1.08 - 0.02*float64(y)), true
}

// furBed is a bed of hides laid on the ground, fur up: brown, with the paler edge of
// the hide showing round it.
func furBed(x, y int) (rgb, bool) {
	if x < 1 || x > size-2 || y < 1 || y > size-2 || (x+y)%7 == 0 && (x < 2 || x > size-3 || y < 2 || y > size-3) {
		return rgb{}, false // the ground shows round it, at a ragged edge
	}
	if x < 3 || x > size-4 || y < 3 || y > size-4 {
		return grain(rgb{196, 170, 128}, 0.06, x, y, 557), true // the skin side, turned up at the edge
	}
	c := rgb{122, 86, 54}.scale(0.84 + 0.34*blotch(x, y, 4, 558))
	if hash(x, y/2, 559) < 0.2 {
		c = c.scale(1.25) // the fur lies in tufts
	}
	return grain(c, 0.08, x, y, 560), true
}

// fiber is a loose bunch of fern stalks, pulled and stripped.
func fiber(x, y int) (rgb, bool) {
	for i, lean := range []int{-2, -1, 0, 1, 2} {
		at := 5 + 2*i + lean*(8-y)/6
		if x == at && y >= 2+abs(lean) && y <= 13 {
			return grain(rgb{150, 168, 96}.scale(0.85+0.3*hash(i, 0, 561)), 0.08, x, y, 562), true
		}
	}
	if y >= 8 && y <= 9 && x >= 4 && x <= 12 {
		return grain(rgb{116, 132, 70}, 0.06, x, y, 563), true // the strand that ties it
	}
	return rgb{}, false
}

// cord is a coil of twisted cord.
func cord(x, y int) (rgb, bool) {
	dx, dy := float64(x)-7.5, (float64(y)-8)*1.25
	d := math.Sqrt(dx*dx + dy*dy)
	switch {
	case d >= 3 && d < 6.2:
		c := rgb{188, 160, 102}
		if (x+2*y)%3 == 0 {
			c = c.scale(0.74) // the twist
		}
		return grain(c, 0.05, x, y, 564), true
	case x >= 12 && y >= 11 && x-y == 1:
		return grain(rgb{188, 160, 102}, 0.05, x, y, 564), true // the loose end
	}
	return rgb{}, false
}

// flint is a flint as it is found: a dark glassy lump, part of its pale rind still on.
func flint(x, y int) (rgb, bool) {
	dx, dy := (float64(x)-7.5)/5.2, (float64(y)-8)/4.2
	d := dx*dx + dy*dy + 0.25*dx*dy
	switch {
	case d > 1:
		return rgb{}, false
	case d > 0.7 && x+y < 15:
		return grain(flintPale, 0.05, x, y, 553), true
	case x-y >= 0 && x-y <= 1 && d < 0.5:
		return grain(flintDark.scale(1.7), 0.05, x, y, 565), true // a face that catches the light
	}
	return grain(flintDark, 0.08, x, y, 552), true
}

// hide is the skin of a beast, stretched: fur in the middle, and four corners where the
// legs were.
func hide(x, y int) (rgb, bool) {
	dx, dy := math.Abs(float64(x)-7.5), math.Abs(float64(y)-7.5)
	body := dx < 5 && dy < 4.5
	leg := dx > 3.5 && dx < 6.8 && dy > 3 && dy < 6.8 && math.Abs(dx-dy) < 1.6
	switch {
	case !body && !leg:
		return rgb{}, false
	case dx > 4 || dy > 3.6 || leg && !body:
		return grain(rgb{196, 170, 128}, 0.06, x, y, 557), true // the edge of the skin
	}
	c := rgb{122, 86, 54}.scale(0.86 + 0.3*blotch(x, y, 4, 558))
	return grain(c, 0.08, x, y, 560), true
}

// handAxe is a flint knapped to a point and an edge, held in the hand: a teardrop,
// with the scars of the flakes struck from it.
func handAxe(x, y int) (rgb, bool) {
	fy := float64(y) - 1.5
	half := 0.9 + 0.42*fy // it widens from its point to its butt
	if fy > 9 {
		half = 4.7 * math.Sqrt(max(1-(fy-9)*(fy-9)/14, 0))
	}
	dx := float64(x) - 7.5
	if fy < 0 || math.Abs(dx) > half {
		return rgb{}, false
	}
	c := flintDark
	switch {
	case math.Abs(dx) > half-1:
		c = c.scale(1.6) // the edge
	case (x+y)%4 == 0:
		c = c.scale(1.25) // a flake scar
	case fy > 9 && hash(x, y, 566) < 0.5:
		c = flintPale.scale(0.9) // the rind left on the butt, for the palm
	}
	return grain(c, 0.07, x, y, 567), true
}

// shaft is a thing made of a straight stick that lies from the lower left to the upper
// right: a pixel is on it between from and to along it, and within half its width.
func shaft(x, y int, from, to, width float64) (along float64, on bool) {
	fx, fy := float64(x)+0.5, float64(y)+0.5
	a, p := fx-fy, fx+fy-16
	return a, a >= from && a <= to && math.Abs(p) <= width
}

// diggingStick is a stout stick with its end cut to a point and hardened dark.
func diggingStick(x, y int) (rgb, bool) {
	a, on := shaft(x, y, -12, 12, 1.1)
	if !on || a > 9 && math.Abs(float64(x+y)-15.5) > (12-a)/3*1.1 {
		return rgb{}, false
	}
	c := woodLight
	switch {
	case a > 8:
		c = rgb{60, 46, 36} // the point
	case x+y >= 16:
		c = woodDark
	}
	return grain(c, 0.08, x, y, 568), true
}

// club is a heavy length of wood, thick at the end it hits with.
func club(x, y int) (rgb, bool) {
	fx, fy := float64(x)+0.5, float64(y)+0.5
	a, p := fx-fy, fx+fy-16
	half := 1.0 + 2.6*ramp(-2, 9, a)
	if a < -12 || a > 12 || math.Abs(p) > half {
		return rgb{}, false
	}
	c := woodLight.scale(0.86 + 0.3*hash(x-y, 0, 569)) // the grain runs along it
	switch {
	case p > half-1:
		c = woodDark
	case a < -8 && int(a)%2 == 0:
		c = rgb{196, 170, 128} // the grip, wound with hide
	}
	return grain(c, 0.07, x, y, 570), true
}

// spear is a long thin shaft with a point.
func spear(x, y int) (rgb, bool) {
	a, on := shaft(x, y, -14, 14, 1.1)
	fx, fy := float64(x)+0.5, float64(y)+0.5
	p := math.Abs(fx + fy - 16)
	if a > 9 && a <= 14 && p <= (14-a)/5*2.2 {
		return grain(rgb{74, 56, 42}, 0.06, x, y, 571), true // the point, hardened
	}
	if !on || a > 9 {
		return rgb{}, false
	}
	c := woodLight
	if x+y >= 16 {
		c = woodDark
	}
	return grain(c, 0.08, x, y, 568), true
}

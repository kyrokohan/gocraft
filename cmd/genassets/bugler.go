package main

import "math"

// The bugler's picture is seven tiles side by side, which the renderer lays on the
// boxes it is built from: its back, its side, its belly, the side of its head, the end
// of its bill, a leg and its crest.
var buglerTiles = []painter{buglerBack, buglerSide, buglerBelly, buglerHead, buglerBill, buglerLeg, buglerCrest}

var (
	buglerHide   = rgb{122, 132, 74}
	buglerStripe = rgb{76, 90, 54}
	buglerCream  = rgb{216, 202, 152}
	buglerRed    = rgb{198, 86, 48}
)

// buglerSkin is its hide at a pixel: pebbled, with dark bands across the back that
// thin out down the sides.
func buglerSkin(x, y int, down float64, seed int) rgb {
	c := buglerHide.scale(0.88 + 0.24*blotch(x, y, 4, seed))
	band := math.Sin(float64(x) * math.Pi / 4) // a band every eight pixels along the body
	if band > 0.55+0.5*down {
		c = buglerStripe.mix(c, 0.25)
	}
	return grain(c, 0.06, x, y, seed+1)
}

func buglerBack(x, y int) (rgb, bool) {
	return buglerSkin(x, y, 0, 500), true
}

// buglerSide is the hide of its side: the bands of the back come down a way, and it
// pales to cream at the belly.
func buglerSide(x, y int) (rgb, bool) {
	down := float64(y) / float64(size-1)
	c := buglerSkin(x, y, down, 502)
	if edge := 10 + int(hash(x, 0, 504)*3); y >= edge {
		c = c.mix(buglerCream, min(float64(y-edge+1)/3, 1))
	}
	return c, true
}

func buglerBelly(x, y int) (rgb, bool) {
	return grain(buglerCream, 0.05, x, y, 505), true
}

// buglerHead is the side of its head, with its front at the right of the tile: an eye,
// and the flat bill of a duck.
func buglerHead(x, y int) (rgb, bool) {
	switch {
	case (x == 8 || x == 9) && (y == 4 || y == 5):
		return rgb{24, 22, 20}, true // the eye
	case x >= 7 && x <= 10 && y >= 3 && y <= 6:
		return rgb{226, 206, 120}, true // the ring round it
	case x >= 10 && y >= 8:
		// The bill, with the line of the mouth along it.
		if y == 11 {
			return rgb{70, 62, 44}, true
		}
		return grain(rgb{164, 150, 100}, 0.05, x, y, 506), true
	}
	return buglerSide(x, y)
}

// buglerBill is the end of the bill, with two nostrils.
func buglerBill(x, y int) (rgb, bool) {
	if y >= 4 && y <= 5 && (x == 5 || x == 10) {
		return rgb{60, 52, 38}, true
	}
	if y >= 11 {
		return grain(rgb{140, 126, 84}, 0.05, x, y, 507), true // the lower jaw
	}
	return grain(rgb{164, 150, 100}, 0.05, x, y, 506), true
}

// buglerLeg is a leg from the hip down: hide, then a dark foot with three toes.
func buglerLeg(x, y int) (rgb, bool) {
	if y >= 13 {
		if x%5 == 4 {
			return rgb{40, 38, 34}, true // between two toes
		}
		return grain(rgb{84, 82, 66}, 0.06, x, y, 508), true
	}
	return buglerSkin(x, y, 1, 509), true
}

// buglerCrest is the hollow crest it calls through: red, darker along its top.
func buglerCrest(x, y int) (rgb, bool) {
	c := buglerRed
	if y < 4 {
		c = c.scale(0.7)
	}
	return grain(c, 0.06, x, y, 510), true
}

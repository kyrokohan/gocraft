package main

// The two great fliers. Each picture is eight tiles laid out as the others are: back,
// side, belly, the side of the head, the beak, a leg, the skin of a wing, and one of its
// own: a galewing's crest, a stiltwing's neck.
var (
	galewingTiles  = []painter{galeBack, galeSide, galeBelly, galeHead, galeBeak, galeLeg, galeWing, galeCrest}
	stiltwingTiles = []painter{stiltBack, stiltSide, stiltBelly, stiltHead, stiltBeak, stiltLeg, stiltWing, stiltNeck}
)

func init() {
	giantTiles["galewing"], giantTiles["stiltwing"] = galewingTiles, stiltwingTiles
}

var (
	galeGrey  = rgb{196, 190, 176} // a galewing: the pale grey of a sea bird, dark at the tips of its wings
	galeDark  = rgb{74, 72, 78}
	galeWhite = rgb{232, 230, 220}
	galeBill  = rgb{226, 176, 70}
	galeRed   = rgb{198, 78, 50}  // its crest
	stiltTan  = rgb{168, 132, 96} // a stiltwing: the tan of the badlands, with a pale neck
	stiltDark = rgb{92, 66, 50}
	stiltPale = rgb{224, 208, 176}
	stiltBill = rgb{214, 196, 150}
)

// downy is the short down of a flier at a pixel: one color, flecked darker.
func downy(x, y int, base, dark rgb, seed int) rgb {
	c := base
	if hash(x, y/2, seed) > 0.8 {
		c = c.mix(dark, 0.3)
	}
	return grain(c, 0.05, x, y, seed+1)
}

// wingSkin is the skin of a wing at a pixel: pale by the body, at the left of the tile
// and its right alike, as both wings wear it, dark toward its hind edge, with the long
// bones of the fingers showing through.
func wingSkin(x, y int, skin, dark rgb, seed int) rgb {
	c := skin.mix(dark, 0.7*float64(y)/float64(size-1))
	if x%5 == 2 {
		c = c.scale(0.8) // a finger
	}
	return grain(c, 0.04, x, y, seed)
}

func galeBack(x, y int) (rgb, bool) { return downy(x, y, galeGrey, galeDark, 1200), true }

func galeSide(x, y int) (rgb, bool) {
	if y >= 9 {
		return downy(x, y, galeWhite, galeGrey, 1203), true
	}
	return downy(x, y, galeGrey, galeDark, 1204), true
}

func galeBelly(x, y int) (rgb, bool) { return downy(x, y, galeWhite, galeGrey, 1206), true }

// galeHead is the side of its head: an eye, and the root of the beak.
func galeHead(x, y int) (rgb, bool) {
	if c, ok := eyeOn(x, y, 8, 6, galeBill); ok {
		return c, true
	}
	if x >= 12 && y >= 7 && y <= 11 {
		return grain(galeBill, 0.04, x, y, 1207), true
	}
	return downy(x, y, galeGrey, galeDark, 1208), true
}

// galeBeak is its beak, the same all round: yellow, with the line of the mouth along it.
func galeBeak(x, y int) (rgb, bool) {
	if y == 9 {
		return darkMouth, true
	}
	return grain(galeBill.scale(1-0.2*float64(y)/float64(size-1)), 0.04, x, y, 1210), true
}

func galeLeg(x, y int) (rgb, bool) { return grain(galeDark, 0.05, x, y, 1213), true }

func galeWing(x, y int) (rgb, bool) { return wingSkin(x, y, galeGrey, galeDark, 1216), true }

func galeCrest(x, y int) (rgb, bool) {
	return grain(galeRed.scale(0.85+0.3*hash(x/3, y, 1219)), 0.04, x, y, 1220), true
}

func stiltBack(x, y int) (rgb, bool) { return downy(x, y, stiltTan, stiltDark, 1230), true }

func stiltSide(x, y int) (rgb, bool) {
	if y >= 10 {
		return downy(x, y, stiltPale, stiltTan, 1233), true
	}
	return downy(x, y, stiltTan, stiltDark, 1234), true
}

func stiltBelly(x, y int) (rgb, bool) { return downy(x, y, stiltPale, stiltTan, 1236), true }

// stiltHead is the side of its head, which is nearly all beak: a small eye at the back
// of it, the long line of the mouth, and a dark tip.
func stiltHead(x, y int) (rgb, bool) {
	if c, ok := eyeOn(x, y, 2, 5, rgb{226, 150, 60}); ok {
		return c, true
	}
	switch {
	case x < 5:
		return downy(x, y, stiltTan, stiltDark, 1237), true
	case y == 9:
		return darkMouth, true
	case x >= 13:
		return grain(stiltDark, 0.05, x, y, 1238), true
	}
	return grain(stiltBill, 0.04, x, y, 1239), true
}

func stiltBeak(x, y int) (rgb, bool) { return grain(stiltDark, 0.05, x, y, 1240), true }

func stiltLeg(x, y int) (rgb, bool) {
	return grain(stiltDark.mix(stiltTan, 0.3), 0.05, x, y, 1243), true
}

func stiltWing(x, y int) (rgb, bool) { return wingSkin(x, y, stiltTan, stiltDark, 1246), true }

func stiltNeck(x, y int) (rgb, bool) { return downy(x, y, stiltPale, stiltTan, 1249), true }

package main

// The three beasts of the sea. Each picture is eight tiles laid out as the others are:
// back, side, belly, the side of the head, the face, a flipper, and two of its own: a
// lakeneck's neck and tail, a leapfin's fin and tail, a deepmaw's fin and the side of
// its lower jaw.
var (
	lakeneckTiles = []painter{neckBack, neckSide, neckBelly, neckHead, neckFace, neckFlipper, neckNeck, neckTail}
	leapfinTiles  = []painter{finBack, finSide, finBelly, finHead, finFace, finFlipper, finFin, finTail}
	deepmawTiles  = []painter{mawBack, mawSide, mawBelly, mawHead, mawFace, mawFlipper, mawFin, mawJaw}
)

func init() {
	giantTiles["lakeneck"], giantTiles["leapfin"], giantTiles["deepmaw"] = lakeneckTiles, leapfinTiles, deepmawTiles
}

var (
	neckSlate = rgb{92, 116, 138} // a lakeneck: the grey blue of a lake under cloud
	neckDark  = rgb{58, 76, 96}
	neckPale  = rgb{214, 216, 196}
	finBlue   = rgb{70, 92, 120} // a leapfin: dark above and white under, with a clean line between
	finDark   = rgb{44, 58, 80}
	finWhite  = rgb{228, 232, 230}
	mawDeep   = rgb{44, 62, 70} // a deepmaw: the green black of deep water, barred pale
	mawBlack  = rgb{24, 34, 40}
	mawPale   = rgb{176, 184, 160}
)

// wetHide is the smooth hide of a beast of the sea at a pixel: one color above and one
// under, which meet at the row line, with a few spots of the dark above it.
func wetHide(x, y, line int, top, dark, under rgb, seed int) rgb {
	switch {
	case y > line:
		return grain(under, 0.04, x, y, seed)
	case y == line:
		return grain(top.mix(under, 0.5), 0.04, x, y, seed)
	}
	c := top.mix(dark, 0.5*(1-float64(y)/float64(line+1)))
	if hash(x/2, y/2, seed+1) > 0.82 {
		c = c.mix(dark, 0.6) // a spot
	}
	return grain(c, 0.05, x, y, seed+2)
}

func neckBack(x, y int) (rgb, bool) {
	return wetHide(x, y, size, neckSlate, neckDark, neckPale, 1100), true
}

func neckSide(x, y int) (rgb, bool) {
	return wetHide(x, y, 10, neckSlate, neckDark, neckPale, 1103), true
}

func neckBelly(x, y int) (rgb, bool) { return grain(neckPale, 0.05, x, y, 1106), true }

// neckHead is the side of its small head: an eye, and a mouth of thin teeth.
func neckHead(x, y int) (rgb, bool) {
	if c, ok := eyeOn(x, y, 7, 5, neckPale); ok {
		return c, true
	}
	if c, ok := jawsOn(x, y, 6, 10); ok {
		return c, true
	}
	return wetHide(x, y, 12, neckSlate, neckDark, neckPale, 1107), true
}

func neckFace(x, y int) (rgb, bool) {
	switch {
	case y == 5 && (x == 5 || x == 10):
		return darkMouth, true // its nostrils
	case y == 10 && x >= 3 && x <= 12:
		return darkMouth, true
	}
	return wetHide(x, y, 12, neckSlate, neckDark, neckPale, 1110), true
}

// flipperOf is a flipper at a pixel: dark, with a paler edge behind.
func flipperOf(x, y int, base, dark rgb, seed int) rgb {
	c := base.mix(dark, 0.4)
	if y >= size-3 {
		c = base.scale(1.2)
	}
	return grain(c, 0.05, x, y, seed)
}

func neckFlipper(x, y int) (rgb, bool) { return flipperOf(x, y, neckSlate, neckDark, 1113), true }

// neckNeck is its neck, the same all round: slate, with rings a little darker.
func neckNeck(x, y int) (rgb, bool) {
	c := neckSlate
	if y%4 == 0 {
		c = c.mix(neckDark, 0.45)
	}
	return grain(c, 0.05, x, y, 1116), true
}

func neckTail(x, y int) (rgb, bool) {
	return wetHide(x, y, 11, neckSlate, neckDark, neckPale, 1119), true
}

func finBack(x, y int) (rgb, bool) {
	return wetHide(x, y, size, finBlue, finDark, finWhite, 1130), true
}

func finSide(x, y int) (rgb, bool) { return wetHide(x, y, 8, finBlue, finDark, finWhite, 1133), true }

func finBelly(x, y int) (rgb, bool) { return grain(finWhite, 0.03, x, y, 1136), true }

// finHead is the side of its beak: the line of the mouth all along it.
func finHead(x, y int) (rgb, bool) {
	if y == 9 {
		return darkMouth, true
	}
	return wetHide(x, y, 8, finBlue, finDark, finWhite, 1137), true
}

func finFace(x, y int) (rgb, bool) { return wetHide(x, y, 8, finBlue, finDark, finWhite, 1140), true }

func finFlipper(x, y int) (rgb, bool) { return flipperOf(x, y, finBlue, finDark, 1143), true }

func finFin(x, y int) (rgb, bool) { return grain(finDark, 0.05, x, y, 1146), true }

// finTail is the side of its body by the tail: an eye is not there, so the same hide.
func finTail(x, y int) (rgb, bool) { return wetHide(x, y, 9, finBlue, finDark, finWhite, 1149), true }

// barred is a deepmaw's hide at a pixel: wetHide, with pale bars down it.
func barred(x, y, line int, seed int) rgb {
	c := wetHide(x, y, line, mawDeep, mawBlack, mawPale, seed)
	if y < line && (x+y/3)%6 == 0 {
		c = c.mix(mawPale, 0.3)
	}
	return c
}

func mawBack(x, y int) (rgb, bool) { return barred(x, y, size, 1160), true }

func mawSide(x, y int) (rgb, bool) { return barred(x, y, 11, 1163), true }

func mawBelly(x, y int) (rgb, bool) { return grain(mawPale, 0.05, x, y, 1166), true }

// mawHead is the side of its skull: a small eye far back, and teeth along the edge of
// the upper jaw.
func mawHead(x, y int) (rgb, bool) {
	if c, ok := eyeOn(x, y, 3, 5, rgb{226, 196, 84}); ok {
		return c, true
	}
	switch {
	case y == size-3:
		return darkMouth, true
	case y >= size-2 && x%3 != 0:
		return tooth, true
	case y >= size-2:
		return darkMouth, true
	}
	return barred(x, y, size, 1167), true
}

// mawFace is the end of its snout from in front: nostrils, and teeth that mesh.
func mawFace(x, y int) (rgb, bool) {
	switch {
	case y == 4 && (x == 5 || x == 10):
		return darkMouth, true
	case y >= size-3 && x%2 == 0:
		return tooth, true
	case y >= size-3:
		return darkMouth, true
	}
	return barred(x, y, size, 1170), true
}

func mawFlipper(x, y int) (rgb, bool) { return flipperOf(x, y, mawDeep, mawBlack, 1173), true }

func mawFin(x, y int) (rgb, bool) { return grain(mawBlack.mix(mawDeep, 0.4), 0.05, x, y, 1176), true }

// mawJaw is the side of its lower jaw: teeth that stand up from the top edge of it.
func mawJaw(x, y int) (rgb, bool) {
	switch {
	case y < 2 && x%3 != 0:
		return tooth, true
	case y < 3:
		return darkMouth, true
	}
	return grain(mawPale.mix(mawDeep, 0.35), 0.05, x, y, 1179), true
}

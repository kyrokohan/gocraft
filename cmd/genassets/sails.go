package main

// The two beasts with a sail on the back. Each picture is eight tiles laid out as the
// others are: back, side, belly, the side of the head, the face, a leg, the sail, and
// the tail.
var (
	sunbackTiles = []painter{sunBack, sunSide, sunBelly, sunHead, sunFace, sunLeg, sunSail, sunTail}
	sailjawTiles = []painter{jawBack, jawSide, jawBelly, jawHead, jawFace, jawLeg, jawSail, jawTail}
)

func init() {
	giantTiles["sunback"], giantTiles["sailjaw"] = sunbackTiles, sailjawTiles
}

var (
	sunHide  = rgb{112, 122, 70} // a sunback: olive, in small scales
	sunDark  = rgb{66, 76, 48}
	sunCream = rgb{206, 196, 140}
	sunFire  = rgb{222, 104, 44} // its sail, which takes the sun
	jawHide  = rgb{84, 104, 100} // a sailjaw: the grey green of deep water
	jawDark  = rgb{48, 62, 64}
	jawCream = rgb{198, 204, 180}
	jawBlue  = rgb{96, 128, 150} // its sail
	tooth    = rgb{232, 226, 204}
)

// scaly is a hide of small scales at a pixel: rows of them, each darker at its lower
// edge, and darker along the back.
func scaly(x, y int, base, dark rgb, down float64, seed int) rgb {
	c := base.mix(dark, 0.55*(1-down)).scale(0.9 + 0.2*hash(x/2, y/2, seed))
	if (x+y/2*2)%3 == 0 && y%2 == 1 {
		c = c.scale(0.78)
	}
	return grain(c, 0.04, x, y, seed+1)
}

// sailOf is a sail at a pixel: skin stretched over long spines that stand up the tile,
// brighter toward its top edge, with a dark rim.
func sailOf(x, y int, skin rgb, seed int) rgb {
	c := skin.scale(0.75 + 0.45*(1-float64(y)/float64(size-1)))
	switch {
	case x%4 == 1:
		c = c.scale(0.62) // a spine
	case y < 2:
		c = c.scale(0.55) // its rim
	}
	return grain(c, 0.04, x, y, seed)
}

// jawsOn draws a row of teeth along the line of a mouth, from x0 to the right edge of a
// tile, at the row y0: the line dark, and a tooth below it at every other pixel.
func jawsOn(x, y, x0, y0 int) (rgb, bool) {
	switch {
	case x < x0:
	case y == y0:
		return darkMouth, true
	case y == y0+1 && x%2 == 0:
		return tooth, true
	}
	return rgb{}, false
}

func sunBack(x, y int) (rgb, bool) { return scaly(x, y, sunHide, sunDark, 0, 1000), true }

func sunSide(x, y int) (rgb, bool) {
	down := float64(y) / float64(size-1)
	c := scaly(x, y, sunHide, sunDark, down, 1003)
	if y >= 12 {
		c = c.mix(sunCream, 0.7)
	}
	return c, true
}

func sunBelly(x, y int) (rgb, bool) { return grain(sunCream, 0.05, x, y, 1006), true }

// sunHead is the side of its head: a deep jaw full of teeth, and a small eye set high.
func sunHead(x, y int) (rgb, bool) {
	if c, ok := eyeOn(x, y, 6, 4, sunCream); ok {
		return c, true
	}
	if c, ok := jawsOn(x, y, 5, 10); ok {
		return c, true
	}
	return scaly(x, y, sunHide, sunDark, float64(y)/float64(size-1), 1007), true
}

func sunFace(x, y int) (rgb, bool) {
	switch {
	case y == 6 && (x == 5 || x == 10):
		return darkMouth, true // its nostrils
	case y == 11 && x >= 3 && x <= 12:
		return darkMouth, true
	case y == 12 && x >= 3 && x <= 12 && x%2 == 1:
		return tooth, true
	}
	return scaly(x, y, sunHide, sunDark, 0.4, 1010), true
}

func sunLeg(x, y int) (rgb, bool) {
	if y >= 14 && x%4 != 0 {
		return grain(rgb{58, 52, 44}, 0.05, x, y, 1013), true // its claws
	}
	return scaly(x, y, sunHide, sunDark, 0.6, 1014), true
}

func sunSail(x, y int) (rgb, bool) { return sailOf(x, y, sunFire, 1017), true }

func sunTail(x, y int) (rgb, bool) { return scaly(x, y, sunHide, sunDark, 0.3, 1020), true }

func jawBack(x, y int) (rgb, bool) { return scaly(x, y, jawHide, jawDark, 0, 1030), true }

func jawSide(x, y int) (rgb, bool) {
	down := float64(y) / float64(size-1)
	c := scaly(x, y, jawHide, jawDark, down, 1033)
	if y >= 11 {
		c = c.mix(jawCream, 0.75)
	}
	return c, true
}

func jawBelly(x, y int) (rgb, bool) { return grain(jawCream, 0.05, x, y, 1036), true }

// jawHead is the side of its long jaws: an eye at the back of them, and teeth all the
// way along.
func jawHead(x, y int) (rgb, bool) {
	if c, ok := eyeOn(x, y, 2, 4, rgb{214, 190, 90}); ok {
		return c, true
	}
	if c, ok := jawsOn(x, y, 1, 9); ok {
		return c, true
	}
	if y >= 11 {
		return grain(jawCream, 0.05, x, y, 1036), true
	}
	return scaly(x, y, jawHide, jawDark, float64(y)/float64(size-1), 1037), true
}

// jawFace is the end of its snout from in front: two nostrils, and teeth that mesh.
func jawFace(x, y int) (rgb, bool) {
	switch {
	case y == 4 && (x == 5 || x == 10):
		return darkMouth, true
	case y == 9:
		return darkMouth, true
	case (y == 8 || y == 10) && x%2 == 0:
		return tooth, true
	}
	return scaly(x, y, jawHide, jawDark, 0.4, 1040), true
}

func jawLeg(x, y int) (rgb, bool) {
	if y >= 13 && x%4 != 0 {
		return grain(rgb{52, 50, 46}, 0.05, x, y, 1043), true
	}
	return scaly(x, y, jawHide, jawDark, 0.6, 1044), true
}

// jawSail is its sail: blue grey, with a darker band across it.
func jawSail(x, y int) (rgb, bool) {
	c := sailOf(x, y, jawBlue, 1047)
	if y >= 6 && y <= 8 {
		c = c.scale(0.75)
	}
	return c, true
}

func jawTail(x, y int) (rgb, bool) { return scaly(x, y, jawHide, jawDark, 0.3, 1050), true }

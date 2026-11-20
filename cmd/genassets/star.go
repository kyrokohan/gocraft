package main

import "math"

// What a fallen star leaves: the core of sky iron at the bottom of its crater, hot and
// then dark, and the lumps broken out of it.

var skyIron = rgb{112, 116, 130} // grey, with the blue of cold metal

// pitted is sky iron at a pixel: grey metal, with the dark thumbprints a star's flight
// leaves on it and a glint here and there.
func pitted(x, y, seed int) rgb {
	c := skyIron.scale(0.8 + 0.4*blotch(x, y, 4, seed))
	switch h := hash(x, y, seed+1); {
	case blotch(x, y, 2, seed+2) > 0.78:
		c = c.scale(0.5) // a pit
	case h < 0.05:
		c = rgb{222, 228, 240} // a glint
	}
	return grain(c, 0.06, x, y, seed+3)
}

// skyIronCore is the core gone dark: pitted metal all through.
func skyIronCore(x, y int) (rgb, bool) {
	return pitted(x, y, 800), true
}

// hotCore is the core as the star left it: it glows from within, orange to yellow, under
// a crust that has cracked dark.
func hotCore(x, y int) (rgb, bool) {
	c := rgb{255, 150, 40}.mix(rgb{255, 232, 130}, blotch(x, y, 4, 805))
	if blotch(x, y, 2, 806) > 0.7 || hash(x, y, 807) > 0.94 {
		c = rgb{96, 40, 30} // the crust
	}
	return grain(c, 0.05, x, y, 808), true
}

// skyIronLump is a lump of sky iron broken out of a core: a rough piece of pitted metal.
func skyIronLump(x, y int) (rgb, bool) {
	dx, dy := (float64(x)-7.5)/5.6, (float64(y)-8.5)/4.6
	edge := 1 + 0.22*math.Sin(3*math.Atan2(dy, dx)+1.3) // rough, and not round
	if far := math.Hypot(dx, dy); far > edge {
		return rgb{}, false
	}
	c := pitted(x, y, 810)
	if dy < -0.35 {
		c = c.scale(1.25) // the top catches the light
	}
	return c, true
}

// skyIronSpear is a spear with a point of sky iron bound to it.
func skyIronSpear(x, y int) (rgb, bool) {
	c, ok := spear(x, y)
	a, _ := shaft(x, y, -14, 14, 1.1)
	switch {
	case !ok:
	case a > 8.5:
		c = pitted(x, y, 815).scale(1.35)
	case a > 6.5:
		c = grain(rgb{196, 168, 110}, 0.06, x, y, 816) // the cord that binds it
	}
	return c, ok
}

// furCloak is a cloak of hides: wide at the shoulders, with a collar of thicker fur and a
// cord at the throat, and the hair of the hides in strokes down it.
func furCloak(x, y int) (rgb, bool) {
	fx, fy := float64(x)-7.5, float64(y)
	wide := 3.2 + 0.36*fy // it widens as it hangs
	if fy < 1 || fy > 14.5 || math.Abs(fx) > wide || fy > 13 && int(math.Abs(fx)+0.5)%3 == 0 {
		return rgb{}, false // the ragged hem
	}
	c := rgb{124, 86, 52}.scale(0.82 + 0.36*hash(x, y/3, 820))
	switch {
	case fy < 4:
		c = rgb{196, 176, 140}.scale(0.85 + 0.3*hash(x, y, 821)) // the collar
	case math.Abs(fx) < 0.6 && fy < 7:
		c = rgb{214, 190, 130} // the cord that ties it
	case math.Abs(fx) > wide-1:
		c = c.scale(0.7) // its edges, in shadow
	}
	return grain(c, 0.06, x, y, 822), true
}

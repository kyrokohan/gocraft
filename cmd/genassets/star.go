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

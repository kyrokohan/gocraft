package main

// The eggs a player can steal, each a thing to carry: an egg on end, with the marks of
// its kind.

var (
	buglerShell     = rgb{222, 206, 164} // a bugler's: sand, speckled brown
	buglerSpot      = rgb{132, 96, 62}
	shieldbrowShell = rgb{188, 176, 150} // a shieldbrow's: grey stone, blotched
	shieldbrowSpot  = rgb{104, 96, 84}
	galewingShell   = rgb{214, 226, 226} // a galewing's: pale sea blue, flecked dark
	galewingSpot    = rgb{70, 84, 100}
)

// eggOf is an egg at a pixel: wider below its middle than above, lit from the upper
// left, with spots where a hash says so: so many in a hundred pixels.
func eggOf(x, y int, shell, spot rgb, spots float64, seed int) (rgb, bool) {
	dy := (float64(y) - 8.5) / 6.6
	wide := 4.9 * (1 + 0.18*dy) // an egg, and not a ball
	dx := (float64(x) - 7.5) / wide
	far := dx*dx + dy*dy
	if far > 1 {
		return rgb{}, false
	}
	c := shell.scale(1.08 - 0.32*far - 0.1*(dx+dy))
	switch {
	case far > 0.8:
		c = shell.scale(0.6) // its edge
	case hash(x, y, seed) < spots:
		c = spot
	}
	return grain(c, 0.03, x, y, seed+1), true
}

func buglerEgg(x, y int) (rgb, bool) { return eggOf(x, y, buglerShell, buglerSpot, 0.16, 1300) }

func shieldbrowEgg(x, y int) (rgb, bool) {
	return eggOf(x/2*2, y/2*2, shieldbrowShell, shieldbrowSpot, 0.3, 1303) // blotches, two pixels wide
}

func galewingEgg(x, y int) (rgb, bool) { return eggOf(x, y, galewingShell, galewingSpot, 0.1, 1306) }

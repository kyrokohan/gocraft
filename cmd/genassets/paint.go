package main

import "math"

// What a painting is made of: the rock that ochre and chalk lie in, the pieces they
// break into, and the paints ground from them, each a heap of powder on a flat stone.
var (
	ochreRed    = rgb{164, 66, 38}
	ochreYellow = rgb{212, 148, 64}
	chalkWhite  = rgb{230, 226, 212}
	charcoal    = rgb{38, 34, 32}
)

func ochreSeam(x, y int) (rgb, bool) {
	// A soft earth, red and yellow in streaks that lie along the bed.
	c := ochreRed.mix(ochreYellow, blotch(x, y/3+x/5, 3, 980))
	if hash(x, y, 981) < 0.06 {
		c = c.scale(0.78)
	}
	return grain(c, 0.06, x, y, 982), true
}

func chalkSeam(x, y int) (rgb, bool) {
	// Soft white rock, with grey where it is damp, and here and there a speck of flint.
	c := chalkWhite.scale(0.9 + 0.12*blotch(x, y, 4, 983))
	switch {
	case hash(x, y, 984) < 0.03:
		c = flintDark
	case blotch(x, y, 3, 985) > 0.8:
		c = c.scale(0.86)
	}
	return grain(c, 0.03, x, y, 986), true
}

// lump returns the painter of a rounded piece of something soft, of a color.
func lump(c rgb, seed int) painter {
	return func(x, y int) (rgb, bool) {
		dx, dy := (float64(x)-7.5)/5.4, (float64(y)-8.5)/4.4
		far := dx*dx + dy*dy + 0.2*dx*dy
		if far > 1 {
			return rgb{}, false
		}
		shade := c.scale(1.12 - 0.3*far)
		if dx > 0.2 && dy > 0.2 {
			shade = shade.scale(0.8) // the side away from the light
		}
		return grain(shade, 0.07, x, y, seed), true
	}
}

// paint returns the painter of a paint of a color: a heap of it on a flat grey stone.
func paint(c rgb, seed int) painter {
	return func(x, y int) (rgb, bool) {
		dx := float64(x) - 7.5
		switch {
		case y >= 10 && y <= 12 && math.Abs(dx) < 7-float64(y-10):
			s, _ := stone(x, y)
			return s.scale(0.95 - 0.12*float64(y-10)), true // the stone it is ground on
		case y < 10 && math.Abs(dx) < 1.2*float64(y-3) && y > 3:
			return grain(c.scale(0.86+0.03*float64(y)), 0.08, x, y, seed), true // the heap
		}
		return rgb{}, false
	}
}

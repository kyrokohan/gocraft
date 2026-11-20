package main

import "math"

// The beasts of the cold steppe. Each picture is eight tiles side by side, laid out as
// the armed plant-eaters' are: its back, its side, its belly, the side of its head with
// its front at the right of the tile, its face, a leg, and two more of its own.
var (
	snowtuskTiles = []painter{tuskBack, tuskSide, tuskBelly, tuskHead, tuskFace, tuskLeg, tuskTrunk, horn}
	slowclawTiles = []painter{clawBearBack, clawBearSide, clawBearBelly, clawBearHead, clawBearFace, clawBearLeg, clawBearArm, clawBearClaw}
)

func init() {
	giantTiles["snowtusk"], giantTiles["slowclaw"] = snowtuskTiles, slowclawTiles
}

var (
	tuskHair  = rgb{122, 82, 52} // the long hair of a snowtusk: red brown
	tuskDark  = rgb{70, 46, 34}
	tuskSkin  = rgb{104, 92, 84} // where it has none: its trunk, and round its eye
	slowFur   = rgb{168, 140, 96}
	slowDark  = rgb{104, 82, 58}
	slowCream = rgb{214, 196, 156}
)

// shaggy is long hair at a pixel: strands that hang, each its own shade, darker toward
// their ends, with a gap between two here and there.
func shaggy(x, y int, base, dark rgb, down float64, seed int) rgb {
	c := base.mix(dark, 0.55*down).scale(0.8 + 0.4*hash(x, 0, seed))
	if hash(x, y/5, seed+1) < 0.14 {
		c = c.scale(0.72) // the shadow between two strands
	}
	return grain(c, 0.05, x, y, seed+2)
}

func tuskBack(x, y int) (rgb, bool) { return shaggy(y, x, tuskHair, tuskDark, 0, 840), true }

// tuskSide is its side: hair that hangs to a ragged skirt.
func tuskSide(x, y int) (rgb, bool) {
	return shaggy(x, y, tuskHair, tuskDark, float64(y)/float64(size-1), 843), true
}

func tuskBelly(x, y int) (rgb, bool) { return shaggy(x, y, tuskDark, tuskDark, 0.4, 846), true }

// tuskHead is the side of its head: a dome of hair, a small eye in bare skin, and a
// little ear.
func tuskHead(x, y int) (rgb, bool) {
	if c, ok := eyeOn(x, y, 9, 7, tuskSkin.scale(1.2)); ok {
		return c, true
	}
	if x >= 2 && x <= 5 && y >= 5 && y <= 10 {
		return grain(tuskDark, 0.05, x, y, 849), true // the ear
	}
	return shaggy(x, y, tuskHair, tuskDark, float64(y)/float64(size-1), 850), true
}

// tuskFace is its face from in front: hair over the brow, and the root of the trunk
// between the eyes.
func tuskFace(x, y int) (rgb, bool) {
	switch {
	case y >= 6 && x >= 5 && x <= 10:
		c := tuskSkin.scale(0.85 + 0.3*hash(0, y/2, 853))
		return grain(c, 0.05, x, y, 854), true
	case y >= 6 && y <= 7 && (x == 3 || x == 12):
		return rgb{22, 20, 18}, true
	}
	return shaggy(x, y, tuskHair, tuskDark, float64(y)/float64(size-1)*0.6, 855), true
}

// tuskLeg is a leg like a pillar: hair to the foot, and three pale nails.
func tuskLeg(x, y int) (rgb, bool) {
	if y >= 13 && x%5 >= 1 && x%5 <= 3 {
		return grain(rgb{198, 186, 160}, 0.05, x, y, 858), true
	}
	return shaggy(x, y, tuskHair, tuskDark, 0.5+0.5*float64(y)/float64(size-1), 859), true
}

// tuskTrunk is its trunk: bare skin in rings, a little hair down the top of it.
func tuskTrunk(x, y int) (rgb, bool) {
	c := tuskSkin.scale(0.9 + 0.2*hash(x, y, 862))
	if y%3 == 2 {
		c = c.scale(0.8)
	}
	return grain(c, 0.05, x, y, 863), true
}

func clawBearBack(x, y int) (rgb, bool) { return shaggy(y, x, slowFur, slowDark, 0.1, 870), true }

func clawBearSide(x, y int) (rgb, bool) {
	return shaggy(x, y, slowFur, slowDark, float64(y)/float64(size-1), 873), true
}

func clawBearBelly(x, y int) (rgb, bool) { return grain(slowCream, 0.06, x, y, 876), true }

// clawBearHead is the side of its head: a long muzzle, pale, with a dark nose at its
// end, a small eye and a round ear.
func clawBearHead(x, y int) (rgb, bool) {
	if c, ok := eyeOn(x, y, 6, 5, slowCream); ok {
		return c, true
	}
	switch {
	case x >= 13 && y >= 7 && y <= 10:
		return rgb{40, 32, 28}, true // the nose
	case x >= 9 && y >= 6 && y <= 12:
		if y == 11 && x < 14 {
			return rgb{60, 46, 38}, true // the line of the mouth
		}
		return grain(slowCream, 0.05, x, y, 877), true
	case x >= 1 && x <= 3 && y >= 2 && y <= 5:
		return grain(slowDark, 0.05, x, y, 878), true // the ear
	}
	return shaggy(x, y, slowFur, slowDark, float64(y)/float64(size-1)*0.7, 879), true
}

// clawBearFace is its face from in front: the muzzle, the nose, and two small eyes.
func clawBearFace(x, y int) (rgb, bool) {
	switch {
	case y >= 9 && y <= 11 && x >= 6 && x <= 9:
		return rgb{40, 32, 28}, true
	case y >= 7 && x >= 4 && x <= 11:
		return grain(slowCream, 0.05, x, y, 877), true
	case y >= 4 && y <= 5 && (x == 4 || x == 11):
		return rgb{22, 20, 18}, true
	}
	return shaggy(x, y, slowFur, slowDark, 0.3, 882), true
}

func clawBearLeg(x, y int) (rgb, bool) {
	if y >= 14 {
		return grain(rgb{58, 48, 42}, 0.06, x, y, 885), true // the sole
	}
	return shaggy(x, y, slowFur, slowDark, 0.4+0.6*float64(y)/float64(size-1), 886), true
}

// clawBearArm is an arm: long fur, darker toward the hand.
func clawBearArm(x, y int) (rgb, bool) {
	return shaggy(x, y, slowFur, slowDark, 0.2+0.8*float64(y)/float64(size-1), 889), true
}

// clawBearClaw is a claw: dark horn, paler along its edge.
func clawBearClaw(x, y int) (rgb, bool) {
	c := rgb{66, 54, 46}.scale(0.85 + 0.4*float64(y)/float64(size-1))
	if x < 3 {
		c = c.scale(1.5)
	}
	return grain(c, 0.05, x, y, 892), true
}

// tusk is a tusk of a snowtusk: a long curve of ivory, dark at its root.
func tusk(x, y int) (rgb, bool) {
	// A quarter of a ring, from the bottom left of the tile up to its top right.
	far := math.Hypot(float64(x)-1.5, float64(y)-1.5)
	if far < 9.5 || far > 12.8-2.2*float64(x)/float64(size-1) || x < 2 || y < 2 {
		return rgb{}, false
	}
	c := ivory.scale(0.84 + 0.2*float64(x)/float64(size-1))
	if y > 11 && x < 6 {
		c = rgb{112, 92, 74} // the root
	}
	return grain(c, 0.04, x, y, 895), true
}

// The hunters of the steppe, laid out as the others are: back, side, belly, the side of
// the head, the face, a leg, the tail, and an ear or a fang.
var (
	howlerTiles    = []painter{howlBack, howlSide, howlBelly, howlHead, howlFace, howlLeg, howlTail, howlEar}
	daggercatTiles = []painter{catBack, catSide, catBelly, catHead, catFace, catLeg, catTail, horn}
)

func init() {
	giantTiles["howler"], giantTiles["daggercat"] = howlerTiles, daggercatTiles
}

var (
	howlFur   = rgb{142, 146, 152} // a howler: grey, paler under
	howlDark  = rgb{78, 80, 88}
	howlPale  = rgb{216, 218, 220}
	catFur    = rgb{196, 156, 96} // a daggercat: tawny, with faint spots
	catDark   = rgb{128, 92, 56}
	catCream  = rgb{232, 216, 180}
	catNose   = rgb{92, 56, 50}
	darkMouth = rgb{44, 34, 32}
)

// fur is short thick fur at a pixel: strands as shaggy hair has, but close, so that one
// is little darker than the next.
func fur(x, y int, base, dark rgb, down float64, seed int) rgb {
	c := base.mix(dark, 0.5*down).scale(0.92 + 0.16*hash(x, y/4, seed))
	return grain(c, 0.05, x, y, seed+2)
}

func howlBack(x, y int) (rgb, bool) { return fur(y, x, howlDark, howlDark, 0, 900), true }

// howlSide is its side: dark along the back, and pale toward the belly.
func howlSide(x, y int) (rgb, bool) {
	down := float64(y) / float64(size-1)
	return fur(x, y, howlDark.mix(howlFur, min(2*down, 1)).mix(howlPale, max(2*down-1, 0)), howlDark, 0, 903), true
}

func howlBelly(x, y int) (rgb, bool) { return grain(howlPale, 0.05, x, y, 906), true }

// howlHead is the side of its head: a long muzzle, pale under, a dark nose, and an eye
// of pale blue.
func howlHead(x, y int) (rgb, bool) {
	if c, ok := eyeOn(x, y, 5, 5, rgb{170, 210, 240}); ok {
		return c, true
	}
	switch {
	case x >= 14 && y >= 6 && y <= 8:
		return darkMouth, true // the nose
	case y == 11 && x >= 8:
		return darkMouth, true // the line of the mouth
	case y >= 9:
		return grain(howlPale, 0.05, x, y, 907), true
	}
	return fur(x, y, howlFur, howlDark, 0.2, 908), true
}

// howlFace is its face from in front: a pale muzzle, the nose, and the mask of its brow.
func howlFace(x, y int) (rgb, bool) {
	switch {
	case y >= 8 && y <= 10 && x >= 6 && x <= 9:
		return darkMouth, true
	case y >= 6 && x >= 4 && x <= 11:
		return grain(howlPale, 0.05, x, y, 907), true
	}
	return fur(x, y, howlFur, howlDark, 0.3, 911), true
}

func howlLeg(x, y int) (rgb, bool) {
	if y >= 14 {
		return grain(howlDark, 0.05, x, y, 914), true // the paw
	}
	return fur(x, y, howlFur, howlDark, 0.3*float64(y)/float64(size-1), 915), true
}

// howlTail is its brush of a tail: grey, tipped dark.
func howlTail(x, y int) (rgb, bool) {
	return fur(x, y, howlFur, howlDark, float64(y)/float64(size-1), 918), true
}

func howlEar(x, y int) (rgb, bool) { return grain(howlDark, 0.06, x, y, 921), true }

// catPelt is a daggercat's coat at a pixel: short, tawny, with faint darker spots.
func catPelt(x, y int, down float64, seed int) rgb {
	c := catFur.mix(catCream, 0.6*down).scale(0.9 + 0.2*blotch(x, y, 4, seed))
	if blotch(x, y, 2, seed+1) > 0.74 && down < 0.7 {
		c = c.mix(catDark, 0.45) // a spot
	}
	return grain(c, 0.04, x, y, seed+2)
}

func catBack(x, y int) (rgb, bool) { return catPelt(x, y, 0, 925), true }

func catSide(x, y int) (rgb, bool) {
	return catPelt(x, y, float64(y)/float64(size-1), 928), true
}

func catBelly(x, y int) (rgb, bool) { return grain(catCream, 0.05, x, y, 931), true }

// catHead is the side of its head: a short broad muzzle, a nose, an eye of amber, and
// the dark line of a mouth its fangs hang from.
func catHead(x, y int) (rgb, bool) {
	if c, ok := eyeOn(x, y, 7, 5, rgb{236, 180, 60}); ok {
		return c, true
	}
	switch {
	case x >= 13 && y >= 7 && y <= 9:
		return catNose, true
	case y == 12 && x >= 8:
		return darkMouth, true
	case y >= 10 && x >= 8:
		return grain(catCream, 0.05, x, y, 932), true
	}
	return catPelt(x, y, 0.3, 933), true
}

// catFace is its face from in front: two eyes, a nose, and the pale muzzle.
func catFace(x, y int) (rgb, bool) {
	switch {
	case y >= 4 && y <= 5 && (x >= 3 && x <= 4 || x >= 11 && x <= 12):
		return rgb{236, 180, 60}, true
	case y >= 8 && y <= 9 && x >= 6 && x <= 9:
		return catNose, true
	case y >= 8 && x >= 3 && x <= 12:
		return grain(catCream, 0.05, x, y, 932), true
	}
	return catPelt(x, y, 0.2, 936), true
}

func catLeg(x, y int) (rgb, bool) {
	if y >= 14 {
		return grain(catCream, 0.05, x, y, 939), true // the paw
	}
	return catPelt(x, y, 0.5, 940), true
}

// catTail is its short tail: ringed, with a dark tip at the bottom of the tile.
func catTail(x, y int) (rgb, bool) {
	c := catPelt(x, y, 0.2, 943)
	if y%5 == 4 || y >= 13 {
		c = c.mix(catDark, 0.6)
	}
	return c, true
}

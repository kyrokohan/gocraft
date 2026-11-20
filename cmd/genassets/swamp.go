package main

import "math"

// The coal swamp: peat, which is its ground; the scale tree, whose bark is all diamonds
// where old leaves grew; and amber.

// peat is black earth, wet, with the pale threads of old roots in it.
func peat(x, y int) (rgb, bool) {
	c := rgb{46, 36, 30}.scale(0.82 + 0.4*blotch(x, y, 4, 950))
	switch h := hash(x, y, 951); {
	case h < 0.06:
		c = rgb{102, 84, 60} // a root
	case h > 0.93:
		c = c.scale(0.6)
	}
	return grain(c, 0.08, x, y, 952), true
}

// scaleTrunkSide is the bark of a scale tree: diamonds in rows, each with the scar of a
// leaf at its middle, grey green.
func scaleTrunkSide(x, y int) (rgb, bool) {
	// Rows of diamonds four wide and four high, each row half a diamond across from the last.
	row := y / 4
	u, v := (x+2*(row%2))%4, y%4
	c := rgb{88, 104, 82}.scale(0.88 + 0.24*hash(x/4, row, 955))
	switch {
	case abs(u-2)+abs(v-2) >= 3:
		c = rgb{52, 62, 52} // the groove between two
	case u == 2 && v == 2:
		c = rgb{148, 150, 112} // the scar
	}
	return grain(c, 0.06, x, y, 956), true
}

// scaleTrunkTop is the trunk cut across: bark, and soft pale pith inside it.
func scaleTrunkTop(x, y int) (rgb, bool) {
	d := math.Hypot(float64(x)-7.5, float64(y)-7.5)
	switch {
	case d > 6.4:
		return grain(rgb{60, 72, 58}, 0.08, x, y, 957), true
	case d > 5:
		return grain(rgb{96, 108, 84}, 0.06, x, y, 958), true
	}
	return grain(rgb{186, 176, 132}, 0.06, x, y, 959), true
}

// scaleCrown is the crown of a scale tree: long thin leaves like grass, that hang.
func scaleCrown(x, y int) (rgb, bool) {
	if hash(x, 0, 960) < 0.3 && y%7 > 1 || hash(x, y, 961) < 0.1 {
		return rgb{}, false
	}
	c := rgb{96, 150, 78}.scale(0.8 + 0.4*hash(x, 0, 962))
	if y > 11 {
		c = c.scale(0.8) // the ends, in shadow
	}
	return grain(c, 0.08, x, y, 963), true
}

// amberPiece is a piece of amber: a drop of clear gold, with a bright edge and a dark
// speck caught in it.
func amberPiece(x, y int) (rgb, bool) {
	dx, dy := (float64(x)-7.5)/4.4, (float64(y)-8.5)/5.4
	far := dx*dx + dy*dy
	if far > 1 {
		return rgb{}, false
	}
	c := rgb{226, 150, 40}.mix(rgb{255, 214, 110}, max(0.8-far, 0))
	switch {
	case far > 0.72:
		c = rgb{150, 84, 26} // its edge
	case x == 8 && y == 9 || x == 9 && y == 10:
		c = rgb{52, 36, 24} // what it caught
	case dx < -0.3 && dy < -0.3:
		c = rgb{255, 240, 190} // a glint
	}
	return grain(c, 0.03, x, y, 966), true
}

// The beasts of the swamp, each a picture of eight tiles laid out as the others are:
// back, side, belly, the side of the head, the face, a leg, and two more of its own. A
// rattlewing's are a wing and the end of its tail, and a logcrawler's a feeler twice.
var (
	rattlewingTiles = []painter{wingBack, wingSide, wingBelly, wingHead, wingFace, wingLeg, wingWing, wingTail}
	logcrawlerTiles = []painter{crawlBack, crawlSide, crawlBelly, crawlHead, crawlFace, crawlLeg, crawlLeg, crawlLeg}
)

func init() {
	giantTiles["rattlewing"], giantTiles["logcrawler"] = rattlewingTiles, logcrawlerTiles
}

var (
	wingGreen = rgb{48, 150, 120} // a rattlewing's body: green, with the shine of metal
	wingBlue  = rgb{40, 96, 170}
	wingEye   = rgb{196, 60, 44} // its great eyes
	wingClear = rgb{206, 226, 232}
	crawlRed  = rgb{124, 50, 36} // a logcrawler's plates
	crawlDark = rgb{58, 26, 22}
	crawlPale = rgb{208, 150, 80} // its legs, and its underside
)

// wingShell is a rattlewing's body at a pixel: rings of green and blue that shine.
func wingShell(x, y, seed int) rgb {
	c := wingGreen.mix(wingBlue, 0.5+0.5*math.Sin(float64(x)*1.3))
	if x%4 == 3 {
		c = c.scale(0.6) // the join of two rings
	}
	if y < 4 {
		c = c.scale(1.35) // where the light is on it
	}
	return grain(c, 0.04, x, y, seed)
}

func wingBack(x, y int) (rgb, bool)  { return wingShell(y, x, 970), true }
func wingSide(x, y int) (rgb, bool)  { return wingShell(x, y, 971), true }
func wingBelly(x, y int) (rgb, bool) { return grain(wingGreen.scale(0.6), 0.05, x, y, 972), true }

// wingHead is the side of its head: nearly all eye.
func wingHead(x, y int) (rgb, bool) {
	if dx, dy := float64(x)-8, float64(y)-7; dx*dx+dy*dy < 36 {
		c := wingEye.scale(0.8 + 0.4*hash(x/2, y/2, 973))
		if dx < -2 && dy < -2 {
			c = rgb{255, 190, 170} // a glint
		}
		return c, true
	}
	return wingShell(x, y, 974), true
}

// wingFace is its face from in front: two great eyes, and its jaws between and under them.
func wingFace(x, y int) (rgb, bool) {
	for _, ex := range []float64{3.5, 11.5} {
		if dx, dy := float64(x)-ex, float64(y)-6; dx*dx+dy*dy < 14 {
			return wingEye.scale(0.8 + 0.4*hash(x/2, y/2, 973)), true
		}
	}
	if y >= 10 && x >= 5 && x <= 10 {
		return grain(rgb{36, 40, 34}, 0.05, x, y, 975), true
	}
	return wingShell(x, y, 976), true
}

func wingLeg(x, y int) (rgb, bool) { return grain(rgb{30, 34, 30}, 0.06, x, y, 977), true }

// wingWing is a wing: clear, with a net of dark veins and a dark mark near its tip.
func wingWing(x, y int) (rgb, bool) {
	switch {
	case y == 2 || x%5 == 0 || (x+y)%7 == 0:
		return rgb{70, 84, 86}, true // a vein
	case x >= 12 && y <= 5:
		return rgb{44, 40, 46}, true // the mark
	}
	return grain(wingClear, 0.03, x, y, 978), true
}

// wingTail is the end of its long tail: thin rings, blue.
func wingTail(x, y int) (rgb, bool) {
	c := wingBlue.scale(0.9 + 0.3*hash(x/3, 0, 979))
	if x%3 == 2 {
		c = c.scale(0.55)
	}
	return grain(c, 0.04, x, y, 980), true
}

// crawlPlates is a logcrawler's back at a pixel: plates one behind another, each dark at
// its hind edge and with a line of light along it.
func crawlPlates(x, y, seed int) rgb {
	c := crawlRed.scale(0.86 + 0.28*hash(x/4, 0, seed))
	switch x % 4 {
	case 3:
		c = crawlDark // the join of two plates
	case 0:
		c = c.scale(1.3)
	}
	return grain(c, 0.05, x, y, seed+1)
}

func crawlBack(x, y int) (rgb, bool) { return crawlPlates(y, x, 985), true }

// crawlSide is its side: the edge of its plates over the pale roots of its legs.
func crawlSide(x, y int) (rgb, bool) {
	if y >= 11 {
		return grain(crawlPale.scale(0.8), 0.05, x, y, 988), true
	}
	return crawlPlates(x, y, 985), true
}

func crawlBelly(x, y int) (rgb, bool) { return grain(crawlPale, 0.05, x, y, 989), true }

// crawlHead is the side of its head: a round plate, with a small dark eye.
func crawlHead(x, y int) (rgb, bool) {
	if c, ok := eyeOn(x, y, 11, 6, crawlPale); ok {
		return c, true
	}
	return grain(crawlRed.scale(0.9), 0.05, x, y, 990), true
}

// crawlFace is its face from in front: two eyes, and jaws under them.
func crawlFace(x, y int) (rgb, bool) {
	switch {
	case y >= 5 && y <= 6 && (x == 4 || x == 11):
		return rgb{22, 20, 18}, true
	case y >= 10 && x >= 5 && x <= 10 && x != 7 && x != 8:
		return crawlDark, true
	}
	return grain(crawlRed.scale(0.9), 0.05, x, y, 990), true
}

// crawlLeg is a leg, or a feeler: thin, pale, in joints.
func crawlLeg(x, y int) (rgb, bool) {
	c := crawlPale
	if y%5 == 4 {
		c = c.scale(0.6)
	}
	return grain(c, 0.05, x, y, 991), true
}

// tar is black, thick and wet: it shines where it has lately moved, and a bubble stands
// on it here and there.
func tar(x, y int) (rgb, bool) {
	c := rgb{20, 18, 20}.scale(0.8 + 0.5*blotch(x, y, 8, 995))
	switch h := hash(x, y, 996); {
	case blotch(x, y, 4, 997) > 0.8:
		c = rgb{58, 54, 62} // the shine of it
	case h < 0.02:
		c = rgb{92, 84, 78} // a bubble
	}
	return grain(c, 0.04, x, y, 998), true
}

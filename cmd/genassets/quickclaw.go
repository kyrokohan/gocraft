package main

// The quickclaw's picture is five tiles side by side, which the renderer lays on the
// boxes it is built from: its back, its side, the side of its head, a leg and an arm.
var quickclawTiles = []painter{quickclawBack, quickclawSide, quickclawHead, quickclawLeg, quickclawArm}

var (
	clawSlate = rgb{58, 62, 74}
	clawRust  = rgb{168, 84, 44}
	clawPale  = rgb{198, 190, 168}
)

// clawFeathers is its coat at a pixel: dark feathers that lie in rows, barred with rust.
func clawFeathers(x, y, seed int) rgb {
	c := clawSlate.scale(0.85 + 0.3*hash(x/2, y, seed))
	if (y+x/4)%5 == 0 {
		c = clawRust.scale(0.8 + 0.3*hash(x, y, seed+1)) // a bar across the feathers
	}
	return grain(c, 0.06, x, y, seed+2)
}

func quickclawBack(x, y int) (rgb, bool) {
	return clawFeathers(x, y, 530), true
}

// quickclawSide is its side: the feathers of the back, and a pale belly under them.
func quickclawSide(x, y int) (rgb, bool) {
	if edge := 11 + int(hash(x, 0, 533)*2); y >= edge {
		return grain(clawPale, 0.05, x, y, 534), true
	}
	return clawFeathers(x, y, 530), true
}

// quickclawHead is the side of its head, with its front at the right of the tile: a
// narrow snout with teeth along it. Its eye is not in the picture: it is a light of
// its own.
func quickclawHead(x, y int) (rgb, bool) {
	switch {
	case x >= 7 && y == 10:
		return rgb{28, 24, 26}, true // the line of the mouth
	case x >= 7 && y == 11 && x%2 == 1:
		return rgb{236, 232, 218}, true // a tooth
	case y >= 12:
		return grain(clawPale, 0.05, x, y, 534), true
	case y < 3 && x < 8:
		return grain(clawRust, 0.08, x, y, 535), true // a crest of feathers
	}
	return clawFeathers(x, y, 530), true
}

// quickclawLeg is a leg from the hip: feathers to the knee, then scales, and claws.
func quickclawLeg(x, y int) (rgb, bool) {
	switch {
	case y < 6:
		return clawFeathers(x, y, 536), true
	case y >= 14:
		if x%4 == 3 {
			return rgb{22, 20, 22}, true // a claw
		}
		return grain(rgb{150, 132, 84}, 0.06, x, y, 537), true
	}
	return grain(rgb{176, 156, 98}.scale(0.9+0.2*hash(0, y/2, 538)), 0.05, x, y, 537), true
}

// quickclawArm is an arm: a fan of long rust feathers.
func quickclawArm(x, y int) (rgb, bool) {
	c := clawRust.scale(0.78 + 0.4*hash(x/2, 0, 539))
	if y > 12 {
		c = clawSlate
	}
	return grain(c, 0.06, x, y, 540), true
}

package main

import "math"

// The armed plant-eaters. The picture of each is eight tiles side by side, which the
// renderer lays on the boxes the beast is built from: its back, its side, its belly, the
// side of its head with its front at the right of the tile, its face, a leg, and two of
// what it is armed with.
var (
	shieldbrowTiles  = []painter{browBack, browSide, browBelly, browHead, browFace, browLeg, browFrill, horn}
	thornbackTiles   = []painter{thornBack, thornSide, thornBelly, thornHead, thornFace, thornLeg, thornPlate, horn}
	boulderbackTiles = []painter{boulderBack, boulderSide, boulderBelly, boulderHead, boulderFace, boulderLeg, boulderClub, boulderTail}

	// giantTiles is the picture of each, by the name of its file.
	giantTiles = map[string][]painter{
		"shieldbrow": shieldbrowTiles, "thornback": thornbackTiles, "boulderback": boulderbackTiles,
		"cloudgrazer": cloudgrazerTiles, "dread": dreadTiles,
	}
)

var (
	ivory = rgb{228, 216, 188}

	browHide  = rgb{134, 114, 90}
	browDark  = rgb{92, 76, 62}
	browCream = rgb{206, 190, 154}
	browFlame = rgb{204, 98, 50} // the eye spots of its frill

	thornHide  = rgb{98, 118, 72}
	thornDark  = rgb{60, 78, 52}
	thornCream = rgb{196, 190, 140}
	thornRust  = rgb{202, 116, 56} // its plates

	boulderHide  = rgb{152, 114, 82}
	boulderDark  = rgb{98, 72, 56}
	boulderCream = rgb{214, 192, 150}
	boulderBone  = rgb{196, 180, 150} // the knobs of its armor
)

// eyeOn draws an eye with its middle at ex, ey on a tile: dark, in a pale ring.
func eyeOn(x, y, ex, ey int, ring rgb) (rgb, bool) {
	switch dx, dy := x-ex, y-ey; {
	case (dx == 0 || dx == 1) && (dy == 0 || dy == 1):
		return rgb{22, 20, 18}, true
	case dx >= -1 && dx <= 2 && dy >= -1 && dy <= 2:
		return ring, true
	}
	return rgb{}, false
}

// horn is a horn, a spike or a tusk: ivory, ringed, and dark at the tip, which is at
// the top of the tile.
func horn(x, y int) (rgb, bool) {
	c := ivory.scale(0.82 + 0.2*float64(y)/float64(size-1))
	switch {
	case y < 3:
		c = rgb{84, 70, 58}
	case y%4 == 3:
		c = c.scale(0.86)
	}
	return grain(c, 0.04, x, y, 640), true
}

// browSkin is a shieldbrow's hide at a pixel: thick, in big pebbles, darker along the
// back.
func browSkin(x, y int, down float64, seed int) rgb {
	c := browHide.mix(browDark, 0.5*(1-down)).scale(0.86 + 0.28*blotch(x, y, 4, seed))
	if hash(x/2, y/2, seed+1) < 0.16 {
		c = c.scale(0.8) // the gap between two pebbles
	}
	return grain(c, 0.05, x, y, seed+2)
}

func browBack(x, y int) (rgb, bool) { return browSkin(x, y, 0, 641), true }

func browSide(x, y int) (rgb, bool) {
	down := float64(y) / float64(size-1)
	c := browSkin(x, y, down, 644)
	if edge := 11 + int(hash(x, 0, 647)*3); y >= edge {
		c = c.mix(browCream, min(float64(y-edge+1)/3, 1))
	}
	return c, true
}

func browBelly(x, y int) (rgb, bool) { return grain(browCream, 0.05, x, y, 648), true }

// browHead is the side of its head: an eye, and a beak like a parrot's.
func browHead(x, y int) (rgb, bool) {
	if c, ok := eyeOn(x, y, 6, 5, rgb{226, 196, 110}); ok {
		return c, true
	}
	switch {
	case x >= 11 && y >= 7 && x-11 >= (y-7)/3:
		if y == 11 && x < 15 {
			return rgb{52, 44, 38}, true // the line of the mouth
		}
		return grain(rgb{196, 168, 96}, 0.05, x, y, 649), true // the beak
	case y >= 13:
		return grain(browCream, 0.05, x, y, 648), true
	}
	return browSkin(x, y, float64(y)/float64(size-1), 644), true
}

// browFace is its face from in front: the beak, two nostrils, and the hide between the
// eyes.
func browFace(x, y int) (rgb, bool) {
	switch {
	case y >= 9 && x >= 4 && x <= 11:
		if y == 10 && (x == 6 || x == 9) {
			return rgb{40, 34, 30}, true
		}
		return grain(rgb{196, 168, 96}, 0.05, x, y, 649), true
	}
	return browSkin(x, y, 0.4, 650), true
}

// browLeg is a leg as thick as a tree: hide in rings, and three blunt nails at the foot.
func browLeg(x, y int) (rgb, bool) {
	switch {
	case y >= 13 && x%5 >= 1 && x%5 <= 3:
		return grain(rgb{62, 54, 48}, 0.06, x, y, 651), true
	case y%5 == 4:
		return grain(browDark, 0.05, x, y, 652), true
	}
	return browSkin(x, y, 0.7, 653), true
}

// browFrill is the shield of bone behind its head, seen flat: hide, edged with knobs,
// with two great spots like eyes of flame.
func browFrill(x, y int) (rgb, bool) {
	if y == 0 && x%3 != 2 || y == 1 && x%3 == 0 {
		return grain(ivory, 0.05, x, y, 654), true // the knobs along its edge
	}
	for _, at := range [2]float64{4, 11} {
		dx, dy := float64(x)+0.5-at, (float64(y)-7)/1.4
		switch d := math.Sqrt(dx*dx + dy*dy); {
		case d < 1.4:
			return grain(rgb{40, 30, 28}, 0.08, x, y, 655), true
		case d < 2.9:
			return grain(browFlame, 0.06, x, y, 656), true
		case d < 3.7:
			return grain(browCream, 0.05, x, y, 657), true
		}
	}
	return browSkin(x, y, 0.3, 658), true
}

// thornSkin is a thornback's hide: olive, banded dark down from the back.
func thornSkin(x, y int, down float64, seed int) rgb {
	c := thornHide.scale(0.88 + 0.24*blotch(x, y, 4, seed))
	if math.Sin(float64(x)*math.Pi/3) > 0.3+0.9*down {
		c = thornDark.mix(c, 0.2)
	}
	return grain(c, 0.06, x, y, seed+1)
}

func thornBack(x, y int) (rgb, bool) { return thornSkin(x, y, 0, 660), true }

func thornSide(x, y int) (rgb, bool) {
	down := float64(y) / float64(size-1)
	c := thornSkin(x, y, down, 662)
	if edge := 11 + int(hash(x, 0, 664)*3); y >= edge {
		c = c.mix(thornCream, min(float64(y-edge+1)/3, 1))
	}
	return c, true
}

func thornBelly(x, y int) (rgb, bool) { return grain(thornCream, 0.05, x, y, 665), true }

// thornHead is the side of its small low head: an eye and a narrow beak.
func thornHead(x, y int) (rgb, bool) {
	if c, ok := eyeOn(x, y, 7, 5, rgb{222, 200, 110}); ok {
		return c, true
	}
	switch {
	case x >= 12 && y >= 8:
		return grain(rgb{170, 160, 104}, 0.05, x, y, 666), true
	case y == 11 && x >= 8:
		return rgb{44, 50, 36}, true
	case y >= 13:
		return grain(thornCream, 0.05, x, y, 665), true
	}
	return thornSkin(x, y, float64(y)/float64(size-1), 662), true
}

func thornFace(x, y int) (rgb, bool) {
	if y >= 10 && x >= 5 && x <= 10 {
		return grain(rgb{170, 160, 104}, 0.05, x, y, 666), true
	}
	return thornSkin(x, y, 0.5, 667), true
}

func thornLeg(x, y int) (rgb, bool) {
	switch {
	case y >= 13 && x%5 >= 1 && x%5 <= 3:
		return grain(rgb{50, 54, 42}, 0.06, x, y, 668), true
	case y%5 == 4:
		return grain(thornDark, 0.05, x, y, 669), true
	}
	return thornSkin(x, y, 0.8, 670), true
}

// thornPlate is one of the plates that stand along its back, seen flat: a leaf of bone,
// rust red, darker at its foot and its rim. The corners of the tile are holes.
func thornPlate(x, y int) (rgb, bool) {
	half := 1.5 + 6.5*math.Sin(math.Pi*min(float64(y)+3, 17)/20)
	dx := math.Abs(float64(x) - 7.5)
	switch {
	case dx > half:
		return rgb{}, false
	case dx > half-1.3 || y > 13:
		return grain(thornRust.scale(0.62), 0.06, x, y, 671), true
	case (x+y)%5 == 0:
		return grain(thornRust.scale(1.2), 0.05, x, y, 672), true // the blood in it shows through
	}
	return grain(thornRust, 0.07, x, y, 673), true
}

// boulderSkin is a boulderback's hide where it has no armor.
func boulderSkin(x, y, seed int) rgb {
	return grain(boulderHide.scale(0.88+0.24*blotch(x, y, 4, seed)), 0.06, x, y, seed+1)
}

// boulderScutes is its armor: rows of bone knobs set in dark hide, every other row
// shifted. knob is how far across a knob is.
func boulderScutes(x, y, knob, seed int) rgb {
	row := y / knob
	at := x + row%2*knob/2
	dx, dy := float64(at%knob)-float64(knob-1)/2, float64(y%knob)-float64(knob-1)/2
	if d := math.Sqrt(dx*dx + dy*dy); d < float64(knob)/2-0.6 {
		c := boulderBone.scale(0.84 + 0.3*hash(at/knob, row, seed))
		if dx+dy < -0.5 {
			c = c.scale(1.14) // the side of the knob that catches the light
		}
		return grain(c, 0.05, x, y, seed+1)
	}
	return grain(boulderDark, 0.08, x, y, seed+2)
}

func boulderBack(x, y int) (rgb, bool) { return boulderScutes(x, y, 4, 680), true }

// boulderSide is its side: the armor comes half way down, with a row of spikes along
// its edge, and hide under it.
func boulderSide(x, y int) (rgb, bool) {
	switch {
	case y < 7:
		return boulderScutes(x, y, 4, 683), true
	case y < 10 && x%4 != 3 && y-7 <= 2-abs(x%4-1):
		return grain(ivory, 0.05, x, y, 686), true // a spike
	case y >= 13:
		return grain(boulderCream, 0.05, x, y, 687), true
	}
	return boulderSkin(x, y, 688), true
}

func boulderBelly(x, y int) (rgb, bool) { return grain(boulderCream, 0.05, x, y, 687), true }

// boulderHead is the side of its flat head: armor over the brow, a small eye under it,
// and a horn at the back corner.
func boulderHead(x, y int) (rgb, bool) {
	if c, ok := eyeOn(x, y, 8, 7, rgb{210, 180, 110}); ok {
		return c, true
	}
	switch {
	case y < 5:
		return boulderScutes(x, y, 4, 690), true
	case x < 3 && y < 9:
		return grain(ivory, 0.05, x, y, 686), true
	case y == 11 && x >= 9:
		return rgb{54, 40, 34}, true
	case y >= 13:
		return grain(boulderCream, 0.05, x, y, 687), true
	}
	return boulderSkin(x, y, 691), true
}

func boulderFace(x, y int) (rgb, bool) {
	switch {
	case y < 5:
		return boulderScutes(x, y, 4, 690), true
	case y == 9 && (x == 6 || x == 9):
		return rgb{44, 34, 30}, true
	case y >= 11 && x >= 4 && x <= 11:
		return grain(rgb{176, 150, 110}, 0.05, x, y, 692), true
	}
	return boulderSkin(x, y, 693), true
}

func boulderLeg(x, y int) (rgb, bool) {
	switch {
	case y >= 13 && x%5 >= 1 && x%5 <= 3:
		return grain(rgb{60, 46, 40}, 0.06, x, y, 694), true
	case y%5 == 4:
		return grain(boulderDark, 0.05, x, y, 695), true
	}
	return boulderSkin(x, y, 696), true
}

// boulderClub is the club at the end of its tail: two great knobs of bone grown
// together, cracked and worn.
func boulderClub(x, y int) (rgb, bool) {
	c := boulderBone.scale(0.74 + 0.36*blotch(x, y, 8, 697))
	if x == 7 || x == 8 || hash(x/3, y/3, 698) < 0.1 {
		c = c.scale(0.66) // the seam between the knobs, and a crack
	}
	return grain(c, 0.06, x, y, 699), true
}

// boulderTail is its tail: stiff, ringed with bone.
func boulderTail(x, y int) (rgb, bool) {
	if x%4 == 0 {
		return grain(boulderBone.scale(0.9), 0.05, x, y, 700), true
	}
	return boulderSkin(x, y, 701), true
}

// The cloudgrazer's picture is laid out as the others are. What it is armed with is
// nothing: the last two tiles are the bands of its neck and the end of its tail.
var cloudgrazerTiles = []painter{cloudBack, cloudSide, cloudBelly, cloudHead, cloudFace, cloudLeg, cloudNeck, cloudTail}

var (
	cloudHide  = rgb{106, 124, 142} // the blue grey of far hills
	cloudDark  = rgb{66, 80, 98}
	cloudCream = rgb{212, 208, 184}
)

// cloudSkin is a cloudgrazer's hide at a pixel: smooth, and dappled pale, darker along
// the back.
func cloudSkin(x, y int, down float64, seed int) rgb {
	c := cloudHide.mix(cloudDark, 0.6*(1-down)).scale(0.9 + 0.2*blotch(x, y, 8, seed))
	if blotch(x, y, 2, seed+1) > 0.74 {
		c = c.mix(cloudCream, 0.35) // a dapple
	}
	return grain(c, 0.04, x, y, seed+2)
}

func cloudBack(x, y int) (rgb, bool) { return cloudSkin(x, y, 0, 710), true }

func cloudSide(x, y int) (rgb, bool) {
	down := float64(y) / float64(size-1)
	c := cloudSkin(x, y, down, 713)
	if edge := 10 + int(hash(x, 0, 716)*3); y >= edge {
		c = c.mix(cloudCream, min(float64(y-edge+1)/4, 1))
	}
	return c, true
}

func cloudBelly(x, y int) (rgb, bool) { return grain(cloudCream, 0.04, x, y, 717), true }

// cloudHead is the side of its small head: an eye, a nostril high on the brow, and a
// mouth that seems to smile.
func cloudHead(x, y int) (rgb, bool) {
	if c, ok := eyeOn(x, y, 6, 5, rgb{232, 214, 150}); ok {
		return c, true
	}
	switch {
	case x >= 11 && x <= 12 && y == 3:
		return rgb{40, 48, 60}, true
	case y == 11 && x >= 7 || y == 10 && x == 6:
		return rgb{46, 54, 66}, true
	case y >= 12:
		return grain(cloudCream, 0.04, x, y, 717), true
	}
	return cloudSkin(x, y, float64(y)/float64(size-1), 713), true
}

func cloudFace(x, y int) (rgb, bool) {
	switch {
	case y == 4 && (x == 5 || x == 10):
		return rgb{40, 48, 60}, true
	case y == 11 && x >= 3 && x <= 12:
		return rgb{46, 54, 66}, true
	case y >= 12:
		return grain(cloudCream, 0.04, x, y, 717), true
	}
	return cloudSkin(x, y, 0.5, 718), true
}

// cloudLeg is a leg like the trunk of a tree: hide in deep folds at the knee, and a
// row of nails round the foot.
func cloudLeg(x, y int) (rgb, bool) {
	switch {
	case y >= 14 && x%4 != 3:
		return grain(rgb{196, 190, 170}, 0.05, x, y, 719), true
	case y == 7 || y == 8 || y == 13:
		return grain(cloudDark, 0.05, x, y, 720), true
	}
	return cloudSkin(x, y, 0.8, 721), true
}

// cloudNeck is its neck: the hide of its side, ringed with pale bands.
func cloudNeck(x, y int) (rgb, bool) {
	c := cloudSkin(x, y, 0.5, 722)
	if x%5 == 0 {
		c = c.mix(cloudCream, 0.5)
	}
	return c, true
}

// cloudTail is the end of its tail: hide that pales to the tip.
func cloudTail(x, y int) (rgb, bool) {
	return cloudSkin(x, y, 0.6, 723).mix(cloudCream, 0.25), true
}

// The dread's picture is laid out as the others are. The last two tiles are its small
// arm and the inside of its mouth.
var dreadTiles = []painter{dreadBack, dreadSide, dreadBelly, dreadHead, dreadFace, dreadLeg, dreadArm, dreadMouth}

var (
	dreadHide  = rgb{66, 58, 56} // the color of a burnt tree
	dreadBand  = rgb{150, 62, 40}
	dreadCream = rgb{196, 176, 142}
	dreadTooth = rgb{238, 232, 212}
)

// dreadSkin is the dread's hide at a pixel: dark and rough, with ragged bands the color
// of dried blood down from its back.
func dreadSkin(x, y int, down float64, seed int) rgb {
	c := dreadHide.scale(0.82 + 0.36*blotch(x, y, 4, seed))
	if math.Sin(float64(x)*math.Pi/4+2*blotch(x, y, 8, seed+1)) > 0.5+0.9*down {
		c = dreadBand.scale(0.8 + 0.3*hash(x, y, seed+2))
	}
	return grain(c, 0.07, x, y, seed+3)
}

func dreadBack(x, y int) (rgb, bool) { return dreadSkin(x, y, 0, 730), true }

func dreadSide(x, y int) (rgb, bool) {
	down := float64(y) / float64(size-1)
	c := dreadSkin(x, y, down, 734)
	if edge := 11 + int(hash(x, 0, 738)*3); y >= edge {
		c = c.mix(dreadCream, min(float64(y-edge+1)/3, 1))
	}
	return c, true
}

func dreadBelly(x, y int) (rgb, bool) { return grain(dreadCream, 0.05, x, y, 739), true }

// dreadHead is the side of its head, with its front at the right of the tile: a ridge
// of bone over where its eye is, which is a light of its own, and a mouth the length of
// the head, full of teeth.
func dreadHead(x, y int) (rgb, bool) {
	switch {
	case y >= 3 && y <= 4 && x >= 2 && x <= 7:
		return grain(dreadBand.scale(0.7), 0.06, x, y, 740), true // the brow
	case y == 9 && x >= 3:
		return rgb{20, 14, 14}, true // the line of the mouth
	case (y == 10 && x >= 4 && x%2 == 0) || (y == 8 && x >= 5 && x%2 == 1):
		return dreadTooth, true
	case y >= 13:
		return grain(dreadCream, 0.05, x, y, 739), true
	}
	return dreadSkin(x, y, float64(y)/float64(size-1), 734), true
}

// dreadFace is its face from in front: two nostrils, and the teeth of both jaws.
func dreadFace(x, y int) (rgb, bool) {
	switch {
	case y == 4 && (x == 5 || x == 10):
		return rgb{20, 14, 14}, true
	case y == 9:
		return rgb{20, 14, 14}, true
	case (y == 8 || y == 10) && x >= 2 && x <= 13 && x%2 == 0:
		return dreadTooth, true
	case y >= 13:
		return grain(dreadCream, 0.05, x, y, 739), true
	}
	return dreadSkin(x, y, 0.5, 741), true
}

// dreadLeg is a leg: hide over great muscle to the knee, then scales, and three claws.
func dreadLeg(x, y int) (rgb, bool) {
	switch {
	case y >= 14:
		if x%5 == 2 {
			return rgb{226, 216, 190}, true // a claw
		}
		return grain(rgb{92, 78, 62}, 0.06, x, y, 742), true
	case y >= 9:
		return grain(rgb{108, 92, 70}.scale(0.88+0.24*hash(0, y, 743)), 0.05, x, y, 744), true
	}
	return dreadSkin(x, y, 0.6, 745), true
}

// dreadArm is one of its small arms: hide, and two claws.
func dreadArm(x, y int) (rgb, bool) {
	if y >= 13 && (x < 6 || x > 9) {
		return rgb{226, 216, 190}, true
	}
	return dreadSkin(x, y, 0.7, 746), true
}

// dreadMouth is the inside of its mouth: red, and dark at the throat.
func dreadMouth(x, y int) (rgb, bool) {
	dx, dy := float64(x)-7.5, float64(y)-7.5
	c := rgb{168, 44, 40}.scale(0.5 + 0.06*math.Sqrt(dx*dx+dy*dy))
	if y == 0 || y == size-1 {
		if x%2 == 0 {
			return dreadTooth, true
		}
	}
	return grain(c, 0.08, x, y, 747), true
}

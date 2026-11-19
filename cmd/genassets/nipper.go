package main

// The nipper's picture is four tiles side by side, which the renderer lays on the boxes
// it is built from: its back, its side, the side of its head and a leg.
var nipperTiles = []painter{nipperBack, nipperSide, nipperHead, nipperLeg}

var (
	nipperGreen  = rgb{64, 132, 84}
	nipperYellow = rgb{222, 198, 84}
	nipperPale   = rgb{206, 214, 168}
)

// nipperBack is its back: green, with a yellow stripe down the middle of it.
func nipperBack(x, y int) (rgb, bool) {
	if x >= 6 && x <= 9 {
		return grain(nipperYellow, 0.06, x, y, 520), true
	}
	return grain(nipperGreen.scale(0.9+0.2*blotch(x, y, 4, 521)), 0.06, x, y, 522), true
}

// nipperSide is its side: green above, pale under, with a row of yellow spots between.
func nipperSide(x, y int) (rgb, bool) {
	switch {
	case y >= 11:
		return grain(nipperPale, 0.05, x, y, 523), true
	case y >= 8 && y <= 9 && x%4 < 2:
		return grain(nipperYellow, 0.06, x, y, 520), true
	}
	return grain(nipperGreen.scale(0.9+0.2*blotch(x, y, 4, 521)), 0.06, x, y, 522), true
}

// nipperHead is the side of its head, with its front at the right of the tile: a big
// dark eye, and a mouth.
func nipperHead(x, y int) (rgb, bool) {
	switch {
	case x >= 7 && x <= 10 && y >= 4 && y <= 7:
		if x >= 8 && x <= 9 && y >= 5 && y <= 6 {
			return rgb{20, 18, 18}, true
		}
		return nipperYellow, true // the ring of its eye
	case y == 11 && x >= 8:
		return rgb{40, 62, 44}, true // the mouth
	case y >= 12:
		return grain(nipperPale, 0.05, x, y, 523), true
	}
	return grain(nipperGreen, 0.06, x, y, 522), true
}

// nipperLeg is a thin leg: green at the thigh, and yellow down to the foot.
func nipperLeg(x, y int) (rgb, bool) {
	if y < 6 {
		return grain(nipperGreen, 0.06, x, y, 522), true
	}
	return grain(nipperYellow.scale(0.85), 0.06, x, y, 524), true
}

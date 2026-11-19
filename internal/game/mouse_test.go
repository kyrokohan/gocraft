package game

import (
	"math"
	"testing"

	"glonk.dev/engine"

	"github.com/kyrokohan/skyspear/internal/world"
)

// The size of the frame in these tests, in cells.
const cols, rows = 100, 30

// withMouse is a game on the test floor that knows the size of its frame, its player
// looking at the floor a little ahead, and a keyboard to move the pointer with.
func withMouse(t *testing.T) (*Game, *engine.Keyboard) {
	t.Helper()
	g := lookingDown(t)
	g.Resize(cols, rows)
	return g, engine.NewKeyboard()
}

// step runs frames of the game with the keyboard as it stands.
func step(t *testing.T, g *Game, kb *engine.Keyboard, frames int) {
	t.Helper()
	for range frames {
		if err := g.Update(frame, kb.Input()); err != nil {
			t.Fatalf("Update: %v", err)
		}
		kb.EndFrame()
	}
}

// dug counts the blocks missing from the test floor.
func dug(g *Game) int {
	n := 0
	for z := range 40 {
		for x := range 40 {
			for y := range 6 {
				if g.world.At(x, y, z) == world.Air {
					n++
				}
			}
		}
	}
	return n
}

func TestThePointerAimsOnceTheMouseMoves(t *testing.T) {
	g, kb := withMouse(t)
	middle := g.target

	// Until the mouse says where it is, the middle of the view aims.
	step(t, g, kb, 1)
	if g.pointing || g.target != middle {
		t.Fatalf("pointing %v at %+v before the mouse moved, wanted the middle", g.pointing, g.target)
	}

	// The pointer in the middle of the frame aims where the middle does.
	kb.MouseMove(cols/2, rows/2)
	step(t, g, kb, 1)
	if !g.pointing || !g.aimed || g.target.X != middle.X || g.target.Y != middle.Y || g.target.Z != middle.Z {
		t.Errorf("pointing %v at %+v with the pointer in the middle, wanted %+v", g.pointing, g.target, middle)
	}

	// Down and to the left of it is a block nearer the player and to its left.
	kb.MouseMove(cols/2-20, rows-8)
	step(t, g, kb, 1)
	if !g.aimed || g.target.X >= middle.X || g.target.Z > middle.Z {
		t.Errorf("aimed %v at %+v with the pointer low and to the left of %+v", g.aimed, g.target, middle)
	}
}

func TestAnArrowKeyPutsTheAimBackInTheMiddle(t *testing.T) {
	g, kb := withMouse(t)
	kb.MouseMove(cols/2-20, rows-8)
	step(t, g, kb, 1)
	if _, fb := drawn(t, g); fb.At(cols/2, rows/2).Character == '+' {
		t.Error("the crosshair shows while the pointer aims")
	}

	kb.Press(engine.KeyLeft)
	step(t, g, kb, 1)
	kb.Release(engine.KeyLeft)

	if g.pointing {
		t.Error("the pointer still aims after an arrow key")
	}
	if _, fb := drawn(t, g); fb.At(cols/2, rows/2).Character != '+' {
		t.Error("the crosshair is not back")
	}
}

func TestLeftClickBreaksAndRightClickPlaces(t *testing.T) {
	g, kb := withMouse(t)
	kb.MouseMove(cols/2, rows/2)
	step(t, g, kb, 1)
	at := g.target

	kb.MouseDown(engine.MouseRight)
	step(t, g, kb, 1)
	kb.MouseUp(engine.MouseRight)
	step(t, g, kb, 1)
	if got := g.world.At(at.X, at.Y+1, at.Z); got != hotbar[0] {
		t.Fatalf("a right click put %v on the block, wanted %v", got.Info().Name, hotbar[0].Info().Name)
	}

	// The pointer is now on the block that was placed.
	kb.MouseDown(engine.MouseLeft)
	step(t, g, kb, 1)
	kb.MouseUp(engine.MouseLeft)
	step(t, g, kb, 1)
	if got := g.world.At(at.X, at.Y+1, at.Z); got != world.Air {
		t.Errorf("a left click left %v there, wanted it broken", got.Info().Name)
	}
	if dug(g) != 0 {
		t.Errorf("%d blocks of the floor are gone after one click on the placed block", dug(g))
	}
}

func TestAHeldButtonRepeats(t *testing.T) {
	g, kb := withMouse(t)
	kb.MouseMove(cols/2, rows/2)
	kb.MouseDown(engine.MouseLeft)

	// A button that is down may be the start of a drag, so nothing breaks at once.
	step(t, g, kb, 6) // a fifth of a second
	if n := dug(g); n != 0 {
		t.Fatalf("%d blocks broken in a fifth of a second of holding, wanted none yet", n)
	}

	step(t, g, kb, 30) // a second more: held still, it digs
	if n := dug(g); n < 3 || n > 6 {
		t.Errorf("%d blocks broken with the button held still for 1.2 seconds, wanted about four", n)
	}

	// Letting go of a button that has dug is not one more click.
	kb.MouseUp(engine.MouseLeft)
	before := dug(g)
	step(t, g, kb, 30)
	if dug(g) != before {
		t.Error("blocks went on breaking after the button was let go")
	}
}

func TestTheWheelPicksTheBlock(t *testing.T) {
	g, kb := withMouse(t)

	kb.Wheel(0, 1)
	step(t, g, kb, 1)
	if g.selected != 1 {
		t.Errorf("one step down picked slot %d, wanted 1", g.selected)
	}

	kb.Wheel(0, -2)
	step(t, g, kb, 1)
	if g.selected != len(hotbar)-1 {
		t.Errorf("two steps up from slot 1 picked %d, wanted it to go round to %d", g.selected, len(hotbar)-1)
	}

	step(t, g, kb, 5)
	if g.selected != len(hotbar)-1 {
		t.Errorf("the slot moved to %d with the wheel still", g.selected)
	}
}

func TestAClickOnTheHotbarPicksItsSlotAndBreaksNothing(t *testing.T) {
	g, kb := withMouse(t)
	x0, y0 := barOrigin(cols, rows)

	kb.MouseMove(x0+1+4*slotPitch+2, y0+2) // inside the fifth slot
	kb.MouseDown(engine.MouseLeft)
	step(t, g, kb, 1)
	kb.MouseUp(engine.MouseLeft)
	step(t, g, kb, 20)

	if g.selected != 4 {
		t.Errorf("slot %d is picked, wanted 4", g.selected)
	}
	if n := dug(g); n != 0 || g.aimed {
		t.Errorf("%d blocks broken and aimed %v with the pointer on the hotbar, wanted nothing", n, g.aimed)
	}
}

func TestTheMouseDoesNothingWhileTheGameIsPaused(t *testing.T) {
	g, kb := withMouse(t)
	kb.MouseMove(cols/2, rows/2)
	step(t, g, kb, 1)
	kb.Press(engine.KeyEscape)
	step(t, g, kb, 1)
	kb.Release(engine.KeyEscape)

	kb.MouseDown(engine.MouseLeft)
	kb.Wheel(0, 3)
	step(t, g, kb, 40)

	if n := dug(g); n != 0 || g.selected != 0 {
		t.Errorf("%d blocks broken and slot %d picked while paused, wanted nothing to change", n, g.selected)
	}
}

// A game that was never told its size cannot tell where the pointer aims, and falls
// back on the middle of the view.
func TestThePointerAimsFromTheMiddleUntilTheSizeIsKnown(t *testing.T) {
	g := lookingDown(t)
	middle := g.target
	kb := engine.NewKeyboard()

	kb.MouseMove(3, 3)
	step(t, g, kb, 1)

	if !g.aimed || g.target != middle {
		t.Errorf("aimed %v at %+v, wanted the middle: %+v", g.aimed, g.target, middle)
	}
}

// cellAngle is how far one cell of a drag turns the view up or down; across, a cell
// turns it half as far, since a cell is half as wide as it is tall.
func cellAngle() float64 {
	return defaultSettings.MouseSpeed * 2 * math.Tan(defaultSettings.fov()/2) / rows
}

func TestDraggingTurnsTheViewAndBreaksNothing(t *testing.T) {
	g, kb := withMouse(t)
	yaw, pitch := g.player.Yaw, g.player.Pitch
	kb.MouseMove(40, 15)
	kb.MouseDown(engine.MouseLeft)
	step(t, g, kb, 1)

	kb.MouseMove(52, 15) // twelve cells to the right
	step(t, g, kb, 1)
	if want := yaw + 12*cellAngle()/2; !near(g.player.Yaw, want) || g.player.Pitch != pitch {
		t.Fatalf("after a drag of 12 cells right: yaw %v pitch %v, wanted yaw %v and the pitch unchanged", g.player.Yaw, g.player.Pitch, want)
	}

	kb.MouseMove(52, 11) // and four up
	step(t, g, kb, 1)
	if want := pitch + 4*cellAngle(); !near(g.player.Pitch, want) {
		t.Errorf("after a drag of 4 cells up: pitch %v, wanted %v", g.player.Pitch, want)
	}

	// Held still at the end of a drag, the button does not start to dig.
	step(t, g, kb, 30)
	kb.MouseUp(engine.MouseLeft)
	step(t, g, kb, 2)
	if n := dug(g); n != 0 {
		t.Errorf("a drag broke %d blocks", n)
	}

	// With the button up the pointer aims again and does not turn the view.
	turned := g.player.Yaw
	kb.MouseMove(20, 15)
	step(t, g, kb, 1)
	if g.player.Yaw != turned {
		t.Error("the view turned with the pointer after the button was let go")
	}
}

func TestAClickThatShiftsByOneColumnIsStillAClick(t *testing.T) {
	g, kb := withMouse(t)
	yaw := g.player.Yaw
	kb.MouseMove(cols/2, rows/2)
	kb.MouseDown(engine.MouseLeft)
	step(t, g, kb, 1)

	kb.MouseMove(cols/2+1, rows/2) // a hand is not perfectly still
	step(t, g, kb, 1)
	kb.MouseUp(engine.MouseLeft)
	step(t, g, kb, 1)

	if g.player.Yaw != yaw {
		t.Error("a click that shifted by one column turned the view")
	}
	if n := dug(g); n != 1 {
		t.Errorf("the click broke %d blocks, wanted 1", n)
	}
}

func TestAClickShorterThanAFrameBreaksOneBlock(t *testing.T) {
	g, kb := withMouse(t)
	kb.MouseMove(cols/2, rows/2)
	step(t, g, kb, 1)

	kb.MouseDown(engine.MouseLeft)
	kb.MouseUp(engine.MouseLeft)
	step(t, g, kb, 3)

	if n := dug(g); n != 1 {
		t.Errorf("a quick click broke %d blocks, wanted 1", n)
	}
}

func TestADragCannotTipTheViewOver(t *testing.T) {
	g, kb := withMouse(t)
	kb.MouseMove(50, 2)
	kb.MouseDown(engine.MouseLeft)
	step(t, g, kb, 1)

	for range 20 { // down the whole frame, again and again
		kb.MouseMove(50, rows-1)
		step(t, g, kb, 1)
		kb.MouseUp(engine.MouseLeft)
		kb.MouseMove(50, 2)
		step(t, g, kb, 1)
		kb.MouseDown(engine.MouseLeft)
		step(t, g, kb, 1)
	}

	if g.player.Pitch != -maxPitch {
		t.Errorf("pitch %v after dragging down many times, wanted it held at %v", g.player.Pitch, -maxPitch)
	}
}

// A game that was never told the size of its frame cannot tell how far a cell turns
// the view, so a drag does nothing, and does not break a block either.
func TestADragDoesNothingUntilTheSizeIsKnown(t *testing.T) {
	g := lookingDown(t)
	kb := engine.NewKeyboard()
	yaw := g.player.Yaw
	kb.MouseMove(10, 10)
	kb.MouseDown(engine.MouseLeft)
	step(t, g, kb, 1)

	kb.MouseMove(30, 10)
	step(t, g, kb, 1)
	kb.MouseUp(engine.MouseLeft)
	step(t, g, kb, 1)

	if g.player.Yaw != yaw || dug(g) != 0 {
		t.Errorf("yaw %v, %d blocks broken: wanted nothing to happen", g.player.Yaw, dug(g))
	}
}

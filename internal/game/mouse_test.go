package game

import (
	"testing"

	"glonk.dev/engine"

	"github.com/kyrokohan/gocraft/internal/world"
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

	step(t, g, kb, 6) // a fifth of a second: a click, and not yet a repeat
	if n := dug(g); n != 1 {
		t.Fatalf("%d blocks broken in a fifth of a second, wanted the one click", n)
	}

	step(t, g, kb, 30) // a second more
	if n := dug(g); n < 4 || n > 7 {
		t.Errorf("%d blocks broken with the button held for 1.2 seconds, wanted about five", n)
	}

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

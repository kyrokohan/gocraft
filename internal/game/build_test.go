package game

import (
	"testing"

	"glonk.dev/engine"

	"github.com/kyrokohan/gocraft/internal/world"
)

// lookingDown is a game on the test floor whose player looks at the floor a little
// ahead: at the top of the block at 20, 5, 21.
func lookingDown(t *testing.T) *Game {
	t.Helper()
	g := onFloor(t)
	g.player.Pitch = -0.9
	hold(t, g, 1)
	return g
}

func TestThePlayerAimsAtTheBlockInTheMiddleOfTheView(t *testing.T) {
	g := lookingDown(t)

	want := world.Hit{X: 20, Y: 5, Z: 21, Face: world.FaceYPos, Block: world.Stone}
	got := g.target
	got.T = 0
	if !g.aimed || got != want {
		t.Errorf("aimed %v at %+v, wanted %+v", g.aimed, got, want)
	}
}

func TestNothingIsAimedAtBeyondTheReach(t *testing.T) {
	g := onFloor(t)
	g.player.Pitch = 1.2 // at the sky

	hold(t, g, 1, 'q')
	hold(t, g, 1, 'e')

	if g.aimed {
		t.Errorf("aimed at %+v, wanted nothing", g.target)
	}
}

func TestQBreaksTheBlockAimedAt(t *testing.T) {
	g := lookingDown(t)

	hold(t, g, 1, 'q')

	if got := g.world.At(20, 5, 21); got != world.Air {
		t.Errorf("the block is %v, wanted it gone", got.Info().Name)
	}
	// With that block gone the player aims through the hole, at the side of the next one.
	want := world.Hit{X: 20, Y: 5, Z: 22, Face: world.FaceZNeg, Block: world.Stone}
	got := g.target
	got.T = 0
	if !g.aimed || got != want {
		t.Errorf("after breaking, aimed %v at %+v, wanted %+v", g.aimed, got, want)
	}
}

func TestHoldingQKeepsDigging(t *testing.T) {
	g := lookingDown(t)

	// A held key repeats, and each repeat is a press.
	kb := engine.NewKeyboard()
	for range 3 {
		kb.Press('q')
		if err := g.Update(frame, kb.Input()); err != nil {
			t.Fatal(err)
		}
		kb.EndFrame()
	}

	dug := 0
	for z := range 40 {
		for x := range 40 {
			for y := range 6 {
				if g.world.At(x, y, z) == world.Air {
					dug++
				}
			}
		}
	}
	if dug != 3 {
		t.Errorf("%d blocks of the floor are gone, wanted 3", dug)
	}
}

func TestDeepstoneCannotBeBroken(t *testing.T) {
	g := lookingDown(t)
	g.world.Set(20, 5, 21, world.Deepstone)

	hold(t, g, 1, 'Q')

	if got := g.world.At(20, 5, 21); got != world.Deepstone {
		t.Errorf("the block is %v, wanted deepstone still", got.Info().Name)
	}
}

func TestEPlacesTheSelectedBlockOnTheFaceAimedAt(t *testing.T) {
	g := lookingDown(t)

	hold(t, g, 1, '8')
	hold(t, g, 1, 'e')

	if got := g.world.At(20, 6, 21); got != world.Brick {
		t.Errorf("the block on top is %v, wanted brick", got.Info().Name)
	}
}

func TestABlockCannotBePlacedWhereThePlayerStands(t *testing.T) {
	g := onFloor(t)
	g.player.Pitch = -maxPitch // at the floor under the feet
	hold(t, g, 1)
	if !g.aimed || g.target.X != 20 || g.target.Z != 20 {
		t.Fatalf("aimed %v at %+v, wanted the block underfoot", g.aimed, g.target)
	}

	hold(t, g, 1, 'e')

	if got := g.world.At(20, 6, 20); got != world.Air {
		t.Errorf("a block of %v was placed inside the player", got.Info().Name)
	}
}

func TestABlockCanBePlacedInWater(t *testing.T) {
	g := lookingDown(t)
	g.world.Set(20, 6, 21, world.Water)

	hold(t, g, 1, 'e')

	if got := g.world.At(20, 6, 21); got != hotbar[0] {
		t.Errorf("the block is %v, wanted %v in place of the water", got.Info().Name, hotbar[0].Info().Name)
	}
}

func TestNumberKeysAndTabPickTheBlock(t *testing.T) {
	g := onFloor(t)

	for i := range hotbar {
		hold(t, g, 1, engine.Key('1'+rune(i)))
		if g.selected != i {
			t.Errorf("key %c picked slot %d, wanted %d", '1'+rune(i), g.selected, i)
		}
	}

	hold(t, g, 1, engine.KeyTab)
	if g.selected != 0 {
		t.Errorf("tab on the last slot picked %d, wanted it to go round to 0", g.selected)
	}
	hold(t, g, 1, engine.KeyTab)
	if g.selected != 1 {
		t.Errorf("tab picked %d, wanted 1", g.selected)
	}
}

func TestEveryHotbarBlockCanBeStoodOnAndBroken(t *testing.T) {
	for _, b := range hotbar {
		if info := b.Info(); !info.Solid || info.Fixed || info.Name == "" {
			t.Errorf("%v is in the hotbar: %+v", b, info)
		}
	}
}

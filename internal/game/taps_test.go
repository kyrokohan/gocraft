package game

import (
	"testing"

	"glonk.dev/engine"
)

// held runs the game for a number of frames with a key held down the way a terminal
// reports it: a press, and then a repeat every frame.
func held(t *testing.T, g *Game, kb *engine.Keyboard, frames int, key engine.Key) {
	t.Helper()
	for range frames {
		kb.Press(key) // pressing a key that is already down is a repeat
		if err := g.Update(frame, kb.Input()); err != nil {
			t.Fatalf("Update: %v", err)
		}
		kb.EndFrame()
	}
}

// letGo releases a key and runs one frame, as there is between two presses.
func letGo(t *testing.T, g *Game, kb *engine.Keyboard, key engine.Key) {
	t.Helper()
	kb.Release(key)
	if err := g.Update(frame, kb.Input()); err != nil {
		t.Fatalf("Update: %v", err)
	}
	kb.EndFrame()
}

func TestHoldingFTurnsFlyingOnOnce(t *testing.T) {
	g := onFloor(t)
	kb := engine.NewKeyboard()

	held(t, g, kb, 8, 'f')
	if !g.player.Flying {
		t.Fatal("after holding f for 8 frames the player is not flying: the repeats turned it off again")
	}

	letGo(t, g, kb, 'f')
	held(t, g, kb, 8, 'f')
	if g.player.Flying {
		t.Error("a second press of f, held, did not turn flying off")
	}
}

func TestHoldingEscapePausesOnce(t *testing.T) {
	g := onFloor(t)
	kb := engine.NewKeyboard()

	held(t, g, kb, 8, engine.KeyEscape)
	if g.screen != paused {
		t.Fatalf("after holding Escape for 8 frames the screen is %d, wanted the pause menu", g.screen)
	}

	letGo(t, g, kb, engine.KeyEscape)
	held(t, g, kb, 8, engine.KeyEscape)
	if g.screen != playing {
		t.Errorf("a second press of Escape, held, left the screen at %d, wanted play", g.screen)
	}
}

// Escape on the question goes back to the title, where Escape quits. A held key must
// not do both.
func TestHoldingEscapeOnTheQuestionDoesNotQuit(t *testing.T) {
	g := onFloor(t)
	g.show(confirming)
	kb := engine.NewKeyboard()

	held(t, g, kb, 6, engine.KeyEscape) // fails the test if Update returns ErrQuit

	if g.screen != onTitle {
		t.Errorf("screen %d, wanted the title", g.screen)
	}
}

func TestAKeyHeldAcrossAPauseIsNotANewPress(t *testing.T) {
	g := onFloor(t)
	kb := engine.NewKeyboard()
	held(t, g, kb, 2, 'f') // flying on, f still down

	kb.Press(engine.KeyEscape) // pause, with f held all the while
	held(t, g, kb, 3, 'f')
	kb.Release(engine.KeyEscape)
	held(t, g, kb, 1, 'f')
	kb.Press(engine.KeyEscape) // and resume
	held(t, g, kb, 3, 'f')

	if g.screen != playing || !g.player.Flying {
		t.Errorf("screen %d, flying %v: wanted play, still flying", g.screen, g.player.Flying)
	}
}

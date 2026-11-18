package game

import "testing"

func TestUsingWhatIsHeldSwingsTheHand(t *testing.T) {
	g := lookingDown(t)
	if g.inHand().Swing != 0 || g.inHand().Item != g.items[g.selected].Item {
		t.Fatalf("the hand is %+v before anything is done", g.inHand())
	}
	hold(t, g, 1, 'q')
	hold(t, g, 1) // a swing begins at its start
	if s := g.inHand().Swing; s <= 0 || s >= 1 {
		t.Errorf("breaking a block swung the hand to %.2f, wanted part way", s)
	}
	hold(t, g, seconds(swingTime))
	if s := g.inHand().Swing; s != 0 {
		t.Errorf("the hand is still at %.2f of a swing", s)
	}
}

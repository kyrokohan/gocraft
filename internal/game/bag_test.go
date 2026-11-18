package game

import (
	"testing"

	"glonk.dev/engine"

	"github.com/kyrokohan/gocraft/internal/world"
)

func TestIOpensTheBagAndClosesIt(t *testing.T) {
	g := inSurvival(t)

	hold(t, g, 1, 'i')
	if g.screen != inBag {
		t.Fatalf("screen %d after I, wanted the bag", g.screen)
	}
	if text, _ := drawn(t, g); !contains(text, "your things", "I: close") {
		t.Errorf("the bag is not drawn:\n%s", text)
	}
	before := g.player
	hold(t, g, 5, 'w')
	if g.player != before {
		t.Error("the player walked with the bag open")
	}
	hold(t, g, 1)
	hold(t, g, 1, 'I')
	if g.screen != playing {
		t.Errorf("screen %d after a second I, wanted play", g.screen)
	}
	press(g, 'i')
	press(g, engine.KeyEscape)
	if g.screen != playing {
		t.Errorf("screen %d after Escape in the bag, wanted play", g.screen)
	}
}

func TestTheKeysMoveAStackFromTheHotbarToThePack(t *testing.T) {
	g := inSurvival(t)
	logs := world.Stack{Item: world.Log.Item(), Count: 9}
	g.items[0] = logs
	press(g, 'i') // the keys start on the slot in use: the first of the hotbar

	press(g, engine.KeyEnter)
	if g.hand != logs || g.items[0] != (world.Stack{}) {
		t.Fatalf("holding %+v and the slot holds %+v, wanted the logs in hand", g.hand, g.items[0])
	}
	press(g, engine.KeyUp) // the bottom row of the pack
	press(g, engine.KeyRight)
	press(g, engine.KeyEnter)
	if want := hotbarSlots + 2*hotbarSlots + 1; g.items[want] != logs || g.hand.Count != 0 {
		t.Errorf("slot %d holds %+v and the hand %+v, wanted the logs put down", want, g.items[want], g.hand)
	}
}

func TestPuttingDownOnTheSameKindStacksAndOnAnotherSwaps(t *testing.T) {
	g := inSurvival(t)
	g.items[0] = world.Stack{Item: world.Dirt.Item(), Count: 60}
	g.items[1] = world.Stack{Item: world.Dirt.Item(), Count: 10}
	g.items[2] = world.Stack{Item: world.Sand.Item(), Count: 1}
	press(g, 'i')

	press(g, engine.KeyRight)
	press(g, engine.KeyEnter) // take the 10
	press(g, engine.KeyLeft)
	press(g, engine.KeyEnter) // 4 fit on the 60
	if g.items[0].Count != world.StackSize || g.hand.Count != 6 {
		t.Fatalf("slot 1 holds %d and the hand %d, wanted 64 and 6", g.items[0].Count, g.hand.Count)
	}
	press(g, engine.KeyRight)
	press(g, engine.KeyRight)
	press(g, engine.KeyEnter) // on the sand: swap
	if g.items[2] != (world.Stack{Item: world.Dirt.Item(), Count: 6}) || g.hand != (world.Stack{Item: world.Sand.Item(), Count: 1}) {
		t.Errorf("slot 3 holds %+v and the hand %+v, wanted them swapped", g.items[2], g.hand)
	}

	press(g, 'i') // what is in hand goes back
	if g.hand.Count != 0 || g.items[1] != (world.Stack{Item: world.Sand.Item(), Count: 1}) {
		t.Errorf("after closing, the hand holds %+v and slot 2 %+v, wanted the sand back in", g.hand, g.items[1])
	}
}

func TestWhatInHandDoesNotFitIsPutDown(t *testing.T) {
	g := inSurvival(t)
	for i := range g.items {
		g.items[i] = world.Stack{Item: world.Stone.Item(), Count: world.StackSize}
	}
	press(g, 'i')
	g.hand = world.Stack{Item: world.Sand.Item(), Count: 3}

	press(g, engine.KeyEscape)

	if len(g.drops) != 1 || g.drops[0].Stack != (world.Stack{Item: world.Sand.Item(), Count: 3}) {
		t.Errorf("the 3 sand that did not fit went to %+v", g.drops)
	}
}

func TestAClickInTheBagTakesAndPuts(t *testing.T) {
	g := inSurvival(t)
	g.Resize(100, 30)
	g.items[4] = world.Stack{Item: world.Clay.Item(), Count: 2}
	press(g, 'i')
	click := func(row, col int) {
		x0, top := g.bagOrigin(g.cols, g.rows)
		kb := engine.NewKeyboard()
		kb.MouseMove(x0+col*slotPitch+slotPitch/2, g.bagRowY(top, row)+barHeight/2)
		kb.MouseDown(engine.MouseLeft)
		g.Update(frame, kb.Input())
	}

	click(3, 4)
	click(0, 8)
	if g.items[hotbarSlots+8] != (world.Stack{Item: world.Clay.Item(), Count: 2}) || g.items[4].Count != 0 {
		t.Errorf("the top right of the pack holds %+v and slot 5 %+v, wanted the clay moved", g.items[hotbarSlots+8], g.items[4])
	}
}

func TestTheCreativeBagHasEveryBlockToPutOnTheHotbar(t *testing.T) {
	g := onFloor(t)
	press(g, 'i')
	if text, _ := drawn(t, g); !contains(text, "every block") {
		t.Errorf("the creative bag is not drawn:\n%s", text)
	}
	for b := world.Grass; b.Known(); b++ {
		found := false
		for r := range g.bagRows() - 1 {
			for c := range hotbarSlots {
				if _, item := g.bagCell(r, c); item == b.Item() {
					found = true
				}
			}
		}
		if found != (b != world.Water && b != world.Deepstone) {
			t.Errorf("%v in the creative bag: %v", b.Info().Name, found)
		}
	}

	// The keys start on the hotbar: up is the last row of blocks.
	press(g, engine.KeyUp)
	row, col := g.cursor/hotbarSlots, g.cursor%hotbarSlots
	_, item := g.bagCell(row, col)
	press(g, engine.KeyEnter)
	press(g, engine.KeyDown)
	press(g, engine.KeyEnter)
	if item == world.NoItem || g.items[col].Item != item {
		t.Errorf("slot %d holds %v, wanted the %v taken from the blocks", col+1, g.items[col].Item.Name(), item.Name())
	}
}

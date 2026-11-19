package game

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"glonk.dev/engine"

	"github.com/kyrokohan/skyspear/internal/world"
)

func TestAHearthTakesOnlyWhatItCooksAndBurns(t *testing.T) {
	g := inSurvival(t)
	at := world.Cell{X: 22, Y: 6, Z: 22}
	g.world.Set(at.X, at.Y, at.Z, world.Hearth)
	g.openStore(world.Hearth, at)
	k := g.hearth(at)

	g.hand = world.Stack{Item: world.Meat, Count: 5}
	if g.pickCell(0, hearthFuel); k.Fuel.Count != 0 {
		t.Errorf("meat went in as fuel")
	}
	if g.pickCell(0, hearthIn); k.In != (world.Stack{Item: world.Meat, Count: 5}) || g.hand.Count != 0 {
		t.Errorf("meat to cook: the hearth holds %+v and the hand %+v", k.In, g.hand)
	}
	for _, not := range []world.Item{world.Dirt.Item(), world.Berries, world.Sand.Item()} {
		g.hand = world.Stack{Item: not, Count: 1}
		if g.pickCell(0, hearthIn); k.In.Item != world.Meat {
			t.Errorf("%s went in to cook, which nothing is cooked from", not.Name())
		}
	}
	g.hand = world.Stack{Item: world.PineLog.Item(), Count: 2}
	if g.pickCell(0, hearthFuel); k.Fuel.Count != 2 {
		t.Errorf("a log did not go in as fuel: %+v", k.Fuel)
	}

	k.Out = world.Stack{Item: world.CookedMeat, Count: 2}
	g.hand = world.Stack{Item: world.CookedMeat, Count: 1}
	if g.pickCell(0, hearthOut); g.hand.Count != 3 || k.Out.Count != 0 {
		t.Errorf("taking what it made: the hand holds %+v and the hearth %+v", g.hand, k.Out)
	}
	if g.pickCell(0, hearthOut); g.hand.Count != 3 || k.Out.Count != 0 {
		t.Errorf("something was put where the hearth puts what it makes")
	}

	text, _ := drawn(t, g)
	for _, want := range []string{"hearth", "your things", "makes cooked meat", "cold: drill or torch"} {
		if !strings.Contains(text, want) {
			t.Errorf("the hearth does not show %q:\n%s", want, text)
		}
	}
}

func TestTheKeysStepOverWhereAHearthHasNoSlot(t *testing.T) {
	g := inSurvival(t)
	g.openStore(world.Hearth, world.Cell{X: 22, Y: 6, Z: 22})

	var cols []int
	for range 4 {
		press(g, engine.KeyRight)
		cols = append(cols, g.cursor%hotbarSlots)
	}
	if want := []int{hearthFuel, hearthOut, hearthIn, hearthFuel}; !slices.Equal(cols, want) {
		t.Errorf("Right went to the columns %v, wanted %v", cols, want)
	}
	press(g, engine.KeyDown)
	for range 3 {
		press(g, engine.KeyRight) // to column 4 of the pack, under no slot of the hearth
	}
	press(g, engine.KeyUp)
	if g.cursor != hearthOut {
		t.Errorf("Up from under no slot went to %d, wanted what the hearth made", g.cursor)
	}
}

func TestAPotKeepsWhatIsPutInItAndSpillsItWhenBroken(t *testing.T) {
	g := inSurvival(t)
	at := world.Cell{X: 22, Y: 6, Z: 22}
	g.world.Set(at.X, at.Y, at.Z, world.ClayPot)
	g.items[0] = world.Stack{Item: world.Cobble.Item(), Count: 10}

	if !g.useBlock(world.Hit{X: at.X, Y: at.Y, Z: at.Z, Block: world.ClayPot}) || g.screen != inBag || g.cursor != 0 {
		t.Fatalf("E on a pot: screen %d with the keys on %d, wanted the bag on the pot's first slot", g.screen, g.cursor)
	}
	text, _ := drawn(t, g)
	if !strings.Contains(text, "pot") || !strings.Contains(text, "your things") {
		t.Errorf("the pot is not shown over the player's things:\n%s", text)
	}
	hotbar := g.bagRows() - 1
	g.pickCell(hotbar, 0)
	g.pickCell(1, 4) // the pot's second row
	press(g, 'i')
	if c := g.pot(at); c.Items[hotbarSlots+4] != (world.Stack{Item: world.Cobble.Item(), Count: 10}) || g.items[0].Count != 0 {
		t.Fatalf("the pot holds %+v", c.Items)
	}

	pots, _ := g.savedStores()
	if len(pots) != 1 || pots[0].Cell != at {
		t.Errorf("saved %+v, wanted the one pot", pots)
	}
	g.spill(at)
	if len(g.pots) != 0 || len(g.drops) != 1 || g.drops[0].Stack != (world.Stack{Item: world.Cobble.Item(), Count: 10}) {
		t.Errorf("broken: %d pots kept and %+v put down, wanted the cobble put down", len(g.pots), g.drops)
	}
}

func TestAPotOrAHearthDoesNotOpenInCreative(t *testing.T) {
	g := onFloor(t) // creative
	if g.useBlock(world.Hit{X: 22, Y: 6, Z: 22, Block: world.ClayPot}) || g.screen != playing {
		t.Errorf("a pot opened in creative: screen %d", g.screen)
	}
}

func TestStoresAndTheBedAreSavedAndComeBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "world")
	g := inSurvival(t)
	g.opts.SavePath = path
	g.pot(world.Cell{X: 1, Y: 6, Z: 2}).Items[3] = world.Stack{Item: world.Coal, Count: 4}
	g.pot(world.Cell{X: 3, Y: 6, Z: 2}) // empty, so not saved
	*g.hearth(world.Cell{X: 5, Y: 6, Z: 2}) = world.HearthStore{
		Cell: world.Cell{X: 5, Y: 6, Z: 2}, In: world.Stack{Item: world.Clay.Item(), Count: 2}, Burn: 3, Burns: 9, Cooked: 1,
	}
	g.bed = &world.Cell{X: 7, Y: 6, Z: 2}
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}

	back := savedGame(t, path)

	if len(back.pots) != 1 || back.pot(world.Cell{X: 1, Y: 6, Z: 2}).Items[3].Count != 4 {
		t.Errorf("the pots came back as %v", back.pots)
	}
	if k := back.hearth(world.Cell{X: 5, Y: 6, Z: 2}); k.In.Count != 2 || k.Burn != 3 || k.Cooked != 1 {
		t.Errorf("the hearth came back as %+v", k)
	}
	if back.bed == nil || *back.bed != (world.Cell{X: 7, Y: 6, Z: 2}) {
		t.Errorf("the bed came back as %v", back.bed)
	}
}

func TestABedrollIsWhereThePlayerWakesAndSleepsAwayTheNight(t *testing.T) {
	g := inSurvival(t)
	bed := world.Cell{X: 25, Y: 6, Z: 25}
	g.world.Set(bed.X, bed.Y, bed.Z, world.FurBed)
	lie := func() { g.useBlock(world.Hit{X: bed.X, Y: bed.Y, Z: bed.Z, Block: world.FurBed}) }

	g.days = 0.2 // the middle of the day
	lie()
	if g.bed == nil || *g.bed != bed || g.days != 0.2 || !strings.Contains(g.notice, "wake here") {
		t.Fatalf("by day: bed %v, days %v, told %q: wanted the bed set and no sleep", g.bed, g.days, g.notice)
	}

	g.days = 0.6 // night
	g.fauna.all = []beast{claw(g.player.Pos.X+3, g.player.Pos.Z)}
	lie()
	if g.days != 0.6 || !strings.Contains(g.notice, "hunter") {
		t.Errorf("with a hunter near: days %v, told %q: wanted no sleep, and why", g.days, g.notice)
	}
	g.fauna.all = nil
	lie()
	if !near(timeOfDay(g.days), firstMorning) || g.days < 0.6 || !strings.Contains(g.notice, "morning") {
		t.Errorf("at night: the clock is at %v of a day, told %q: wanted the next morning", timeOfDay(g.days), g.notice)
	}

	g.health = 0
	g.wake()
	if p := g.player.Pos; p.X != 25.5 || p.Y != 6 || p.Z != 25.5 {
		t.Errorf("woke at %v, wanted on the bedroll", p)
	}

	g.world.Set(bed.X, bed.Y, bed.Z, world.Air)
	g.wake()
	if p, spawn := g.player.Pos, g.world.Spawn(); p != spawn || g.bed != nil || !strings.Contains(g.notice, "gone") {
		t.Errorf("with the bedroll gone: woke at %v and told %q, wanted the spawn %v", p, g.notice, spawn)
	}
}

func TestEPlacesABedrollAndThenLiesOnIt(t *testing.T) {
	g := lookingDown(t) // at the top of the block at 20, 5, 21
	g.items[g.selected] = world.Stack{Item: world.FurBed.Item(), Count: 1}

	hold(t, g, 1, 'e')
	if got := g.world.At(20, 6, 21); got != world.FurBed {
		t.Fatalf("E put %s on the floor, wanted a bedroll", got.Info().Name)
	}
	hold(t, g, seconds(1), 'e')
	if g.bed == nil || *g.bed != (world.Cell{X: 20, Y: 6, Z: 21}) || g.world.At(20, 7, 21) != world.Air {
		t.Errorf("E on the bedroll: bed %v, and %s over it: wanted the bed there and nothing built", g.bed, g.world.At(20, 7, 21).Info().Name)
	}

}

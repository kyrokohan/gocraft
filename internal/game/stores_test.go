package game

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"glonk.dev/engine"
	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/world"
)

func TestAKilnCooksAPieceAtATimeWhileItHasFuel(t *testing.T) {
	k := &world.KilnStore{
		In:   world.Stack{Item: world.CopperOre.Item(), Count: 3},
		Fuel: world.Stack{Item: world.Planks.Item(), Count: 1}, // a piece and a half
	}

	stepKiln(k, cookTime-0.1)
	if k.Out.Count != 0 || k.Fuel.Count != 0 || k.Burn == 0 {
		t.Fatalf("before a piece is cooked: made %+v, fuel %+v burning %v", k.Out, k.Fuel, k.Burn)
	}
	stepKiln(k, 6)
	if k.Out != (world.Stack{Item: world.CopperBar, Count: 1}) || k.In.Count != 2 {
		t.Fatalf("after the planks burned: made %+v from %+v, wanted one bar", k.Out, k.In)
	}
	if k.Burn != 0 || !near(k.Cooked, 1.5*cookTime-cookTime) {
		t.Fatalf("the fire has %v left and the next piece %v cooked: wanted no fire, and half a piece kept", k.Burn, k.Cooked)
	}
	stepKiln(k, 60)
	if k.Out.Count != 1 {
		t.Fatalf("with no fuel, it made %+v", k.Out)
	}

	k.Fuel = world.Stack{Item: world.Coal, Count: 1}
	stepKiln(k, cookTime/2+0.1)
	if k.Out.Count != 2 {
		t.Fatalf("coal did not finish the half-cooked piece: made %+v", k.Out)
	}
	stepKiln(k, 10)
	if k.Out.Count != 3 || k.In.Count != 0 || k.Cooked != 0 || k.Burn == 0 {
		t.Errorf("made %+v from what is left, %+v, with the fire at %v: wanted all of it cooked and the coal still burning", k.Out, k.In, k.Burn)
	}
	stepKiln(k, 60)
	if k.Burn != 0 {
		t.Errorf("the fire is at %v a minute on, wanted the coal burned out", k.Burn)
	}
}

func TestAKilnWaitsForEnoughAndForRoom(t *testing.T) {
	k := &world.KilnStore{
		In:   world.Stack{Item: world.Berries, Count: 2},
		Fuel: world.Stack{Item: world.Coal, Count: 3},
	}
	stepKiln(k, 30)
	if k.Out.Count != 0 || k.Fuel.Count != 3 {
		t.Errorf("from 2 berries it made %+v and burned %d coal: a loaf needs 3, and nothing should burn", k.Out, 3-k.Fuel.Count)
	}

	k.In = world.Stack{Item: world.Sand.Item(), Count: 5}
	k.Out = world.Stack{Item: world.Brick.Item(), Count: 1}
	stepKiln(k, 30)
	if k.Out.Item != world.Brick.Item() || k.In.Count != 5 {
		t.Errorf("with brick where the glass goes, it made %+v", k.Out)
	}
}

func TestAKilnTakesOnlyWhatItCooksAndBurns(t *testing.T) {
	g := inSurvival(t)
	at := world.Cell{X: 22, Y: 6, Z: 22}
	g.world.Set(at.X, at.Y, at.Z, world.Kiln)
	g.openStore(world.Kiln, at)
	k := g.kiln(at)

	g.hand = world.Stack{Item: world.Berries, Count: 5}
	if g.pickCell(0, kilnFuel); k.Fuel.Count != 0 {
		t.Errorf("berries went in as fuel")
	}
	if g.pickCell(0, kilnIn); k.In != (world.Stack{Item: world.Berries, Count: 5}) || g.hand.Count != 0 {
		t.Errorf("berries to cook: the kiln holds %+v and the hand %+v", k.In, g.hand)
	}
	g.hand = world.Stack{Item: world.Dirt.Item(), Count: 1}
	if g.pickCell(0, kilnIn); k.In.Item != world.Berries {
		t.Errorf("dirt went in to cook, which nothing is cooked from")
	}
	g.hand = world.Stack{Item: world.PineLog.Item(), Count: 2}
	if g.pickCell(0, kilnFuel); k.Fuel.Count != 2 {
		t.Errorf("a log did not go in as fuel: %+v", k.Fuel)
	}

	k.Out = world.Stack{Item: world.BerryLoaf, Count: 2}
	g.hand = world.Stack{Item: world.BerryLoaf, Count: 1}
	if g.pickCell(0, kilnOut); g.hand.Count != 3 || k.Out.Count != 0 {
		t.Errorf("taking what it made: the hand holds %+v and the kiln %+v", g.hand, k.Out)
	}
	if g.pickCell(0, kilnOut); g.hand.Count != 3 || k.Out.Count != 0 {
		t.Errorf("something was put where the kiln puts what it makes")
	}

	text, _ := drawn(t, g)
	for _, want := range []string{"kiln", "your things", "makes berry loaf", "burns wood or coal"} {
		if !strings.Contains(text, want) {
			t.Errorf("the kiln does not show %q:\n%s", want, text)
		}
	}
}

func TestTheKeysStepOverWhereAKilnHasNoSlot(t *testing.T) {
	g := inSurvival(t)
	g.openStore(world.Kiln, world.Cell{X: 22, Y: 6, Z: 22})

	var cols []int
	for range 4 {
		press(g, engine.KeyRight)
		cols = append(cols, g.cursor%hotbarSlots)
	}
	if want := []int{kilnFuel, kilnOut, kilnIn, kilnFuel}; !slices.Equal(cols, want) {
		t.Errorf("Right went to the columns %v, wanted %v", cols, want)
	}
	press(g, engine.KeyDown)
	for range 3 {
		press(g, engine.KeyRight) // to column 4 of the pack, under no slot of the kiln
	}
	press(g, engine.KeyUp)
	if g.cursor != kilnOut {
		t.Errorf("Up from under no slot went to %d, wanted what the kiln made", g.cursor)
	}
}

func TestACrateKeepsWhatIsPutInItAndSpillsItWhenBroken(t *testing.T) {
	g := inSurvival(t)
	at := world.Cell{X: 22, Y: 6, Z: 22}
	g.world.Set(at.X, at.Y, at.Z, world.Crate)
	g.items[0] = world.Stack{Item: world.Cobble.Item(), Count: 10}

	if !g.useBlock(world.Hit{X: at.X, Y: at.Y, Z: at.Z, Block: world.Crate}) || g.screen != inBag || g.cursor != 0 {
		t.Fatalf("E on a crate: screen %d with the keys on %d, wanted the bag on the crate's first slot", g.screen, g.cursor)
	}
	text, _ := drawn(t, g)
	if !strings.Contains(text, "crate") || !strings.Contains(text, "your things") {
		t.Errorf("the crate is not shown over the player's things:\n%s", text)
	}
	hotbar := g.bagRows() - 1
	g.pickCell(hotbar, 0)
	g.pickCell(1, 4) // the crate's second row
	press(g, 'i')
	if c := g.crate(at); c.Items[hotbarSlots+4] != (world.Stack{Item: world.Cobble.Item(), Count: 10}) || g.items[0].Count != 0 {
		t.Fatalf("the crate holds %+v", c.Items)
	}

	crates, _ := g.savedStores()
	if len(crates) != 1 || crates[0].Cell != at {
		t.Errorf("saved %+v, wanted the one crate", crates)
	}
	g.spill(at)
	if len(g.crates) != 0 || len(g.drops) != 1 || g.drops[0].Stack != (world.Stack{Item: world.Cobble.Item(), Count: 10}) {
		t.Errorf("broken: %d crates kept and %+v put down, wanted the cobble put down", len(g.crates), g.drops)
	}
}

func TestACrateOrAKilnDoesNotOpenInCreative(t *testing.T) {
	g := onFloor(t) // creative
	if g.useBlock(world.Hit{X: 22, Y: 6, Z: 22, Block: world.Crate}) || g.screen != playing {
		t.Errorf("a crate opened in creative: screen %d", g.screen)
	}
}

func TestStoresAndTheBedAreSavedAndComeBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "world")
	g := inSurvival(t)
	g.opts.SavePath = path
	g.crate(world.Cell{X: 1, Y: 6, Z: 2}).Items[3] = world.Stack{Item: world.Coal, Count: 4}
	g.crate(world.Cell{X: 3, Y: 6, Z: 2}) // empty, so not saved
	*g.kiln(world.Cell{X: 5, Y: 6, Z: 2}) = world.KilnStore{
		Cell: world.Cell{X: 5, Y: 6, Z: 2}, In: world.Stack{Item: world.Clay.Item(), Count: 2}, Burn: 3, Burns: 9, Cooked: 1,
	}
	g.bed = &world.Cell{X: 7, Y: 6, Z: 2}
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}

	back := savedGame(t, path)

	if len(back.crates) != 1 || back.crate(world.Cell{X: 1, Y: 6, Z: 2}).Items[3].Count != 4 {
		t.Errorf("the crates came back as %v", back.crates)
	}
	if k := back.kiln(world.Cell{X: 5, Y: 6, Z: 2}); k.In.Count != 2 || k.Burn != 3 || k.Cooked != 1 {
		t.Errorf("the kiln came back as %+v", k)
	}
	if back.bed == nil || *back.bed != (world.Cell{X: 7, Y: 6, Z: 2}) {
		t.Errorf("the bed came back as %v", back.bed)
	}
}

func TestABedrollIsWhereThePlayerWakesAndSleepsAwayTheNight(t *testing.T) {
	g := inSurvival(t)
	bed := world.Cell{X: 25, Y: 6, Z: 25}
	g.world.Set(bed.X, bed.Y, bed.Z, world.Bedroll)
	lie := func() { g.useBlock(world.Hit{X: bed.X, Y: bed.Y, Z: bed.Z, Block: world.Bedroll}) }

	g.days = 0.2 // the middle of the day
	lie()
	if g.bed == nil || *g.bed != bed || g.days != 0.2 || !strings.Contains(g.notice, "wake here") {
		t.Fatalf("by day: bed %v, days %v, told %q: wanted the bed set and no sleep", g.bed, g.days, g.notice)
	}

	g.days = 0.6 // night
	g.gloams = []gloam{{pos: g.player.Pos.Add(geometry.Vector3{X: 3}), health: gloamHealth}}
	lie()
	if g.days != 0.6 || !strings.Contains(g.notice, "gloam") {
		t.Errorf("with a gloam near: days %v, told %q: wanted no sleep, and why", g.days, g.notice)
	}
	g.gloams = nil
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
	g.items[g.selected] = world.Stack{Item: world.Bedroll.Item(), Count: 1}

	hold(t, g, 1, 'e')
	if got := g.world.At(20, 6, 21); got != world.Bedroll {
		t.Fatalf("E put %s on the floor, wanted a bedroll", got.Info().Name)
	}
	hold(t, g, seconds(1), 'e')
	if g.bed == nil || *g.bed != (world.Cell{X: 20, Y: 6, Z: 21}) || g.world.At(20, 7, 21) != world.Air {
		t.Errorf("E on the bedroll: bed %v, and %s over it: wanted the bed there and nothing built", g.bed, g.world.At(20, 7, 21).Info().Name)
	}

}

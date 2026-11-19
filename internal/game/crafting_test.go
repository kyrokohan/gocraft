package game

import (
	"testing"

	"glonk.dev/engine"

	"github.com/kyrokohan/skyspear/internal/world"
)

// recipeFor returns the recipe of an item, if it can be made where the player stands.
func recipeFor(g *Game, item world.Item) (recipe, bool) {
	for _, r := range g.makeable() {
		if r.makes == item {
			return r, true
		}
	}
	return recipe{}, false
}

func TestALogIsMadeIntoSticksByHandAndNothingMakesPlanks(t *testing.T) {
	g := inSurvival(t)
	g.items[4] = world.Stack{Item: world.PineLog.Item(), Count: 1}

	r, ok := recipeFor(g, world.Stick)
	if !ok || !g.craft(r) {
		t.Fatal("sticks could not be made from a log by hand")
	}
	if g.items[0] != (world.Stack{Item: world.Stick, Count: 4}) || g.items[4] != (world.Stack{}) {
		t.Errorf("after making sticks, slot 1 holds %+v and slot 5 %+v", g.items[0], g.items[4])
	}
	if g.craft(r) {
		t.Error("sticks were made with no log left")
	}
	for _, old := range []world.Item{world.Planks.Item(), world.WoodPick, world.StonePick, world.CopperPick, world.IronPick, world.ClayPot.Item()} {
		for _, r := range recipes {
			if r.makes == old {
				t.Errorf("%s can still be made", old.Name())
			}
		}
	}
}

func TestFernsLeaveFiberAndFiberIsTwistedIntoCord(t *testing.T) {
	g := inSurvival(t)
	g.leave(20, 6, 21, world.Fern)
	g.leave(21, 6, 21, world.TallFern)
	fiber := 0
	for _, d := range g.drops {
		if d.Item == world.Fiber {
			fiber += d.Count
		}
	}
	if fiber != 3 {
		t.Fatalf("a fern and a tall fern left %d fiber, wanted 3", fiber)
	}

	g.items[0] = world.Stack{Item: world.Fiber, Count: 3}
	if r, ok := recipeFor(g, world.Cord); !ok || !g.craft(r) {
		t.Fatal("cord could not be made from three fibers by hand")
	}
	if g.items[0] != (world.Stack{Item: world.Cord, Count: 1}) {
		t.Errorf("after twisting cord the first slot holds %+v", g.items[0])
	}
}

func TestFlintIsKnappedOnlyNearAKnappingStone(t *testing.T) {
	g := inSurvival(t)
	g.items[0] = world.Stack{Item: world.Flint, Count: 3}
	g.items[1] = world.Stack{Item: world.Stick, Count: 2}
	g.items[2] = world.Stack{Item: world.Cord, Count: 1}

	if _, ok := recipeFor(g, world.FlintPick); ok {
		t.Fatal("a pick can be made with no knapping stone")
	}
	g.world.Set(26, 6, 26, world.KnappingStone)
	if _, ok := recipeFor(g, world.FlintPick); ok {
		t.Fatal("a pick can be made with the knapping stone out of reach")
	}
	g.world.Set(22, 6, 22, world.KnappingStone)
	r, ok := recipeFor(g, world.FlintPick)
	if !ok || !g.craft(r) {
		t.Fatal("a pick could not be made beside a knapping stone")
	}
	if g.items[0] != (world.Stack{Item: world.FlintPick, Count: 1}) || g.items[1] != (world.Stack{}) || g.items[2] != (world.Stack{}) {
		t.Errorf("after making a pick, the first slots hold %+v", g.items[:3])
	}
}

// The first knapping stone is made of stone dug by hand, and the tools of wood need
// no stone at all.
func TestTheFirstToolsNeedNoToolToMake(t *testing.T) {
	g := inSurvival(t)
	g.items[0] = world.Stack{Item: world.Cobble.Item(), Count: 4}
	g.items[1] = world.Stack{Item: world.Stick, Count: 5}
	g.items[2] = world.Stack{Item: world.Log.Item(), Count: 1}
	for _, item := range []world.Item{world.KnappingStone.Item(), world.DiggingStick, world.Spear, world.Club} {
		if r, ok := recipeFor(g, item); !ok || !g.craft(r) {
			t.Errorf("%s could not be made by hand", item.Name())
		}
	}
}

func TestTheKeysMakeAThingFromTheList(t *testing.T) {
	g := inSurvival(t)
	g.items[3] = world.Stack{Item: world.Log.Item(), Count: 2}
	sticks, digger := []world.Item{world.Stick}, []world.Item{world.DiggingStick}

	press(g, 'i')
	for range hotbarSlots {
		press(g, engine.KeyRight) // to the last column, and on into the list
	}
	if !g.inList {
		t.Fatal("Right from the last column did not go into the list")
	}
	if text, _ := drawn(t, g); !contains(text, "make by hand", "stick ×4", "from log") {
		t.Errorf("the list and what its first line needs are not shown:\n%s", text)
	}
	press(g, engine.KeyEnter)
	if g.items.count(sticks) != 4 || g.items[3].Count != 1 {
		t.Fatalf("Enter on the first line left %+v", g.items[:5])
	}
	for range 3 {
		press(g, engine.KeyDown) // past cord and the knapping stone, to the digging stick
	}
	press(g, engine.KeyEnter)
	if g.items.count(digger) != 1 || g.items.count(sticks) != 2 {
		t.Errorf("Enter on the fourth line left %+v", g.items[:5])
	}

	press(g, engine.KeyLeft)
	if g.inList || g.cursor%hotbarSlots != hotbarSlots-1 {
		t.Errorf("Left did not go back to the last column: in the list %v, on slot %d", g.inList, g.cursor)
	}
}

func TestAClickOnTheListMakesAThing(t *testing.T) {
	g := inSurvival(t)
	g.items[0] = world.Stack{Item: world.Cobble.Item(), Count: 4}
	g.Resize(100, 30)
	press(g, 'i')

	x, y, _ := g.listBox(g.cols, g.rows)
	kb := engine.NewKeyboard()
	kb.MouseMove(x+4, y+3) // the third line: a knapping stone
	kb.MouseDown(engine.MouseLeft)
	g.Update(frame, kb.Input())

	if g.items[0] != (world.Stack{Item: world.KnappingStone.Item(), Count: 1}) || g.recipe != 2 {
		t.Errorf("the click left %+v, on line %d", g.items[:2], g.recipe)
	}
}

func TestStoneDugByHandTakesLongAndLeavesRubble(t *testing.T) {
	g := digging(t, world.Stone)
	hand, yields := world.Stone.DigWith(world.NoItem)
	quick, _ := world.Stone.DigWith(world.FlintPick)
	if !yields || hand < 4 || quick > hand/2 {
		t.Fatalf("stone by hand takes %v s and yields %v, and with a flint pick %v s", hand, yields, quick)
	}

	hold(t, g, seconds(hand)-3, 'q')
	if g.world.At(20, 5, 21) != world.Stone {
		t.Fatal("the stone broke before its time by hand")
	}
	hold(t, g, 4, 'q')
	if g.world.At(20, 5, 21) != world.Air || len(g.drops) != 1 || g.drops[0].Item != world.Cobble.Item() {
		t.Errorf("by hand the stone is %v and left %+v, wanted it gone and rubble left", g.world.At(20, 5, 21).Info().Name, g.drops)
	}

	coal := digging(t, world.CoalOre)
	slow, _ := world.CoalOre.DigWith(world.NoItem)
	hold(t, coal, seconds(slow)+2, 'q')
	if coal.world.At(20, 5, 21) != world.Air || len(coal.drops) != 0 {
		t.Errorf("by hand the coal is %v and left %+v, wanted it gone and nothing left", coal.world.At(20, 5, 21).Info().Name, coal.drops)
	}
}

func TestAToolWearsAndIsGoneWhenWornOut(t *testing.T) {
	g := digging(t, world.Dirt)
	g.items[0] = world.Stack{Item: world.WoodShovel, Count: 1}
	fast, _ := world.Dirt.DigWith(world.WoodShovel)

	hold(t, g, seconds(max(fast, minDig))+1, 'q')
	if g.world.At(20, 5, 21) != world.Air || g.items[0].Wear != 1 {
		t.Fatalf("after a dig with the shovel the dirt is %v and the shovel %+v", g.world.At(20, 5, 21).Info().Name, g.items[0])
	}

	g.world.Set(20, 5, 21, world.Dirt)
	g.items[0].Wear = world.WoodShovel.Uses() - 1
	hold(t, g, 1)
	hold(t, g, seconds(max(fast, minDig))+1, 'q')
	if g.items[0] != (world.Stack{}) || len(g.drops) != 2 {
		t.Errorf("the last dig left the shovel %+v and %d drops", g.items[0], len(g.drops))
	}
}

func TestToolsDoNotStack(t *testing.T) {
	g := inSurvival(t)
	pick := world.Stack{Item: world.WoodPick, Count: 1}
	g.items.add(pick)
	g.items.add(world.Stack{Item: world.WoodPick, Count: 1, Wear: 9})
	if g.items[0] != pick || g.items[1] != (world.Stack{Item: world.WoodPick, Count: 1, Wear: 9}) {
		t.Fatalf("two picks went in as %+v", g.items[:3])
	}

	// One put down on the other changes places with it.
	press(g, 'i')
	press(g, engine.KeyEnter)
	press(g, engine.KeyRight)
	press(g, engine.KeyEnter)
	if g.hand != (world.Stack{Item: world.WoodPick, Count: 1, Wear: 9}) || g.items[1] != pick {
		t.Errorf("holding %+v with %+v in the slot, wanted the picks swapped", g.hand, g.items[1])
	}
}

func TestSomeLeavesLeaveAFruit(t *testing.T) {
	g := inSurvival(t)
	x := 0
	for ; x < 1000; x++ {
		if item, _ := world.Leaves.DropAt(x, 5, 0); item == world.Fruit {
			break
		}
	}
	g.leave(x, 5, 0, world.Leaves)
	if len(g.drops) != 1 || g.drops[0].Item != world.Fruit {
		t.Errorf("the leaves at %d, 5, 0 left %+v, wanted a fruit", x, g.drops)
	}
}

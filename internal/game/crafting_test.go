package game

import (
	"testing"

	"glonk.dev/engine"

	"github.com/kyrokohan/gocraft/internal/world"
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

func TestALogIsMadeIntoPlanksByHand(t *testing.T) {
	g := inSurvival(t)
	g.items[4] = world.Stack{Item: world.PineLog.Item(), Count: 1}

	r, ok := recipeFor(g, world.Planks.Item())
	if !ok || !g.craft(r) {
		t.Fatal("planks could not be made from a log by hand")
	}
	if g.items[0] != (world.Stack{Item: world.Planks.Item(), Count: 4}) || g.items[4] != (world.Stack{}) {
		t.Errorf("after making planks, slot 1 holds %+v and slot 5 %+v", g.items[0], g.items[4])
	}
	if g.craft(r) {
		t.Error("planks were made with no log left")
	}
}

func TestAToolIsMadeOnlyNearAWorkbench(t *testing.T) {
	g := inSurvival(t)
	g.items[0] = world.Stack{Item: world.Planks.Item(), Count: 3}
	g.items[1] = world.Stack{Item: world.Stick, Count: 2}

	if _, ok := recipeFor(g, world.WoodPick); ok {
		t.Fatal("a pick can be made with no workbench")
	}
	g.world.Set(26, 6, 26, world.Workbench)
	if _, ok := recipeFor(g, world.WoodPick); ok {
		t.Fatal("a pick can be made with the workbench out of reach")
	}
	g.world.Set(22, 6, 22, world.Workbench)
	r, ok := recipeFor(g, world.WoodPick)
	if !ok || !g.craft(r) {
		t.Fatal("a pick could not be made beside a workbench")
	}
	if g.items[0] != (world.Stack{Item: world.WoodPick, Count: 1}) || g.items[1] != (world.Stack{}) {
		t.Errorf("after making a pick, the first slots hold %+v", g.items[:2])
	}
}

func TestAKilnBurnsFuelForWhatItMakes(t *testing.T) {
	g := inSurvival(t)
	g.world.Set(22, 6, 22, world.Kiln)
	g.items[0] = world.Stack{Item: world.CopperOre.Item(), Count: 2}
	g.items[1] = world.Stack{Item: world.Planks.Item(), Count: 1}
	bar := []world.Item{world.CopperBar}

	r, ok := recipeFor(g, world.CopperBar)
	if !ok || !g.craft(r) {
		t.Fatal("a bar could not be made at a kiln")
	}
	if g.items.count(bar) != 1 || g.items[0].Count != 1 || g.items[1].Item == world.Planks.Item() {
		t.Fatalf("after making a bar, the first slots hold %+v", g.items[:2])
	}
	if g.craft(r) {
		t.Fatal("a bar was made with nothing to burn")
	}
	if g.items[0].Count != 1 {
		t.Error("the ore was used though nothing was made")
	}
	g.items[5] = world.Stack{Item: world.Coal, Count: 1}
	if !g.craft(r) || g.items[5] != (world.Stack{}) || g.items.count(bar) != 2 {
		t.Errorf("coal did not burn for a second bar: %+v", g.items[:6])
	}
}

func TestTheKeysMakeAThingFromTheList(t *testing.T) {
	g := inSurvival(t)
	g.items[3] = world.Stack{Item: world.Log.Item(), Count: 2}
	planks, sticks := []world.Item{world.Planks.Item()}, []world.Item{world.Stick}

	press(g, 'i')
	for range hotbarSlots {
		press(g, engine.KeyRight) // to the last column, and on into the list
	}
	if !g.inList {
		t.Fatal("Right from the last column did not go into the list")
	}
	if text, _ := drawn(t, g); !contains(text, "make by hand", "planks ×4", "from log") {
		t.Errorf("the list and what its first line needs are not shown:\n%s", text)
	}
	press(g, engine.KeyEnter)
	if g.items.count(planks) != 4 || g.items[3].Count != 1 {
		t.Fatalf("Enter on the first line left %+v", g.items[:5])
	}
	press(g, engine.KeyDown)
	press(g, engine.KeyEnter)
	if g.items.count(sticks) != 4 || g.items.count(planks) != 2 {
		t.Errorf("Enter on the second line left %+v", g.items[:5])
	}

	press(g, engine.KeyLeft)
	if g.inList || g.cursor%hotbarSlots != hotbarSlots-1 {
		t.Errorf("Left did not go back to the last column: in the list %v, on slot %d", g.inList, g.cursor)
	}
}

func TestAClickOnTheListMakesAThing(t *testing.T) {
	g := inSurvival(t)
	g.items[0] = world.Stack{Item: world.Planks.Item(), Count: 4}
	g.Resize(100, 30)
	press(g, 'i')

	x, y, _ := g.listBox(g.cols, g.rows)
	kb := engine.NewKeyboard()
	kb.MouseMove(x+4, y+3) // the third line: a workbench
	kb.MouseDown(engine.MouseLeft)
	g.Update(frame, kb.Input())

	if g.items[0] != (world.Stack{Item: world.Workbench.Item(), Count: 1}) || g.recipe != 2 {
		t.Errorf("the click left %+v, on line %d", g.items[:2], g.recipe)
	}
}

func TestStoneDugByHandTakesLongAndLeavesNothing(t *testing.T) {
	g := digging(t, world.Stone)
	hand, yields := world.Stone.DigWith(world.NoItem)
	if yields || hand <= world.Stone.Info().Dig {
		t.Fatalf("stone by hand takes %v s and yields %v", hand, yields)
	}

	hold(t, g, seconds(hand)-3, 'q')
	if g.world.At(20, 5, 21) != world.Stone {
		t.Fatal("the stone broke before its time by hand")
	}
	hold(t, g, 4, 'q')
	if g.world.At(20, 5, 21) != world.Air || len(g.drops) != 0 {
		t.Errorf("by hand the stone is %v and left %+v, wanted it gone and nothing", g.world.At(20, 5, 21).Info().Name, g.drops)
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

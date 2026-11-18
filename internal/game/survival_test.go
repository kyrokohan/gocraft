package game

import (
	"math"
	"path/filepath"
	"strings"
	"testing"

	"glonk.dev/engine"
	"glonk.dev/geometry"

	"github.com/kyrokohan/gocraft/internal/world"
)

// inSurvival is a game on the test floor played in survival, with nothing carried.
func inSurvival(t *testing.T) *Game {
	t.Helper()
	g := onFloor(t)
	g.begin(true)
	return g
}

// digging is inSurvival, with the player looking at the top of the block at 20, 5, 21,
// which is made the given block.
func digging(t *testing.T, b world.Block) *Game {
	t.Helper()
	g := inSurvival(t)
	g.world.Set(20, 5, 21, b)
	g.player.Pitch = -0.9
	hold(t, g, 1)
	if !g.aimed || g.target.X != 20 || g.target.Y != 5 || g.target.Z != 21 {
		t.Fatalf("aimed %v at %+v, wanted the block at 20, 5, 21", g.aimed, g.target)
	}
	return g
}

func seconds(s float64) int {
	return int(math.Round(s / frame))
}

// contains reports whether text holds every one of the words.
func contains(text string, words ...string) bool {
	for _, w := range words {
		if !strings.Contains(text, w) {
			return false
		}
	}
	return true
}

func mustFrame(t *testing.T) *engine.FrameBuffer {
	t.Helper()
	fb, err := engine.NewFrameBuffer(100, 30)
	if err != nil {
		t.Fatal(err)
	}
	return fb
}

func TestInSurvivalABlockTakesItsTimeToDig(t *testing.T) {
	g := digging(t, world.Stone)

	hold(t, g, seconds(world.Stone.Info().Dig)-3, 'q')
	if got := g.world.At(20, 5, 21); got != world.Stone {
		t.Fatalf("the stone is %v before its time was up", got.Info().Name)
	}
	g.Draw(mustFrame(t))
	if part := g.rend.Crack.Part; part < 0.8 || part >= 1 {
		t.Errorf("nearly dug, the stone is drawn %v of the way cracked", part)
	}

	hold(t, g, 4, 'q')
	if got := g.world.At(20, 5, 21); got != world.Air {
		t.Errorf("the stone is %v once its time was up", got.Info().Name)
	}
	if len(g.drops) != 1 || g.drops[0].Item != world.Cobble.Item() || g.drops[0].Count != 1 {
		t.Errorf("the stone left %+v, wanted one cobble", g.drops)
	}
}

func TestLettingGoOrLookingAwayStartsTheDigAgain(t *testing.T) {
	g := digging(t, world.Dirt)
	most := seconds(world.Dirt.Info().Dig) - 3

	hold(t, g, most, 'q')
	hold(t, g, 1) // let go
	hold(t, g, most, 'q')
	if got := g.world.At(20, 5, 21); got != world.Dirt {
		t.Fatalf("the dirt broke after two short digs: %v", got.Info().Name)
	}

	hold(t, g, 1)
	hold(t, g, most, 'q')
	g.player.Pitch = -1.4 // the block under the feet
	hold(t, g, 4, 'q')
	if got := g.world.At(20, 5, 21); got != world.Dirt {
		t.Errorf("the dirt broke though the dig moved on to another block: %v", got.Info().Name)
	}
	g.Draw(mustFrame(t))
	if g.rend.Crack.Part == 0 {
		t.Error("the block now dug is drawn with no cracks")
	}
}

func TestAHeldButtonDigsAndADragDoesNot(t *testing.T) {
	g := digging(t, world.Dirt)
	g.Resize(100, 30)
	kb := engine.NewKeyboard()
	kb.MouseMove(50, 15)
	kb.MouseDown(engine.MouseLeft)
	for range seconds(world.Dirt.Info().Dig) + 2 {
		g.Update(frame, kb.Input())
		kb.EndFrame()
	}
	if got := g.world.At(20, 5, 21); got != world.Air {
		t.Fatalf("the button was held on the dirt and it is still %v", got.Info().Name)
	}

	g = digging(t, world.Dirt)
	g.Resize(100, 30)
	kb = engine.NewKeyboard()
	kb.MouseMove(50, 15)
	kb.MouseDown(engine.MouseLeft)
	for i := range seconds(world.Dirt.Info().Dig) + 2 {
		kb.MouseMove(50+i, 15) // a drag
		g.Update(frame, kb.Input())
		kb.EndFrame()
	}
	if g.world.At(20, 5, 21) != world.Dirt {
		t.Error("a drag dug the block")
	}
}

func TestWhatABlockLeavesIsPickedUpWhenThePlayerComesNear(t *testing.T) {
	g := digging(t, world.Dirt)
	hold(t, g, seconds(world.Dirt.Info().Dig)+1, 'q')
	if len(g.drops) != 1 || g.items[0].Count != 0 {
		t.Fatalf("drops %+v, carried %+v: wanted the dirt on the ground", g.drops, g.items[0])
	}

	hold(t, g, 20, 'w') // into the hole it left
	hold(t, g, 20)

	if len(g.drops) != 0 || g.items[0] != (world.Stack{Item: world.Dirt.Item(), Count: 1}) {
		t.Errorf("drops %+v, carried %+v: wanted the dirt in the first slot", g.drops, g.items[0])
	}
}

func TestADropFallsToTheGroundAndIsGoneInTime(t *testing.T) {
	g := inSurvival(t)
	g.drops = []drop{{Drop: world.Drop{Stack: world.Stack{Item: world.Sand.Item(), Count: 3}, Pos: geometry.Vector3{X: 30.5, Y: 12, Z: 30.5}}}}

	hold(t, g, 60)
	if len(g.drops) != 1 || !near(g.drops[0].Pos.Y, 6) {
		t.Fatalf("after two seconds the drop is %+v, wanted it on the floor at 6", g.drops)
	}
	g.drops[0].Age = dropLife - frame/2
	hold(t, g, 1)
	if len(g.drops) != 0 {
		t.Errorf("an old drop is still there: %+v", g.drops)
	}
}

func TestABushGivesBerriesThatAreEaten(t *testing.T) {
	g := inSurvival(t)
	g.world.Set(20, 6, 21, world.Bush)
	g.player.Pitch = -0.9
	hold(t, g, 1)
	if !g.aimed || g.target.Block != world.Bush {
		t.Fatalf("aimed %v at %+v, wanted the bush", g.aimed, g.target)
	}
	hold(t, g, seconds(minDig)+1, 'q')
	hold(t, g, 30)
	if g.items[0] != (world.Stack{Item: world.Berries, Count: 2}) {
		t.Fatalf("the bush gave %+v, wanted 2 berries", g.items[0])
	}

	g.food = 10
	hold(t, g, 1, 'e')
	if want := 10 + float64(world.Berries.Food()); g.food != want || g.items[0].Count != 1 {
		t.Errorf("after eating, food %v and %d berries, wanted %v and 1", g.food, g.items[0].Count, want)
	}

	g.food = maxFood
	hold(t, g, 1)
	hold(t, g, 1, 'e')
	if g.items[0].Count != 1 {
		t.Error("a berry was eaten with no room for it")
	}
}

func TestInSurvivalABlockPlacedIsUsedUp(t *testing.T) {
	g := digging(t, world.Stone)
	g.items[0] = world.Stack{Item: world.Planks.Item(), Count: 2}

	for range 2 {
		hold(t, g, 1)
		hold(t, g, 1, 'e')
		if got := g.world.At(20, 6, 21); got != world.Planks {
			t.Fatalf("the block placed is %v, wanted planks", got.Info().Name)
		}
		g.world.Set(20, 6, 21, world.Air)
	}
	if g.items[0] != (world.Stack{}) {
		t.Fatalf("after placing both, the slot holds %+v", g.items[0])
	}
	hold(t, g, 1)
	hold(t, g, 1, 'e')
	if g.world.At(20, 6, 21) != world.Air {
		t.Error("a block was placed from an empty slot")
	}
}

func TestAFallHurtsByHowFarItWent(t *testing.T) {
	for _, tc := range []struct {
		name   string
		height float64
		wet    bool
		health float64
	}{
		{"ten blocks", 10, false, maxHealth - 7},
		{"three blocks", 3, false, maxHealth},
		{"ten into water", 10, true, maxHealth},
	} {
		g := inSurvival(t)
		if tc.wet {
			g.world.Set(20, 6, 20, world.Water)
		}
		g.player.Pos.Y = 6 + tc.height

		hold(t, g, 60)

		if g.health != tc.health || !g.player.OnGround {
			t.Errorf("%s: health %v, on the ground %v: wanted %v", tc.name, g.health, g.player.OnGround, tc.health)
		}
	}
}

func TestUnderWaterTheAirRunsOutAndThenItHurts(t *testing.T) {
	g := inSurvival(t)
	for y := 6; y < 10; y++ {
		g.world.Set(20, y, 20, world.Water)
	}

	hold(t, g, seconds(maxAir-1))
	if g.health != maxHealth || g.air <= 0 {
		t.Fatalf("health %v and air %v before the air ran out", g.health, g.air)
	}
	hold(t, g, seconds(3))
	if g.air != 0 || g.health >= maxHealth-2 {
		t.Errorf("health %v and air %v two seconds after the air ran out", g.health, g.air)
	}

	for y := 6; y < 10; y++ {
		g.world.Set(20, y, 20, world.Air)
	}
	hold(t, g, seconds(maxAir/breathBack)+1)
	if g.air != maxAir {
		t.Errorf("out of the water the air came back to %v", g.air)
	}
}

func TestFoodGoesDownAndAFullStomachHeals(t *testing.T) {
	g := inSurvival(t)
	g.health = 10

	hold(t, g, seconds(mendTime)+1)
	if g.health != 11 || g.food != maxFood {
		t.Fatalf("health %v and food %v after a mend, wanted 11 and full", g.health, g.food)
	}
	hold(t, g, seconds(hungerTime-mendTime))
	if g.food != maxFood-1 {
		t.Errorf("food %v after %v s, wanted one gone", g.food, hungerTime)
	}

	g.food, g.health = mendFood-1, 10
	hold(t, g, seconds(2*mendTime))
	if g.health != 10 {
		t.Errorf("health %v with food under %v, wanted no mending", g.health, mendFood)
	}
}

func TestHungerHurtsButCannotKill(t *testing.T) {
	g := inSurvival(t)
	g.food, g.health = 0, 3

	hold(t, g, seconds(5*mendTime))

	if g.health != 1 || g.screen != playing {
		t.Errorf("health %v, screen %d after a long hunger: wanted 1 and still playing", g.health, g.screen)
	}
}

func TestAPlayerWithNoHealthFaintsAndWakesWithItsThings(t *testing.T) {
	g := inSurvival(t)
	g.items[3] = world.Stack{Item: world.Log.Item(), Count: 5}
	g.player.Pos = geometry.Vector3{X: 30.5, Y: 6, Z: 30.5}
	g.food = 4

	g.hurt(maxHealth + 5)
	if g.screen != fainted || g.health != 0 {
		t.Fatalf("screen %d, health %v: wanted a faint", g.screen, g.health)
	}
	if text, _ := drawn(t, g); !contains(text, "you fainted", itemWake) {
		t.Errorf("the faint is not shown:\n%s", text)
	}
	before := g.player.Pos
	hold(t, g, 10, 'w')
	if g.player.Pos != before {
		t.Error("the player moved while fainted")
	}

	press(g, engine.KeyEnter) // Wake up
	if g.screen != playing || g.health != maxHealth || g.food != maxFood || g.air != maxAir {
		t.Errorf("screen %d, health %v, food %v, air %v: wanted play, whole again", g.screen, g.health, g.food, g.air)
	}
	if spawn := g.world.Spawn(); g.player.Pos != spawn {
		t.Errorf("woke at %v, wanted the spawn at %v", g.player.Pos, spawn)
	}
	if g.items[3] != (world.Stack{Item: world.Log.Item(), Count: 5}) {
		t.Errorf("woke carrying %+v, wanted the 5 logs", g.items[3])
	}
}

func TestOnlyACreativePlayerFlies(t *testing.T) {
	g := inSurvival(t)
	hold(t, g, 1, 'f')
	if g.player.Flying {
		t.Error("f turned flying on in survival")
	}
}

func TestInCreativeNothingHurtsAndBlocksHaveNoEnd(t *testing.T) {
	g := onFloor(t)
	g.player.Pos.Y = 30
	hold(t, g, 60)
	if g.health != maxHealth {
		t.Errorf("a fall in creative left health %v", g.health)
	}

	g = lookingDown(t)
	hold(t, g, 1, '8')
	hold(t, g, 1, 'e')
	hold(t, g, 1)
	hold(t, g, 1, 'q')
	if g.items[7] != (world.Stack{Item: world.Brick.Item(), Count: 1}) || len(g.drops) != 0 {
		t.Errorf("after placing a brick and breaking it, slot 8 holds %+v and %d things lie about", g.items[7], len(g.drops))
	}
}

func TestSurvivalIsSavedWithItsThingsAndBody(t *testing.T) {
	path := filepath.Join(t.TempDir(), "world")
	g := savedGame(t, path)
	startNew(t, g, true)
	g.items[0] = world.Stack{Item: world.Berries, Count: 7}
	g.items[20] = world.Stack{Item: world.Cobble.Item(), Count: 64}
	g.health, g.food, g.air = 12, 9, 3
	g.drops = []drop{{Drop: world.Drop{Stack: world.Stack{Item: world.Sand.Item(), Count: 2}, Pos: g.player.Pos, Age: 4}}}
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}

	back := savedGame(t, path)
	if !back.survival || back.items != g.items || back.health != 12 || back.food != 9 || back.air != 3 {
		t.Errorf("came back in survival %v with %+v, health %v, food %v, air %v",
			back.survival, back.items[:3], back.health, back.food, back.air)
	}
	if len(back.drops) != 1 || back.drops[0].Drop != g.drops[0].Drop {
		t.Errorf("came back with %+v lying about, wanted %+v", back.drops, g.drops)
	}
}

func TestASaveFromBeforeModesIsCreative(t *testing.T) {
	path := filepath.Join(t.TempDir(), "world")
	w := floor()
	if err := w.Save(path, world.Meta{Pos: geometry.Vector3{X: 20.5, Y: 6, Z: 20.5}, Slot: 2}); err != nil {
		t.Fatal(err)
	}

	g := savedGame(t, path)

	if g.survival || g.items != creativeStart() || g.health != maxHealth {
		t.Errorf("an old save opened in survival %v with %+v and health %v", g.survival, g.items[:3], g.health)
	}
}

func TestASurvivalPlayerSavedWithNoHealthWakes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "world")
	w := floor()
	if err := w.Save(path, world.Meta{Pos: geometry.Vector3{X: 30.5, Y: 6, Z: 30.5}, Mode: world.Survival, Food: 3}); err != nil {
		t.Fatal(err)
	}

	g := savedGame(t, path)

	if g.health != maxHealth || g.food != maxFood || g.player.Pos == (geometry.Vector3{X: 30.5, Y: 6, Z: 30.5}) {
		t.Errorf("health %v, food %v at %v: wanted the player woken at the spawn", g.health, g.food, g.player.Pos)
	}
}

func TestTheBodyIsShownInSurvivalOnly(t *testing.T) {
	g := inSurvival(t)
	g.items[0] = world.Stack{Item: world.Dirt.Item(), Count: 12}
	text, _ := drawn(t, g)
	if !contains(text, "♥", "●", "12") || contains(text, "○") {
		t.Errorf("survival does not show health, food and a count, or shows air while breathing:\n%s", text)
	}
	g.air = 3
	if text, _ = drawn(t, g); !contains(text, "○") {
		t.Errorf("short of breath, the air is not shown:\n%s", text)
	}

	g = onFloor(t)
	if text, _ = drawn(t, g); contains(text, "♥") || contains(text, "●") {
		t.Errorf("creative shows the body:\n%s", text)
	}
}

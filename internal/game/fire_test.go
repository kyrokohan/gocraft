package game

import (
	"math"
	"path/filepath"
	"testing"

	"glonk.dev/engine"
	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/world"
)

// firstMidnight is the days of play to the middle of the first night, which is quiet.
const firstMidnight = 0.7

// atHearth is a game in survival whose player looks down at a cold hearth at 20, 5, 21,
// with the item given in hand and the fuel given in the hearth.
func atHearth(t *testing.T, held world.Item, fuel world.Stack) (*Game, world.Cell) {
	t.Helper()
	g := digging(t, world.Hearth)
	at := world.Cell{X: 20, Y: 5, Z: 21}
	g.items[0] = world.Stack{Item: held, Count: 1}
	g.hearth(at).Fuel = fuel
	return g, at
}

func TestAHearthBurnsWhatItIsFedAndGoesOut(t *testing.T) {
	k := &world.HearthStore{Fuel: world.Stack{Item: world.Stick, Count: 2}}
	stepHearth(k, 60)
	if k.Burn != 0 || k.Fuel.Count != 2 {
		t.Fatalf("a hearth nobody lit burns: fire %v, fuel %+v", k.Burn, k.Fuel)
	}

	if !kindle(k) || k.Burn != burnTime(world.Stick) || k.Fuel.Count != 1 {
		t.Fatalf("lit: fire %v with fuel %+v, wanted one stick burning and one left", k.Burn, k.Fuel)
	}
	stepHearth(k, burnTime(world.Stick)+1)
	if k.Burn == 0 || k.Fuel.Count != 0 {
		t.Fatalf("after the first stick: fire %v, fuel %+v: wanted the second burning, with nothing to cook", k.Burn, k.Fuel)
	}
	stepHearth(k, burnTime(world.Stick))
	if k.Burn != 0 {
		t.Errorf("with the fuel gone the fire is at %v, wanted it out", k.Burn)
	}
	if kindle(k) {
		t.Error("a hearth with nothing to burn was lit")
	}
	if kindle(&world.HearthStore{Fuel: world.Stack{Item: world.Berries, Count: 3}}) {
		t.Error("a hearth was lit with berries")
	}
}

func TestWoodAndCoalBurnAndALogLongerThanAStick(t *testing.T) {
	for _, log := range logs {
		if burnTime(log) <= burnTime(world.Stick) || burnTime(log) >= burnTime(world.Coal) {
			t.Errorf("%v burns %v s, a stick %v and coal %v: wanted between them", log.Name(), burnTime(log), burnTime(world.Stick), burnTime(world.Coal))
		}
	}
	for _, item := range []world.Item{world.Berries, world.Stone.Item(), world.Meat, world.Resin, world.NoItem} {
		if burnTime(item) != 0 {
			t.Errorf("%v burns", item.Name())
		}
	}
}

func TestAHearthCooksWhileItBurns(t *testing.T) {
	for from, want := range map[world.Item]world.Item{
		world.Meat:        world.CookedMeat,
		world.PalmSeeds:   world.RoastSeeds,
		world.Spear:       world.HardSpear,
		world.SoftPot:     world.ClayPot.Item(),
		world.Clay.Item(): world.Brick.Item(),
	} {
		k := &world.HearthStore{
			In:   world.Stack{Item: from, Count: 1, Wear: 0},
			Fuel: world.Stack{Item: world.Coal, Count: 1},
		}
		if from == world.Spear {
			k.In.Wear = 40 // a spear that has seen use comes out whole
		}
		stepHearth(k, 2*cookTime)
		if k.Out.Count != 0 {
			t.Errorf("a cold hearth made %+v of %v", k.Out, from.Name())
		}
		kindle(k)
		stepHearth(k, cookTime-0.5)
		if k.Out.Count != 0 {
			t.Errorf("%v was cooked before its time", from.Name())
		}
		stepHearth(k, 1)
		if k.Out != (world.Stack{Item: want, Count: 1}) || k.In.Count != 0 {
			t.Errorf("of %v the hearth made %+v, wanted one %v", from.Name(), k.Out, want.Name())
		}
	}
	for _, old := range []world.Item{world.Sand.Item(), world.Berries, world.Moss} {
		if cooksFrom(old) {
			t.Errorf("a hearth still cooks %v", old.Name())
		}
	}
}

func TestAHearthThatGoesOutKeepsWhatWasHalfCooked(t *testing.T) {
	k := &world.HearthStore{
		In:   world.Stack{Item: world.Meat, Count: 3},
		Fuel: world.Stack{Item: world.Stick, Count: 1},
	}
	kindle(k)
	stepHearth(k, 60)
	whole := int(burnTime(world.Stick) / cookTime)
	if k.Out.Count != whole || k.In.Count != 3-whole || math.Abs(k.Cooked-(burnTime(world.Stick)-float64(whole)*cookTime)) > 0.3 {
		t.Errorf("a stick cooked %+v of %+v and left the next piece %v cooked, wanted %d and the rest of its time", k.Out, k.In, k.Cooked, whole)
	}
	if k.Burn != 0 {
		t.Errorf("the fire is at %v, wanted it out", k.Burn)
	}

	// With room for no more of what it makes, it waits and burns on.
	k = &world.HearthStore{
		In:   world.Stack{Item: world.Meat, Count: 1},
		Fuel: world.Stack{Item: world.Coal, Count: 1},
		Out:  world.Stack{Item: world.Brick.Item(), Count: 1},
	}
	kindle(k)
	stepHearth(k, 3*cookTime)
	if k.Out.Item != world.Brick.Item() || k.In.Count != 1 || k.Burn == 0 {
		t.Errorf("with brick where the meat goes: made %+v from %+v, fire %v", k.Out, k.In, k.Burn)
	}
}

func TestAFireDrillLightsAHearthThatHasFuelInSixSeconds(t *testing.T) {
	g, at := atHearth(t, world.FireDrill, world.Stack{Item: world.Stick, Count: 3})

	hold(t, g, seconds(drillTime-0.5), 'e')
	if g.hearth(at).Burn != 0 || g.screen != playing {
		t.Fatalf("after %v s of drilling: fire %v on screen %d, wanted it not lit yet and nothing opened", drillTime-0.5, g.hearth(at).Burn, g.screen)
	}
	hold(t, g, seconds(1), 'e')
	if k := g.hearth(at); k.Burn == 0 || k.Fuel.Count != 2 {
		t.Fatalf("after %v s of drilling: fire %v, fuel %+v, wanted it lit with one stick", drillTime+0.5, k.Burn, k.Fuel)
	}
	hold(t, g, 2)
	if b := g.world.At(at.X, at.Y, at.Z); b != world.HearthLit {
		t.Errorf("the hearth that was lit is %v, wanted one that burns", b)
	}
	if g.items[0].Wear != 1 {
		t.Errorf("the drill has %d uses on it, wanted 1", g.items[0].Wear)
	}
	if got := g.world.LightAt(at.X, at.Y+1, at.Z).Block(); got != 13 {
		t.Errorf("over the lit hearth the light is %d, wanted 13", got)
	}
}

func TestDrillingThatStopsStartsAgainFromNothing(t *testing.T) {
	g, at := atHearth(t, world.FireDrill, world.Stack{Item: world.Stick, Count: 3})
	hold(t, g, seconds(drillTime-1), 'e')
	hold(t, g, seconds(0.5))
	hold(t, g, seconds(drillTime-1), 'e')
	if g.hearth(at).Burn != 0 {
		t.Error("two spells of drilling with a rest between lit the hearth, wanted one unbroken spell")
	}
}

func TestAFireDrillDoesNotLightAHearthWithNoFuel(t *testing.T) {
	g, at := atHearth(t, world.FireDrill, world.Stack{})
	hold(t, g, 1, 'e')
	if g.screen != inBag || g.opened != world.Hearth {
		t.Fatalf("E with a drill on a hearth with nothing in it: screen %d, wanted the hearth opened to be fed", g.screen)
	}
	press(g, 'i')
	hold(t, g, seconds(drillTime+2), 'e')
	if g.hearth(at).Burn != 0 || g.world.At(at.X, at.Y, at.Z) != world.Hearth || g.items[0].Wear != 0 {
		t.Errorf("a hearth with no fuel was lit: fire %v, drill worn %d", g.hearth(at).Burn, g.items[0].Wear)
	}
}

func TestAHearthGivesLightOnlyWhileItBurns(t *testing.T) {
	g, at := atHearth(t, world.NoItem, world.Stack{})
	k := g.hearth(at)
	k.Fuel = world.Stack{Item: world.Stick, Count: 1}
	above := func() int { return g.world.LightAt(at.X, at.Y+1, at.Z).Block() }

	if hold(t, g, 2); above() != 0 {
		t.Fatalf("a cold hearth gives light %d", above())
	}
	kindle(k)
	if hold(t, g, 2); above() != 13 || g.world.At(at.X, at.Y, at.Z) != world.HearthLit {
		t.Fatalf("a burning hearth gives light %d and is %v, wanted 13 from one that burns", above(), g.world.At(at.X, at.Y, at.Z))
	}
	hold(t, g, seconds(burnTime(world.Stick)+1))
	if above() != 0 || g.world.At(at.X, at.Y, at.Z) != world.Hearth {
		t.Errorf("with its fuel gone the hearth gives light %d and is %v, wanted a cold one", above(), g.world.At(at.X, at.Y, at.Z))
	}
}

func TestAHearthThatBurnsOpensAndSpillsAsOne(t *testing.T) {
	g, at := atHearth(t, world.NoItem, world.Stack{Item: world.Coal, Count: 2})
	kindle(g.hearth(at))
	hold(t, g, 2)
	hold(t, g, 1, 'e')
	if g.screen != inBag || g.opened != world.Hearth || g.at != at {
		t.Fatalf("E on a hearth that burns: screen %d, opened %v", g.screen, g.opened)
	}
	text, _ := drawn(t, g)
	if !contains(text, "hearth", "burning") {
		t.Errorf("the open hearth does not say it burns:\n%s", text)
	}
	press(g, 'i')

	// Broken, it leaves a hearth and what it held, and is forgotten.
	g.items[0] = world.Stack{Item: world.FlintPick, Count: 1}
	hold(t, g, seconds(4), 'q')
	if g.world.At(at.X, at.Y, at.Z) != world.Air || len(g.hearths) != 0 {
		t.Fatalf("the hearth was not broken: %v there, %d hearths known", g.world.At(at.X, at.Y, at.Z), len(g.hearths))
	}
	left := map[world.Item]int{}
	for _, d := range g.drops {
		left[d.Stack.Item] += d.Stack.Count
	}
	for _, s := range g.items {
		left[s.Item] += s.Count
	}
	if left[world.Hearth.Item()] != 1 || left[world.Coal] != 1 || left[world.HearthLit.Item()] != 0 {
		t.Errorf("a broken hearth left %v, wanted a cold hearth and its coal", left)
	}
}

func TestATorchIsLitAtAFireAndNotWithoutOne(t *testing.T) {
	for _, c := range []struct {
		at   world.Block
		lit  bool
		name string
	}{
		{world.Fire, true, "a fire"},
		{world.HearthLit, true, "a hearth that burns"},
		{world.Torch, true, "a torch set down"},
		{world.Hearth, false, "a cold hearth"},
		{world.Stone, false, "a stone"},
		{world.CharredLog, false, "a charred log"},
	} {
		g := inSurvival(t)
		g.world.Set(20, 6, 22, world.Stone)
		g.world.Set(20, 7, 22, c.at) // in front of the player's eyes
		g.items[0] = world.Stack{Item: world.UnlitTorch, Count: 3}
		g.player.Pitch = 0.05
		hold(t, g, 1)
		if !g.aimed || g.target.Block != c.at {
			t.Fatalf("%s: aimed %v at %+v", c.name, g.aimed, g.target)
		}
		hold(t, g, 1, 'e')
		press(g, 'i') // a cold hearth opens: shut it again
		want := world.Stack{Item: world.UnlitTorch, Count: 3}
		if c.lit {
			want.Item = world.Torch.Item()
		}
		if g.items[0] != want {
			t.Errorf("E with unlit torches on %s left %+v in hand, wanted %+v", c.name, g.items[0], want)
		}
	}
}

func TestALitTorchLightsAColdHearthThatHasFuel(t *testing.T) {
	g, at := atHearth(t, world.Torch.Item(), world.Stack{Item: world.Log.Item(), Count: 1})
	hold(t, g, 1, 'e')
	hold(t, g, 2)
	if g.hearth(at).Burn == 0 || g.world.At(at.X, at.Y, at.Z) != world.HearthLit || g.screen != playing {
		t.Errorf("a torch on a hearth with wood: fire %v, block %v, screen %d", g.hearth(at).Burn, g.world.At(at.X, at.Y, at.Z), g.screen)
	}
	if g.items[0] != (world.Stack{Item: world.Torch.Item(), Count: 1}) {
		t.Errorf("the torch that lit it is now %+v", g.items[0])
	}

	g, at = atHearth(t, world.Torch.Item(), world.Stack{})
	hold(t, g, 1, 'e')
	if g.hearth(at).Burn != 0 || g.screen != inBag {
		t.Errorf("a torch on a hearth with no fuel: fire %v, screen %d, wanted it opened to be fed", g.hearth(at).Burn, g.screen)
	}
}

func TestATorchInTheHandLightsThePlayersPlaceAndGoesWithThePlayer(t *testing.T) {
	g := atNight(t)
	here := func() int {
		h := g.player.head()
		return g.world.LightAt(int(math.Floor(h.X)), int(math.Floor(h.Y)), int(math.Floor(h.Z))).Block()
	}
	hold(t, g, 2)
	if here() != 0 {
		t.Fatalf("with nothing in hand the light where the player stands is %d", here())
	}
	dark := g.sky.Brightness(g.lightAt(g.player.Pos))

	g.items[0] = world.Stack{Item: world.Torch.Item(), Count: 1}
	hold(t, g, 2)
	if here() != 14 {
		t.Fatalf("with a torch in hand the light where the player stands is %d, wanted 14", here())
	}
	if lit := g.sky.Brightness(g.lightAt(g.player.Pos)); lit <= hunterLit || dark >= hunterLit {
		t.Errorf("the ground is %.3f bright with a torch and %.3f without: wanted over and under what holds a hunter, %.3f", lit, dark, hunterLit)
	}

	from := g.player.Pos
	hold(t, g, seconds(2), 'w')
	if g.player.Pos.Distance(from) < 6 || here() != 14 {
		t.Errorf("%.1f blocks on, the light where the player stands is %d, wanted 14", g.player.Pos.Distance(from), here())
	}
	if back := g.world.LightAt(int(from.X), 7, int(from.Z)).Block(); back >= 14 || back != 14-int(math.Floor(g.player.Pos.Z))+int(math.Floor(from.Z)) {
		t.Errorf("where the player was the light is %d, wanted it %d cells dimmer", back, int(math.Floor(g.player.Pos.Z))-int(math.Floor(from.Z)))
	}

	g.selected = 1 // the torch is put away
	hold(t, g, 2)
	if here() != 0 {
		t.Errorf("with the torch put away the light where the player stands is %d", here())
	}

	g.selected, g.items[0] = 0, world.Stack{Item: world.UnlitTorch, Count: 1}
	hold(t, g, 2)
	if here() != 0 {
		t.Errorf("an unlit torch lights the place: %d", here())
	}
}

func TestATorchInTheHandBurnsOutAndOneSetDownDoesNot(t *testing.T) {
	g := atNight(t)
	g.world.Set(36, 6, 36, world.Torch) // too far off to light where the player stands
	g.items[0] = world.Stack{Item: world.Torch.Item(), Count: 2}

	// The time a torch has burned is counted as the game runs, and is not run through
	// here: five minutes of frames, three times over, is a long test.
	hold(t, g, seconds(10))
	if math.Abs(g.torch-10) > 0.1 {
		t.Fatalf("after 10 s in the hand the torch has burned %v s", g.torch)
	}
	g.torch = torchTime - 1.5
	hold(t, g, seconds(1))
	if g.items[0].Count != 2 {
		t.Fatalf("half a second short of its time the hand holds %+v, wanted both torches still", g.items[0])
	}
	hold(t, g, seconds(1))
	if g.items[0] != (world.Stack{Item: world.Torch.Item(), Count: 1}) || g.items.count([]world.Item{world.Stick}) != 1 {
		t.Fatalf("after %v s the hand holds %+v and %d sticks, wanted one torch and the stick of the other",
			torchTime, g.items[0], g.items.count([]world.Item{world.Stick}))
	}
	if g.notice == "" || g.torch > 1 {
		t.Errorf("the player was told %q, and the next torch has burned %v s: wanted word of it, and a fresh torch", g.notice, g.torch)
	}
	g.torch = torchTime - 0.5
	hold(t, g, seconds(1))
	if g.items[0].Item == world.Torch.Item() || g.items.count([]world.Item{world.Stick}) != 2 {
		t.Errorf("after the second torch's time the hand holds %+v", g.items[0])
	}
	h := g.player.head()
	if got := g.world.LightAt(int(math.Floor(h.X)), int(math.Floor(h.Y)), int(math.Floor(h.Z))).Block(); got != 0 {
		t.Errorf("with the last torch burned out the player's place is lit %d", got)
	}
	if g.world.At(36, 6, 36) != world.Torch {
		t.Error("the torch set down burned out")
	}

	// In creative nothing is used up.
	c := onFloor(t)
	c.items[0] = world.Stack{Item: world.Torch.Item(), Count: 1}
	c.torch = torchTime - 0.5
	hold(t, c, seconds(2))
	if c.items[0].Count != 1 {
		t.Errorf("in creative the torch in hand burned out: %+v", c.items[0])
	}
}

func TestHowFarATorchHasBurnedIsSaved(t *testing.T) {
	path := filepath.Join(t.TempDir(), "world")
	g := inSurvival(t)
	g.opts.SavePath = path
	g.items[0] = world.Stack{Item: world.Torch.Item(), Count: 1}
	hold(t, g, seconds(20))
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}
	if back := savedGame(t, path); math.Abs(back.torch-20) > 0.1 {
		t.Errorf("the torch came back %v s burned, wanted 20", back.torch)
	}
}

func TestAQuickclawKeepsOffAPlayerWhoCarriesATorch(t *testing.T) {
	g := atNight(t)
	g.items[0] = world.Stack{Item: world.Torch.Item(), Count: 1}
	g.fauna.all = []beast{claw(20.5, 32.5)}

	nearest := math.Inf(1)
	for range seconds(15) {
		hold(t, g, 1)
		nearest = min(nearest, g.fauna.all[0].Pos.Distance(g.player.Pos))
	}
	if nearest < 2.5 || nearest > 6 {
		t.Errorf("the quickclaw came to %.1f from a player with a torch, wanted it to wait at the edge of the light", nearest)
	}
	if g.health != maxHealth {
		t.Errorf("the player with a torch was bitten: health %v", g.health)
	}

	// Walking, the light goes along, and so the quickclaw keeps its distance.
	for range seconds(4) {
		hold(t, g, 1, 's') // backward, toward where it waits
	}
	if g.health != maxHealth {
		t.Errorf("the player who walked at it with a torch was bitten: health %v", g.health)
	}

	g.selected = 1 // the torch is put away
	hold(t, g, seconds(6))
	if g.health == maxHealth {
		t.Error("with the torch put away the quickclaw did not come for the player")
	}
}

func TestNoQuickclawComesOnTheFirstNightAndOneDoesOnTheSecond(t *testing.T) {
	g := inSurvival(t)
	g.days = firstMidnight
	g.fauna.all = nil
	hold(t, g, seconds(30))
	if n := len(only(g.fauna.all, render.Quickclaw)); n != 0 {
		t.Errorf("%d quickclaws came on the first night, wanted it quiet", n)
	}
	// One that is there all the same takes no notice of the player.
	g.fauna.all, g.fauna.timer = []beast{claw(20.5, 26.5)}, math.Inf(1)
	hold(t, g, seconds(8))
	if g.health != maxHealth {
		t.Errorf("on the first night the player was bitten: health %v", g.health)
	}

	g = inSurvival(t)
	g.days = firstMidnight + 1
	g.fauna.all = nil
	hold(t, g, seconds(30))
	if n := len(only(g.fauna.all, render.Quickclaw)); n == 0 {
		t.Error("no quickclaw came on the second night")
	}
	if midnight != firstMidnight+1 {
		t.Errorf("the night the other tests are held in is %v, wanted the second", midnight)
	}
}

func TestFireBurnsAPlayerWhoStandsInIt(t *testing.T) {
	g := inSurvival(t)
	g.world.Set(20, 6, 20, world.Fire)
	hold(t, g, seconds(2))
	if lost := maxHealth - g.health; lost < 2 || lost > 8 {
		t.Errorf("two seconds in a fire cost %v points of health, wanted a few", lost)
	}
	was := g.health
	g.world.Set(20, 6, 20, world.Air)
	g.world.Set(20, 6, 22, world.Fire) // two blocks off
	hold(t, g, seconds(2))
	if g.health < was {
		t.Errorf("a fire two blocks off cost %v points", was-g.health)
	}

	c := onFloor(t)
	c.world.Set(20, 6, 20, world.Fire)
	hold(t, c, seconds(2))
	if c.health != maxHealth {
		t.Errorf("in creative a fire cost %v points", maxHealth-c.health)
	}
}

func TestRawPalmSeedsArePoisonAndRoastOnesAreFood(t *testing.T) {
	g := inSurvival(t)
	g.food = 10
	g.items[0] = world.Stack{Item: world.PalmSeeds, Count: 2}
	if !g.eat() || g.health != maxHealth-world.PalmSeeds.Poison() || g.items[0].Count != 1 || g.notice == "" {
		t.Errorf("raw seeds eaten: health %v, %+v left, told %q", g.health, g.items[0], g.notice)
	}
	g.health = maxHealth
	g.items[0] = world.Stack{Item: world.RoastSeeds, Count: 1}
	if !g.eat() || g.health != maxHealth || g.food <= 10 {
		t.Errorf("roast seeds eaten: health %v, food %v", g.health, g.food)
	}
}

func TestSmokeRisesFromAHearthThatBurns(t *testing.T) {
	g, at := atHearth(t, world.NoItem, world.Stack{Item: world.Coal, Count: 1})
	hold(t, g, seconds(3))
	if len(g.bits) != 0 {
		t.Fatalf("a cold hearth gave off %d bits", len(g.bits))
	}
	kindle(g.hearth(at))
	hold(t, g, seconds(3))
	if len(g.bits) < 3 {
		t.Fatalf("a burning hearth gave off %d puffs of smoke in three seconds", len(g.bits))
	}
	highest := 0.0
	for _, b := range g.bits {
		if !b.smoke || b.pos.Y < float64(at.Y)+1 || math.Abs(b.pos.X-20.5) > 1.5 || math.Abs(b.pos.Z-21.5) > 1.5 {
			t.Fatalf("a bit at %+v (smoke %v) is not smoke over the hearth at %+v", b.pos, b.smoke, at)
		}
		highest = max(highest, b.pos.Y-float64(at.Y)-1)
	}
	if highest < 1 {
		t.Errorf("the smoke rose %.1f blocks, wanted it to rise", highest)
	}
	drawn(t, g) // and it can be drawn
}

func TestSmokeRisesFromATreeThatLightningStruck(t *testing.T) {
	g, err := New(Options{Seed: 7})
	if err != nil {
		t.Fatal(err)
	}
	g.Play()
	g.begin(false) // creative, so nothing hunts and nothing hurts
	fires := g.world.Struck(g.player.Pos.X, g.player.Pos.Z, 4000)
	if len(fires) == 0 {
		t.Fatal("no struck tree within 4,000 blocks of where seed 7 begins")
	}
	top := fires[0]
	g.player.Pos = geometry.Vector3{X: float64(top.X) + 6.5, Y: float64(top.Y) + 3, Z: float64(top.Z) + 0.5}
	g.player.Flying = true
	g.world.Follow(g.player.Pos.X, g.player.Pos.Z)
	g.world.LoadAll()
	if g.world.At(top.X, top.Y, top.Z) != world.Fire {
		t.Fatalf("the top of the struck tree at %+v is %v", top, g.world.At(top.X, top.Y, top.Z))
	}

	hush(g)
	hold(t, g, seconds(4))
	over := 0
	for _, b := range g.bits {
		if b.smoke && b.pos.Y > float64(top.Y)+1.5 && math.Abs(b.pos.X-float64(top.X)-0.5) < 2 {
			over++
		}
	}
	if over < 3 {
		t.Errorf("%d puffs of smoke stand over the struck tree after four seconds, of %d bits", over, len(g.bits))
	}

	// A fire that has been put out smokes no more. The fire of a camp near is put out too.
	for _, f := range append(fires, g.world.Camps(g.player.Pos.X, g.player.Pos.Z, smokeReach)...) {
		g.world.Set(f.X, f.Y, f.Z, world.Air)
	}
	hold(t, g, seconds(smokeLife+1))
	if len(g.bits) != 0 {
		t.Errorf("with the fires out there are still %d bits a while on", len(g.bits))
	}
}

func TestAPineLogThatHoldsResinLeavesItWithTheLog(t *testing.T) {
	g := inSurvival(t)
	x := 0
	for ; ; x++ {
		if _, n := world.PineLog.AlsoAt(x, 6, 3); n > 0 {
			break
		}
	}
	g.leave(x, 6, 3, world.PineLog)
	got := map[world.Item]int{}
	for _, d := range g.drops {
		got[d.Stack.Item] += d.Stack.Count
	}
	if got[world.PineLog.Item()] != 1 || got[world.Resin] != 1 || len(got) != 2 {
		t.Errorf("a pine log with resin left %v, wanted the log and the resin", got)
	}
}

func TestWhatIsMadeForFire(t *testing.T) {
	makes := func(item world.Item) (recipe, bool) {
		for _, r := range recipes {
			if r.makes == item {
				return r, true
			}
		}
		return recipe{}, false
	}
	for item, n := range map[world.Item]int{world.Hearth.Item(): 1, world.FireDrill: 1, world.UnlitTorch: 2, world.SoftPot: 1} {
		if r, ok := makes(item); !ok || r.at != byHand || r.n != n {
			t.Errorf("%v: recipe %+v, there %v: wanted %d made by hand", item.Name(), r, ok, n)
		}
	}
	for _, gone := range []world.Item{
		world.Torch.Item(), world.ClayPot.Item(), world.HearthLit.Item(), world.Glass.Item(),
		world.BerryLoaf, world.MossCake, world.CookedMeat, world.HardSpear,
	} {
		if _, ok := makes(gone); ok {
			t.Errorf("%v is made from a recipe, wanted it not", gone.Name())
		}
	}

	g := inSurvival(t)
	g.items[0] = world.Stack{Item: world.Cobble.Item(), Count: 8}
	g.items[1] = world.Stack{Item: world.Stick, Count: 2}
	g.items[2] = world.Stack{Item: world.RedwoodLog.Item(), Count: 1}
	g.items[3] = world.Stack{Item: world.Cord, Count: 1}
	g.items[4] = world.Stack{Item: world.Resin, Count: 1}
	for _, item := range []world.Item{world.Hearth.Item(), world.FireDrill, world.UnlitTorch} {
		r, _ := makes(item)
		if !g.craft(r) {
			t.Fatalf("%v could not be made of 8 cobble, 2 sticks, a log, a cord and resin", item.Name())
		}
	}
	for item, want := range map[world.Item]int{world.Hearth.Item(): 1, world.FireDrill: 1, world.UnlitTorch: 2, world.Cobble.Item(): 0, world.Stick: 0} {
		if n := g.items.count([]world.Item{item}); n != want {
			t.Errorf("the player carries %d of %v, wanted %d", n, item.Name(), want)
		}
	}
}

// A fire is the one place fire is found, so a hand that slips does not put it out.
func TestAFireIsNotBrokenOrBuiltOverAndGoesWithWhatItBurns(t *testing.T) {
	g := inSurvival(t)
	g.world.Set(20, 6, 22, world.CharredLog)
	g.world.Set(20, 7, 22, world.Fire)
	g.player.Pitch = 0.05
	g.items[0] = world.Stack{Item: world.Cobble.Item(), Count: 5}
	hold(t, g, 1)
	if !g.aimed || g.target.Block != world.Fire {
		t.Fatalf("aimed %v at %+v, wanted the fire", g.aimed, g.target)
	}

	hold(t, g, seconds(3), 'q')
	if g.world.At(20, 7, 22) != world.Fire {
		t.Fatal("a fire was broken by hand")
	}
	hold(t, g, 2, 'e')
	if g.world.At(20, 7, 22) != world.Fire {
		t.Fatalf("a block took the place of a fire: %v there", g.world.At(20, 7, 22))
	}

	g.world.Set(20, 7, 21, world.Air) // the block went against the fire, between it and the player
	g.items[0] = world.Stack{}
	g.player.Pitch = -0.5 // down, at the log it burns on
	hold(t, g, 1)
	if !g.aimed || g.target.Block != world.CharredLog {
		t.Fatalf("aimed %v at %+v, wanted the charred log", g.aimed, g.target)
	}
	hold(t, g, seconds(4), 'q')
	if g.world.At(20, 6, 22) != world.Air || g.world.At(20, 7, 22) != world.Air {
		t.Errorf("with its log broken there is %v and %v, wanted the fire gone with it", g.world.At(20, 6, 22), g.world.At(20, 7, 22))
	}
	for _, d := range g.drops {
		if d.Stack.Item != world.Coal {
			t.Errorf("the log and its fire left %+v, wanted only coal", d.Stack)
		}
	}
}

// What a frame costs while the player walks, with nothing in hand and with a lit torch,
// whose light is worked out again in every cell the player comes into.
func BenchmarkWalk(b *testing.B) {
	for _, c := range []struct {
		name string
		held world.Item
	}{{"empty hand", world.NoItem}, {"lit torch", world.Torch.Item()}} {
		b.Run(c.name, func(b *testing.B) {
			g, err := New(Options{Seed: 1})
			if err != nil {
				b.Fatal(err)
			}
			g.Play()
			g.days = firstMidnight
			g.items[0] = world.Stack{Item: c.held, Count: 1}
			fb, err := engine.NewFrameBuffer(240, 67)
			if err != nil {
				b.Fatal(err)
			}
			kb := engine.NewKeyboard()
			kb.Press('w', ' ')
			from := g.player.Pos
			for b.Loop() {
				if err := g.Update(1.0/60, kb.Input()); err != nil {
					b.Fatal(err)
				}
				kb.EndFrame()
				fb.Clear()
				g.Draw(fb)
			}
			b.ReportMetric(g.player.Pos.Distance(from)/float64(b.N)*60, "blocks/s")
		})
	}
}

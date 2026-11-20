package game

import (
	"math"
	"path/filepath"
	"testing"

	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/world"
)

// raised is a beast of a kind that is the player's, at x, z on the plain: young or grown.
func raised(kind render.Kind, x, z float64, young bool) beast {
	c := giant(kind, x, z)
	c.tame, c.young = true, young
	return c
}

// byFire is a game in survival on the plain by day whose player stands at z = 0.5 with
// a fire that burns at 3, 3, and the beasts given.
func byFire(t *testing.T, beasts ...beast) *Game {
	t.Helper()
	g := among(t, 0.5, beasts...)
	g.world.Set(3, 6, 3, world.Fire)
	return g
}

func TestAnEggIsPutDownAndHatchesAfterADayByAFire(t *testing.T) {
	g := byFire(t)
	g.items[0], g.selected = world.Stack{Item: world.BuglerEgg, Count: 1}, 0
	g.player.Pitch = -0.6 // down at the ground, two blocks ahead
	hold(t, g, 2, 'e')
	if len(g.fauna.eggs) != 1 || carried(g, world.BuglerEgg) != 0 {
		t.Fatalf("E with an egg in hand at the ground left %d eggs there and %d with the player", len(g.fauna.eggs), carried(g, world.BuglerEgg))
	}
	e := &g.fauna.eggs[0]
	if e.wild || e.herd != 0 || e.kind != render.Bugler || e.Pos.Y != 6 || math.Hypot(e.Pos.X-0.5, e.Pos.Z-2.5) > 1.5 {
		t.Fatalf("the egg put down is %+v: wanted a bugler's, nobody's but the player's, on the ground ahead", *e)
	}
	// By the fire it gathers warmth, a second each second.
	hold(t, g, seconds(4))
	if e = &g.fauna.eggs[0]; e.warm < 2.5 || e.warm > 4.1 {
		t.Errorf("four seconds within six blocks of a fire gave an egg %.1f s of warmth", e.warm)
	}
	// A day of it, and no less, hatches it.
	e.warm = hatchTime - 3
	hold(t, g, seconds(1))
	if len(g.fauna.eggs) != 1 || len(g.fauna.all) != 0 {
		t.Fatalf("an egg hatched %.0f s short of a day of warmth", hatchTime-g.fauna.eggs[0].warm)
	}
	at := g.fauna.eggs[0].Pos
	hold(t, g, seconds(4))
	if len(g.fauna.eggs) != 0 || len(g.fauna.all) != 1 {
		t.Fatalf("a day of warmth left %d eggs and %d beasts: wanted the egg hatched", len(g.fauna.eggs), len(g.fauna.all))
	}
	if c := g.fauna.all[0]; c.kind != render.Bugler || !c.young || !c.tame || c.herd != 0 || c.dead || c.health <= 0 || c.Pos.Distance(at) > 2.5 {
		t.Errorf("what hatched is a %s at %+v, young: %v, tame: %v, with %v health: wanted a young bugler that is the player's, where the egg lay", c.kind, c.Pos, c.young, c.tame, c.health)
	}

	// Away from a fire an egg keeps what it has, and gathers none. A wild one gathers
	// none by a fire.
	g = byFire(t)
	g.fauna.eggs = []egg{
		{kind: render.Galewing, Pos: geometry.Vector3{X: 3.5, Y: 6, Z: 30.5}, warm: 100},
		{kind: render.Bugler, Pos: geometry.Vector3{X: 4.5, Y: 6, Z: 5.5}, herd: 3, wild: true},
		{kind: render.Shieldbrow, Pos: geometry.Vector3{X: 2.5, Y: 6, Z: 5.5}, warm: 100},
	}
	hold(t, g, seconds(5))
	if far, wild, near := g.fauna.eggs[0].warm, g.fauna.eggs[1].warm, g.fauna.eggs[2].warm; far != 100 || wild != 0 || near < 103 {
		t.Errorf("after five seconds an egg 27 blocks from the fire has %.1f s of warmth, a wild one by it %.1f and the player's by it %.1f: wanted 100, 0 and 105", far, wild, near)
	}
	g.world.Set(3, 6, 3, world.Air) // the fire is out
	hold(t, g, seconds(3))
	was := g.fauna.eggs[2].warm
	hold(t, g, seconds(5))
	if now := g.fauna.eggs[2].warm; now != was || now < 100 {
		t.Errorf("with the fire out an egg went from %.1f s of warmth to %.1f: wanted it to keep what it had", was, now)
	}
	drawn(t, g)
}

func TestAHatchlingFollowsAndGrowsInTwoFedDays(t *testing.T) {
	g := byFire(t, raised(render.Bugler, 0.5, 3.5, true))
	c := &g.fauna.all[0]
	full := breeds[render.Bugler].health
	// It follows.
	stand(g, geometry.Vector3{X: 0.5, Y: 6, Z: 30.5})
	hold(t, g, seconds(12))
	if far := c.Pos.Distance(g.player.Pos); far > followFar+3 {
		t.Errorf("twelve seconds after the player went 27 blocks off, its hatchling is %.1f blocks from it", far)
	}
	// Unfed, it does not grow.
	hold(t, g, seconds(5))
	if c.age != 0 || !c.young {
		t.Fatalf("an unfed hatchling grew %.1f s in five", c.age)
	}
	// A meal from the hand is a day of growing, and it takes no other while that lasts.
	meal := func(food world.Item) bool {
		g.items[0], g.selected = world.Stack{Item: food, Count: 2}, 0
		return g.feed(prey{i: 0})
	}
	if !meal(world.Berries) || c.fedFor != dayLength || g.items[0].Count != 1 {
		t.Fatalf("a hatchling given berries has %.0f s of growing before it, and the player has %d berries left: wanted a day, and one", c.fedFor, g.items[0].Count)
	}
	if meal(world.Berries) || g.items[0].Count != 2 {
		t.Error("a hatchling that has eaten today took a second meal")
	}
	hold(t, g, seconds(4))
	if math.Abs(c.age-4) > 0.2 || math.Abs(c.fedFor-(dayLength-4)) > 0.2 {
		t.Errorf("four seconds after its meal a hatchling has grown %.1f s and has %.1f s of growing before it", c.age, c.fedFor)
	}
	// When the meal has run out it stops, a day older, and is still young.
	c.age, c.fedFor = dayLength-2, 2
	hold(t, g, seconds(5))
	if math.Abs(c.age-dayLength) > 1e-6 || c.fedFor != 0 || !c.young {
		t.Fatalf("a hatchling whose meal ran out has grown %.1f s, with %.1f s before it, young: %v: wanted one day, and no more", c.age, c.fedFor, c.young)
	}
	// A second day, fed, and it is grown, with all a grown one's health.
	if !meal(world.Fruit) {
		t.Fatal("a hatchling whose meal had run out took no fruit")
	}
	c.age, c.fedFor = growTime-2, 2
	hold(t, g, seconds(4))
	if c.young || !c.tame || c.health != full || c.dead {
		t.Fatalf("a hatchling that has grown two fed days is young: %v, with %v health of %v", c.young, c.health, full)
	}
	if meal(world.Berries) {
		t.Error("a grown beast that is the player's took a meal")
	}

	// What each eats.
	for kind, foods := range map[render.Kind][]world.Item{
		render.Bugler:     {world.Berries, world.Fruit, world.Fiber},
		render.Shieldbrow: {world.Berries, world.Fruit, world.Fiber},
		render.Galewing:   {world.Meat},
	} {
		for _, food := range []world.Item{world.Berries, world.Fruit, world.Fiber, world.Meat, world.Stick} {
			young := raised(kind, 0.5, 3.5, true)
			ate, _ := young.feed(food)
			if want := containsItem(foods, food); ate != want {
				t.Errorf("a young %s given %v ate: %v, wanted %v", kind, food, ate, want)
			}
		}
	}
	// A young galewing walks after the player: it does not fly until it is grown.
	g = byFire(t, raised(render.Galewing, 0.5, 3.5, true))
	stand(g, geometry.Vector3{X: 0.5, Y: 6, Z: 20.5})
	hold(t, g, seconds(10))
	if c := g.fauna.all[0]; c.aloft || c.Pos.Y > 6.6 || c.Pos.Distance(g.player.Pos) > followFar+4 {
		t.Errorf("a young galewing that is the player's is at %+v, in the air: %v: wanted it on foot, by the player", c.Pos, c.aloft)
	}
}

func containsItem(all []world.Item, one world.Item) bool {
	for _, a := range all {
		if a == one {
			return true
		}
	}
	return false
}

func TestATameBeastThatLosesAllItsHealthLimpsHomeAndHeals(t *testing.T) {
	g := among(t, 0.5, raised(render.Bugler, 0.5, 30.5, false), giant(render.Quickclaw, 20.5, 30.5))
	g.world = wide()
	g.bed = &world.Cell{X: 0, Y: 6, Z: -40}
	g.days = 0.2
	c := &g.fauna.all[0]
	b := breeds[render.Bugler]
	if c.hit(g.player.Pos, 1000) || c.dead || !c.limping || c.health <= 0 {
		t.Fatalf("a beast that is the player's, hit for all its health, is dead: %v, limping: %v, with %v health", c.dead, c.limping, c.health)
	}
	left := c.health
	if c.hit(g.player.Pos, 1000) || c.health != left || c.dead {
		t.Errorf("a beast that limps home was hurt again: %v health, and it had %v", c.health, left)
	}
	if p := g.fauna.prey(&g.fauna.all[1]); p != nil {
		t.Errorf("a quickclaw goes for a %s that limps home", p.kind)
	}
	g.fauna.all = g.fauna.all[:1]
	c = &g.fauna.all[0]

	// It goes to the bed, at half a walk, and stays there.
	hold(t, g, seconds(1))
	from := c.Pos
	hold(t, g, seconds(10))
	went, want := from.Z-c.Pos.Z, 10*b.walk/2
	if math.Abs(went-want) > 0.35*want || math.Abs(c.Pos.X-from.X) > 2 || c.hurry || c.flee > 0 {
		t.Errorf("in ten seconds a beast that limps home went %.1f blocks toward the bed: wanted %.1f, half a walk", went, want)
	}
	c.Pos = geometry.Vector3{X: 2.5, Y: 6, Z: -38.5}
	hold(t, g, seconds(6))
	if far := math.Hypot(c.Pos.X-0.5, c.Pos.Z+39.5); far > 4 || !c.limping {
		t.Errorf("a beast that limped home is %.1f blocks from the bed, limping: %v", far, c.limping)
	}
	// A day on it is whole.
	g.days += 0.9
	hold(t, g, seconds(1))
	if !c.limping || c.health == b.health {
		t.Fatalf("nine tenths of a day after it fell a beast is whole")
	}
	g.days += 0.11
	hold(t, g, seconds(1))
	if c.limping || c.health != b.health || c.dead {
		t.Errorf("a day after it fell a beast limps: %v, with %v health of %v", c.limping, c.health, b.health)
	}

	// With no bed it goes to where a new player starts. A wild beast dies as it did.
	g = among(t, 0.5, raised(render.Bugler, 30.5, 30.5, false), giant(render.Bugler, 3.5, 3.5))
	g.fauna.all[0].hit(g.player.Pos, 1000)
	if !g.fauna.all[1].hit(g.player.Pos, 1000) || !g.fauna.all[1].dead {
		t.Error("a wild bugler hit for all its health is not dead")
	}
	hold(t, g, seconds(2))
	home := g.world.Spawn()
	if c := g.fauna.all[0]; facing(c, home) > 0.3 || !c.walking {
		t.Errorf("a beast that limps home with no bed faces %.2f radians off the way to where a player starts, walking: %v", facing(c, home), c.walking)
	}
}

func TestATameGuardDoesNotTurnOnItsPlayer(t *testing.T) {
	g := among(t, 0.5, raised(render.Shieldbrow, 0.5, 4.5, false))
	c := &g.fauna.all[0]
	for range seconds(10) {
		hold(t, g, 1)
		if c.lower || c.ire > 0 || g.health != maxHealth {
			t.Fatalf("a shieldbrow that is the player's turned on it: head down %v, charging %v, %v points lost", c.lower, c.ire > 0, maxHealth-g.health)
		}
	}
	// A wild one four blocks off still does.
	g = among(t, 0.5, giant(render.Shieldbrow, 0.5, 4.5))
	hold(t, g, seconds(3))
	if c := g.fauna.all[0]; !c.lower && c.ire <= 0 {
		t.Error("a wild shieldbrow four blocks from a player does not turn on it")
	}
}

func TestASaveHoldsABeastThatIsRaised(t *testing.T) {
	path := filepath.Join(t.TempDir(), "world")
	g := savedGame(t, path)
	startNew(t, g, true)
	young := raised(render.Shieldbrow, g.player.Pos.X+2, g.player.Pos.Z, true)
	young.Pos.Y, young.age, young.fedFor = g.player.Pos.Y, 700, 120
	hurt := raised(render.Bugler, g.player.Pos.X-2, g.player.Pos.Z, false)
	hurt.Pos.Y, hurt.limping, hurt.mendAt, hurt.health = g.player.Pos.Y, true, 3.25, 1
	g.fauna.all = []beast{young, hurt}
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}
	back := savedGame(t, path)
	back.Play()
	if len(back.fauna.all) != 2 {
		t.Fatalf("a save of two beasts opened with %d", len(back.fauna.all))
	}
	if c := back.fauna.all[0]; !c.tame || !c.young || c.age != 700 || c.fedFor != 120 || c.limping {
		t.Errorf("a young beast grown 700 s with 120 s before it came back tame: %v, young: %v, grown %.0f s with %.0f before it", c.tame, c.young, c.age, c.fedFor)
	}
	if c := back.fauna.all[1]; !c.tame || c.young || !c.limping || c.mendAt != 3.25 || c.health != 1 {
		t.Errorf("a beast that limps home came back tame: %v, limping: %v, whole on day %.2f, with %v health", c.tame, c.limping, c.mendAt, c.health)
	}
}

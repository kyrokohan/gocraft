package game

import (
	"math"
	"testing"

	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/sound"
	"github.com/kyrokohan/skyspear/internal/world"
)

// pit digs a pit in a plain: across x from x0 to x1, along z from 10 to 12, so many
// blocks deep, with ground under it and round it, and a cover over all of it if asked.
func pit(w *world.World, x0, x1, deep int, covered bool) {
	for x := -12; x <= 12; x++ {
		for z := 2; z <= 20; z++ {
			for y := range 5 {
				w.Set(x, y, z, world.Stone)
			}
		}
	}
	for x := x0; x <= x1; x++ {
		for z := 10; z <= 12; z++ {
			for y := 5; y > 5-deep; y-- {
				w.Set(x, y, z, world.Air)
			}
			if covered {
				w.Set(x, 5, z, world.PitCover)
			}
		}
	}
}

// covers counts the pit covers left over the pit.
func covers(w *world.World) (n int) {
	for x := -12; x <= 12; x++ {
		for z := 2; z <= 20; z++ {
			if w.At(x, 5, z) == world.PitCover {
				n++
			}
		}
	}
	return n
}

// crosses walks a beast of a kind straight at a covered pit three blocks wide and four
// deep for a while, in a game, and returns the game.
func crosses(t *testing.T, kind render.Kind) *Game {
	t.Helper()
	away := -30.5 // the player, too far off to be anything to the beast
	if breeds[kind].heavy {
		away = -8.5 // or near enough to hear the cover go, and still no threat
	}
	g := among(t, away, giant(kind, 0.5, 5.5))
	g.days = midnight // so that a quickclaw does not go with the day
	pit(g.world, -1, 1, 4, true)
	for range seconds(14) {
		if len(g.fauna.all) != 1 {
			t.Fatalf("the %s is gone", kind)
		}
		if c := &g.fauna.all[0]; c.Pos.Y > 5.5 && c.Pos.Z < 16 {
			c.walking, c.timer, c.Yaw = true, math.Inf(1), 0 // it keeps on, straight at the pit, while it is up
		}
		hold(t, g, 1)
	}
	return g
}

func TestAPitCoverHoldsThePlayerAndALightBeastAndBreaksUnderAHeavyOne(t *testing.T) {
	// The player walks over it.
	g := among(t, 5.5)
	pit(g.world, -1, 1, 4, true)
	hold(t, g, seconds(4), 'w')
	if g.player.Pos.Z < 14 || !near(g.player.Pos.Y, 6) || covers(g.world) != 9 {
		t.Errorf("the player walked to %+v with %d covers left, wanted across the pit on all 9", g.player.Pos, covers(g.world))
	}

	for _, kind := range []render.Kind{render.Quickclaw, render.Mossback, render.Nipper} {
		g := crosses(t, kind)
		if c := g.fauna.all[0]; c.Pos.Y < 5.9 || covers(g.world) != 9 || c.Pos.Z < 13 {
			t.Errorf("a %s at a covered pit is at %+v with %d covers left, wanted it across on all 9", kind, c.Pos, covers(g.world))
		}
	}
	for _, kind := range []render.Kind{render.Bugler, render.Shieldbrow, render.Thornback, render.Boulderback, render.Dread} {
		g := crosses(t, kind)
		c := g.fauna.all[0]
		if covers(g.world) != 0 {
			t.Errorf("a %s left %d of the 9 covers, wanted the whole cover to give way", kind, covers(g.world))
		}
		if !near(c.Pos.Y, 2) || c.Pos.Z < 9.5 || c.Pos.Z > 13.5 {
			t.Errorf("a %s that broke through is at %+v, wanted it on the floor of the pit, four blocks down", kind, c.Pos)
		}
		if heard(g)[sound.Break(sound.Wood)] == 0 || !breeds[kind].heavy {
			t.Errorf("a %s broke through: breaks heard %d, heavy %v", kind, heard(g)[sound.Break(sound.Wood)], breeds[kind].heavy)
		}
	}
}

func TestAFallHurtsAHeavyBeastByItsSizeAndItsDepth(t *testing.T) {
	falls := func(kind render.Kind, deep int) (lost, part float64) {
		w := plain()
		pit(w, -1, 1, deep, false)
		c := giant(kind, 0.5, 11.5)
		c.Pos.Y = 6 // over the open pit
		for range seconds(4) {
			c.Step(w, frame)
		}
		if !near(c.Pos.Y, float64(6-deep)) {
			t.Fatalf("a %s dropped into a pit %d deep is at height %.2f", kind, deep, c.Pos.Y)
		}
		return breeds[kind].health - c.health, (breeds[kind].health - c.health) / breeds[kind].health
	}
	if lost, part := falls(render.Dread, 4); part < 0.28 || part > 0.38 {
		t.Errorf("a dread that fell four blocks lost %.0f health, %.2f of it: wanted about a third", lost, part)
	}
	if _, shallow := falls(render.Dread, 2); shallow != 0 {
		t.Errorf("a dread that fell two blocks lost %.2f of its health, wanted none", shallow)
	}
	_, four := falls(render.Bugler, 4)
	_, six := falls(render.Bugler, 6)
	if lost, _ := falls(render.Bugler, 4); !(six > four && four > 0) || lost >= breeds[render.Dread].health/6 {
		t.Errorf("a bugler loses %.2f of its health to four blocks and %.2f to six: wanted more the deeper, and fewer points than a dread", four, six)
	}
	if lost, _ := falls(render.Quickclaw, 6); lost != 0 {
		t.Errorf("a quickclaw, which is light, lost %v health to a fall", lost)
	}
	// Far enough, the fall kills.
	w := plain()
	pit(w, -1, 1, 4, false)
	c := giant(render.Bugler, 0.5, 11.5)
	c.Pos.Y = 11 // five blocks over the open pit: nine to fall
	for range seconds(4) {
		c.Step(w, frame)
	}
	if !c.dead {
		t.Errorf("a bugler that fell nine blocks lives, with %v health", c.health)
	}
}

func TestADreadInAPitCannotLeaveOrBiteAndIsKilledFromTheRim(t *testing.T) {
	b := breeds[render.Dread]
	g := hunting(t, 16.5, dread(0.5, 4.5)) // the player beyond the pit: the dread comes over it
	pit(g.world, -1, 1, 4, true)
	hold(t, g, seconds(10))
	c := g.fauna.all[0]
	if !near(c.Pos.Y, 2) || covers(g.world) != 0 {
		t.Fatalf("the dread that came for the player is at %+v with %d covers left, wanted it in the pit", c.Pos, covers(g.world))
	}
	if part := (b.health - c.health) / b.health; part < 0.28 || part > 0.38 {
		t.Errorf("the fall cost the dread %.2f of its health, wanted about a third", part)
	}

	// The player comes to the rim. The dread cannot climb the four blocks, or reach.
	g.player.Pos = geometry.Vector3{X: 0.5, Y: 6, Z: 13.6}
	was := g.health
	hold(t, g, seconds(20))
	if c := g.fauna.all[0]; c.Pos.Y > 2.5 || g.health < was {
		t.Fatalf("twenty seconds on the dread is at height %.1f and the player has %v of %v health: wanted it held, and the player whole", c.Pos.Y, g.health, was)
	}

	// Spears thrown down from the rim kill it.
	g.player.Yaw, g.player.Pitch = math.Pi, -0.9 // back at the pit, and down
	holding(g, world.HardSpear, 1)
	if hold(t, g, 1); !g.preyed {
		t.Error("from the rim, a spear in hand does not reach the dread in the pit to stab it")
	}
	thrown := 0
	for ; thrown < 60 && len(only(g.fauna.all, render.Dread)) == 1 && !g.fauna.all[0].dead; thrown++ {
		holding(g, world.HardSpear, 1)
		g.hitWait = 0
		before := g.fauna.all[0].health
		hold(t, g, 1, 'r')
		flies(t, g)
		if g.fauna.all[0].health >= before && thrown < 3 {
			t.Fatalf("spear %d thrown down from the rim missed the dread in the pit: %+v", thrown+1, g.missiles)
		}
	}
	if thrown >= 60 {
		t.Fatalf("sixty spears from the rim did not kill the dread: %v health left", g.fauna.all[0].health)
	}
	if thrown > 26 {
		t.Errorf("it took %d spears from the rim, wanted the fall to have done a third of the work", thrown)
	}
	hold(t, g, seconds(fallTime+0.5))
	meat := 0
	for _, d := range g.drops {
		if d.Stack.Item == world.Meat {
			meat += d.Stack.Count
		}
	}
	if meat != b.meat {
		t.Errorf("the dead dread left %d pieces of meat, wanted %d", meat, b.meat)
	}
}

func TestAStampedeHurtsALighterBeastInItsWayAndNotAHeavyOne(t *testing.T) {
	g := among(t, -38.5)
	g.fauna = herdOf(6, 0.5, 20.5)
	g.days = midnight // quickclaws are about, and do not go
	stood := claw(0.5, 27.5)
	stood.herd = 9
	g.fauna.all = append(g.fauna.all, stood, giant(render.Shieldbrow, 6.5, 30.5))
	g.fauna.raise(1, geometry.Vector3{X: 0.5, Y: 6, Z: 10.5}) // from behind the herd: it runs at them
	hurt := false
	for range seconds(6) {
		hold(t, g, 1)
		for _, c := range g.fauna.all {
			hurt = hurt || c.kind == render.Quickclaw && (c.dead || c.health < breeds[render.Quickclaw].health)
		}
	}
	if !hurt && len(only(g.fauna.all, render.Quickclaw)) == 1 {
		t.Error("a herd ran over a quickclaw and did not hurt it")
	}
	for _, c := range only(g.fauna.all, render.Shieldbrow) {
		if c.health != breeds[render.Shieldbrow].health {
			t.Errorf("a shieldbrow in the way of a stampede lost %v health, wanted none: it is too heavy to run down", breeds[render.Shieldbrow].health-c.health)
		}
	}
}

func TestAPebbleDrivesAHerdIntoAPit(t *testing.T) {
	g := thrower(t, world.Sling, 0)
	g.items[1] = world.Stack{Item: world.Pebble, Count: 1}
	for x := -12; x <= 12; x++ { // ground under the pit, and the pit: wide, across the herd's way
		for z := 20; z <= 34; z++ {
			for y := range 5 {
				g.world.Set(x, y, z, world.Stone)
			}
		}
	}
	for x := -9; x <= 9; x++ {
		for z := 27; z <= 29; z++ {
			for y := 5; y > 2; y-- {
				g.world.Set(x, y, z, world.Air)
			}
			g.world.Set(x, 5, z, world.PitCover)
		}
	}
	g.fauna = herdOf(6, 0.5, 21.5)

	// A pebble to land short of the herd, so that it runs on, away from the player.
	pitch := 0.0
	for ; reachOf(fliers[pebbleSlung], pitch) < 13; pitch += 0.002 {
	}
	g.player.Pitch = pitch
	hold(t, g, 1, 'r')
	flies(t, g)
	if g.fauna.alarms[1].left <= 0 {
		t.Fatalf("the pebble that landed short of the herd did not alarm it")
	}
	hold(t, g, seconds(5))
	fell := 0
	for _, c := range g.fauna.all {
		if c.Pos.Y < 4.5 {
			fell++
		}
	}
	if fell == 0 {
		t.Errorf("a herd driven at a covered pit: none fell in")
	}
	for _, c := range g.fauna.all {
		if c.Pos.Y < 4.5 && c.Pos.Y > 3.5 {
			t.Errorf("a bugler is at height %.1f, wanted it on the floor of the pit at 3", c.Pos.Y)
		}
	}
}

func TestAPitCoverIsMadeByHandAndLooksLikeAnyBlockToThePlayer(t *testing.T) {
	g := inSurvival(t)
	g.items[0] = world.Stack{Item: world.Stick, Count: 2}
	g.items[1] = world.Stack{Item: world.Fiber, Count: 2}
	r, ok := recipeFor(g, world.PitCover.Item())
	if !ok || r.at != byHand || !g.craft(r) {
		t.Fatal("a pit cover could not be made by hand of 2 sticks and 2 fibers")
	}
	if n := g.items.count([]world.Item{world.PitCover.Item()}); n != 2 {
		t.Errorf("made %d covers, wanted 2", n)
	}
	if info := world.PitCover.Info(); !info.Solid || info.Stands() || info.Dig <= 0 || info.Dig > 1 {
		t.Errorf("a pit cover: solid %v, stands %v, dig %v: wanted a block a person stands on and breaks by hand", info.Solid, info.Stands(), info.Dig)
	}
	if materialOf(world.PitCover) != sound.Wood {
		t.Error("a pit cover does not sound of wood")
	}
}

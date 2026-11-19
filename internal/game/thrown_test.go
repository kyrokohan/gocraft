package game

import (
	"math"
	"path/filepath"
	"strings"
	"testing"

	"glonk.dev/engine"
	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/world"
)

// thrower is a game in survival on the plain, with no beasts and none to come, whose
// player stands at its origin, faces +z and looks up at an angle, with an item in hand.
func thrower(t *testing.T, with world.Item, pitch float64) *Game {
	t.Helper()
	g := inSurvival(t)
	g.world = plain()
	g.player.Pos = geometry.Vector3{X: 0.5, Y: 6, Z: 0.5}
	g.player.Pitch = pitch
	g.fauna.all, g.fauna.timer = nil, math.Inf(1)
	holding(g, with, 1)
	return g
}

// flies steps a game until what was thrown has landed, and returns how high it rose
// over the ground and where it came down. It fails the test if nothing does in ten
// seconds.
func flies(t *testing.T, g *Game) (peak float64, landed geometry.Vector3) {
	t.Helper()
	for range seconds(10) {
		for _, m := range g.missiles {
			peak = max(peak, m.pos.Y-6)
			landed = m.pos
		}
		if len(g.missiles) == 0 || g.missiles[0].landed {
			return peak, landed
		}
		hold(t, g, 1)
	}
	t.Fatalf("what was thrown is still in the air after ten seconds: %+v", g.missiles)
	return 0, geometry.Vector3{}
}

// reachOf is how far along the ground a thing thrown at an angle from the height of
// the eye comes down, by the arithmetic of an arc.
func reachOf(f flight, pitch float64) float64 {
	up, along := f.speed*math.Sin(pitch), f.speed*math.Cos(pitch)
	return along * (up + math.Sqrt(up*up+2*f.weight*eyeHeight)) / f.weight
}

func TestRThrowsTheSpearInHandAndItFliesInAnArc(t *testing.T) {
	const pitch = 0.5
	g := thrower(t, world.Spear, pitch)
	hold(t, g, 1, 'r')
	if len(g.missiles) != 1 || g.items[0].Count != 0 {
		t.Fatalf("R with a spear: %d things in the air and %+v in hand, wanted the spear thrown", len(g.missiles), g.items[0])
	}
	if m := g.missiles[0]; m.stack.Item != world.Spear || m.vel.Y <= 0 || m.vel.Z <= 0 || math.Abs(m.vel.X) > 0.01 {
		t.Fatalf("thrown: %+v, wanted a spear going up and ahead", m)
	}

	peak, landed := flies(t, g)
	f := fliers[spearThrown]
	if want := eyeHeight + math.Pow(f.speed*math.Sin(pitch), 2)/(2*f.weight); math.Abs(peak-want) > 0.6 {
		t.Errorf("the spear rose %.1f over the ground, wanted %.1f as an arc does", peak, want)
	}
	if want := reachOf(f, pitch); math.Abs(landed.Z-0.5-want) > 1.5 || math.Abs(landed.X-0.5) > 0.1 {
		t.Errorf("the spear came down %.1f ahead and %.1f aside, wanted %.1f ahead where the arc ends", landed.Z-0.5, landed.X-0.5, want)
	}
	if landed.Y < 6 || landed.Y > 6.7 {
		t.Errorf("the spear lies at height %.2f, wanted it on the ground at 6", landed.Y)
	}

	// Thrown level it comes down sooner, and thrown again and again it lands in the same place.
	level := thrower(t, world.Spear, 0)
	hold(t, level, 1, 'r')
	if _, at := flies(t, level); at.Z >= landed.Z || at.Z < 4 {
		t.Errorf("thrown level the spear came down %.1f ahead, and %.1f thrown up", at.Z-0.5, landed.Z-0.5)
	}
}

func TestAThrownSpearSticksWhereItLandsAndIsPickedUpAgain(t *testing.T) {
	g := thrower(t, world.Spear, 0.3)
	g.items[0].Wear = 7
	hold(t, g, 1, 'r')
	_, at := flies(t, g)

	hold(t, g, seconds(3))
	if len(g.missiles) != 1 || !g.missiles[0].landed || g.missiles[0].pos != at || len(g.drops) != 0 {
		t.Fatalf("three seconds on: %+v and %d drops, wanted the spear stuck where it landed", g.missiles, len(g.drops))
	}
	drawn(t, g) // it is drawn as it sticks

	g.player.Pos = geometry.Vector3{X: at.X, Y: 6, Z: at.Z - 1}
	hold(t, g, seconds(1))
	if len(g.missiles) != 0 || g.items.count([]world.Item{world.Spear}) != 1 {
		t.Fatalf("standing by it: %d still stuck and %d spears carried, wanted it picked up", len(g.missiles), g.items.count([]world.Item{world.Spear}))
	}
	for _, s := range g.items {
		if s.Item == world.Spear && s.Wear != 8 {
			t.Errorf("the spear came back with %d uses on it, wanted the 7 it had and one for the throw", s.Wear)
		}
	}
}

func TestAThrownSpearHurtsABeastMoreThanAStabAndLiesByIt(t *testing.T) {
	for _, spear := range []world.Item{world.Spear, world.HardSpear} {
		g := thrower(t, spear, 0)
		one := quarry(7)
		one.young, one.Pos.X, one.Pos.Z = false, 0.5, 7.5
		g.fauna.all = []beast{one}
		hold(t, g, 1, 'r')
		flies(t, g)

		lost := 100 - g.fauna.all[0].health
		if lost <= spear.Damage() || lost != spear.Damage()+thrownMore {
			t.Errorf("a thrown %s took %v points, and a stab takes %v: wanted %v more", spear.Name(), lost, spear.Damage(), thrownMore)
		}
		if g.fauna.all[0].hurt <= 0 || g.fauna.all[0].flee <= 0 {
			t.Errorf("the beast a %s hit does not show it or run: %+v", spear.Name(), g.fauna.all[0])
		}
		hold(t, g, seconds(1))
		if len(g.missiles) != 0 || len(g.drops) != 1 || g.drops[0].Stack.Item != spear || g.drops[0].Stack.Wear != 1 {
			t.Fatalf("after a %s hit a beast: %d in the air and %+v on the ground, wanted the spear lying there", spear.Name(), len(g.missiles), g.drops)
		}
		if d := g.drops[0].Pos; math.Abs(d.Z-7.5) > 2.5 {
			t.Errorf("the %s lies %.1f from the beast it hit", spear.Name(), d.Z-7.5)
		}
	}
}

func TestAMossbackIsNotHurtByAThrownSpear(t *testing.T) {
	g := thrower(t, world.Spear, -0.15) // a little down: a mossback is low
	one := quarry(5)
	one.kind, one.young, one.health, one.Pos.X, one.Pos.Z = render.Mossback, false, 0, 0.5, 5.5
	g.fauna.all = []beast{one}
	hold(t, g, 1, 'r')
	flies(t, g)
	if c := g.fauna.all[0]; c.dead || c.flee <= 0 {
		t.Errorf("a mossback hit by a spear: dead %v, running %v: wanted it alive and running", c.dead, c.flee > 0)
	}
}

func TestASlingThrowsPebblesFartherThanASpearFlies(t *testing.T) {
	const pitch = 0.3
	g := thrower(t, world.Sling, pitch)
	g.items[1] = world.Stack{Item: world.Pebble, Count: 5}
	hold(t, g, 1, 'r')
	if len(g.missiles) != 1 || g.missiles[0].stack != (world.Stack{Item: world.Pebble, Count: 1}) {
		t.Fatalf("R with a sling: in the air %+v, wanted one pebble", g.missiles)
	}
	if g.items[1].Count != 4 || g.items[0] != (world.Stack{Item: world.Sling, Count: 1, Wear: 1}) {
		t.Errorf("after a shot: %+v pebbles and the sling %+v, wanted 4 and a use gone", g.items[1], g.items[0])
	}
	_, pebble := flies(t, g)
	if want := reachOf(fliers[pebbleSlung], pitch); math.Abs(pebble.Z-0.5-want) > 2 {
		t.Errorf("the pebble came down %.1f ahead, wanted %.1f", pebble.Z-0.5, want)
	}

	spear := thrower(t, world.Spear, pitch)
	hold(t, spear, 1, 'r')
	if _, at := flies(t, spear); pebble.Z < at.Z+10 {
		t.Errorf("a pebble flew %.1f and a spear %.1f, wanted the pebble a good deal farther", pebble.Z, at.Z)
	}

	// It lies where it landed, to be picked up again.
	hold(t, g, seconds(1))
	if len(g.missiles) != 0 || len(g.drops) != 1 || g.drops[0].Stack != (world.Stack{Item: world.Pebble, Count: 1}) {
		t.Errorf("after it landed: %d in the air and %+v on the ground, wanted the pebble lying there", len(g.missiles), g.drops)
	}

	// Held, R throws again, but not at once.
	g = thrower(t, world.Sling, pitch)
	g.items[1] = world.Stack{Item: world.Pebble, Count: 5}
	hold(t, g, seconds(1), 'r')
	if thrown := 5 - g.items[1].Count; thrown < 2 || thrown > 3 {
		t.Errorf("a second of R threw %d pebbles, wanted two or three", thrown)
	}

	// With no pebbles nothing is thrown, and the player is told why.
	g = thrower(t, world.Sling, pitch)
	hold(t, g, 1, 'r')
	if len(g.missiles) != 0 || g.items[0].Wear != 0 || g.notice == "" {
		t.Errorf("R with a sling and no pebbles: %d thrown, sling %+v, told %q", len(g.missiles), g.items[0], g.notice)
	}
}

func TestAPebbleAlarmsAHerdItLandsAmong(t *testing.T) {
	const pitch = 0.12
	lands := func(herd bool, with world.Item) (*Game, geometry.Vector3) {
		g := thrower(t, with, pitch)
		g.items[1] = world.Stack{Item: world.Pebble, Count: 1}
		at := reachOf(fliers[pebbleSlung], pitch) + 0.5
		if herd {
			g.fauna = herdOf(6, 0.5, at)
		}
		hold(t, g, 1, 'r')
		for range seconds(10) {
			if len(g.missiles) == 0 || g.missiles[0].landed {
				break
			}
			if g.fauna.alarms[1].left > 0 {
				t.Fatalf("the herd took alarm while the pebble was in the air at %+v", g.missiles[0].pos)
			}
			hold(t, g, 1)
		}
		hold(t, g, 2)
		return g, geometry.Vector3{Z: at}
	}

	g, at := lands(true, world.Sling)
	if at.Z < 20 {
		t.Fatalf("the herd stands %.1f off, too near to be calm", at.Z)
	}
	if g.fauna.alarms[1].left <= 0 {
		t.Fatalf("a pebble landed among a herd %.0f blocks off and it took no alarm", at.Z)
	}
	for _, c := range g.fauna.all {
		if c.flee <= 0 || c.health != breeds[render.Bugler].health {
			t.Errorf("a bugler of the herd: running %v with health %v, wanted it running and not hit", c.flee > 0, c.health)
		}
	}

	// A herd that nothing lands near stays calm.
	calm := thrower(t, world.Sling, pitch)
	calm.fauna = herdOf(6, 30.5, at.Z)
	calm.items[1] = world.Stack{Item: world.Pebble, Count: 1}
	hold(t, calm, 1, 'r')
	flies(t, calm)
	hold(t, calm, 2)
	if calm.fauna.alarms[1].left > 0 {
		t.Error("a pebble that landed 30 blocks from a herd alarmed it")
	}
}

func TestRThrowsNothingButASpearOrWithASling(t *testing.T) {
	for _, with := range []world.Item{world.NoItem, world.Club, world.Stick, world.Pebble, world.Stone.Item(), world.FlintKnife} {
		g := thrower(t, with, 0.2)
		hold(t, g, 2, 'r')
		if len(g.missiles) != 0 || g.items[0].Item != with {
			t.Errorf("R with %q in hand threw %d things and left %+v", with.Name(), len(g.missiles), g.items[0])
		}
	}
}

// A spear that comes down nearly level still sticks: what holds it is the block it met.
func TestASpearThrownLevelSticksToo(t *testing.T) {
	g := thrower(t, world.Spear, 0.02)
	hold(t, g, 1, 'r')
	flies(t, g)
	hold(t, g, seconds(2))
	if len(g.missiles) != 1 || !g.missiles[0].landed || len(g.drops) != 0 {
		t.Errorf("a spear thrown level: %d stuck and %d lying two seconds on, wanted it stuck", len(g.missiles), len(g.drops))
	}
	if in := g.missiles[0].stuck; g.world.At(in.X, in.Y, in.Z) != world.Stone {
		t.Errorf("it sticks in %v at %+v, wanted the stone of the ground", g.world.At(in.X, in.Y, in.Z), in)
	}
}

func TestAThrownSpearStopsAtAWallAndFallsWhenTheWallIsBroken(t *testing.T) {
	g := inSurvival(t)
	g.fauna.all, g.fauna.timer = nil, math.Inf(1)
	wallAt(g, 26) // its face is 5.5 ahead
	g.player.Pitch = 0.1
	holding(g, world.Spear, 1)
	hold(t, g, 1, 'r')
	_, at := flies(t, g)
	if at.Z < 25 || at.Z >= 26 || at.Y < 7 {
		t.Fatalf("the spear stopped at %+v, wanted it stuck in the face of the wall at z=26", at)
	}
	hold(t, g, seconds(1))
	if len(g.missiles) != 1 {
		t.Fatalf("a second on %d spears stick in the wall, wanted the one", len(g.missiles))
	}

	for y := 6; y < 14; y++ {
		g.world.Set(20, y, 26, world.Air)
	}
	hold(t, g, seconds(2))
	if len(g.missiles) != 0 || g.items.count([]world.Item{world.Spear}) != 1 && (len(g.drops) != 1 || g.drops[0].Pos.Y > 6.5) {
		t.Errorf("with the wall gone: %d stuck, drops %+v, wanted the spear fallen to the ground", len(g.missiles), g.drops)
	}
}

func TestAWornSpearBreaksWhenItLands(t *testing.T) {
	g := thrower(t, world.Spear, 0.2)
	g.items[0].Wear = world.Spear.Uses() - 1
	hold(t, g, 1, 'r')
	flies(t, g)
	hold(t, g, seconds(1))
	if len(g.missiles) != 0 || len(g.drops) != 0 {
		t.Errorf("a spear on its last use: %d stuck and %d lying, wanted it broken", len(g.missiles), len(g.drops))
	}
}

func TestInCreativeAThrownSpearIsKeptAndLeavesNothing(t *testing.T) {
	g := onFloor(t)
	g.items[0] = world.Stack{Item: world.Spear, Count: 1}
	hold(t, g, 1, 'r') // level, to come down on the floor
	if len(g.missiles) != 1 || g.items[0] != (world.Stack{Item: world.Spear, Count: 1}) {
		t.Fatalf("in creative R with a spear: %d thrown and %+v in hand", len(g.missiles), g.items[0])
	}
	flies(t, g)
	hold(t, g, seconds(1))
	if len(g.missiles) != 0 || len(g.drops) != 0 {
		t.Errorf("in creative a thrown spear left %d stuck and %d lying", len(g.missiles), len(g.drops))
	}
}

func TestWhatWasThrownIsSavedAsAThingOnTheGround(t *testing.T) {
	path := filepath.Join(t.TempDir(), "world")
	g := inSurvival(t)
	g.opts.SavePath = path
	g.fauna.all, g.fauna.timer = nil, math.Inf(1)
	holding(g, world.Spear, 1)
	hold(t, g, 1, 'r') // level, to come down on the floor
	flies(t, g)
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}
	back := savedGame(t, path)
	if len(back.drops) != 1 || back.drops[0].Stack != (world.Stack{Item: world.Spear, Count: 1, Wear: 1}) {
		t.Errorf("the spear stuck in the ground came back as %+v, wanted it lying there", back.drops)
	}
}

func TestASlingAndPebblesAreMadeByHand(t *testing.T) {
	g := inSurvival(t)
	g.items[0] = world.Stack{Item: world.Cord, Count: 2}
	g.items[1] = world.Stack{Item: world.Hide, Count: 1}
	g.items[2] = world.Stack{Item: world.Gravel.Item(), Count: 1}
	for item, n := range map[world.Item]int{world.Sling: 1, world.Pebble: 4} {
		r, ok := recipeFor(g, item)
		if !ok || r.at != byHand || !g.craft(r) {
			t.Fatalf("%s could not be made by hand of 2 cords, a hide and gravel", item.Name())
		}
		if got := g.items.count([]world.Item{item}); got != n {
			t.Errorf("made %d of %s, wanted %d", got, item.Name(), n)
		}
	}
	if left := g.items.count([]world.Item{world.Cord, world.Hide, world.Gravel.Item()}); left != 0 {
		t.Errorf("%d of what they are made of is left", left)
	}
}

func TestThePauseMenuSaysWhatRDoesAndEveryLineOfItFits(t *testing.T) {
	said := false
	for _, line := range help {
		if len(line) > helpWidth-4 {
			t.Errorf("the line %q is %d long, and the menu shows %d", line, len(line), helpWidth-4)
		}
		said = said || strings.HasPrefix(line, "R ") && strings.Contains(line, "throw")
	}
	if !said {
		t.Error("no line of the pause menu says what R does")
	}
}

// What a frame costs with thirty spears stuck in the ground in front of the player.
func BenchmarkThrown(b *testing.B) {
	g, err := New(Options{Seed: 1})
	if err != nil {
		b.Fatal(err)
	}
	g.Play()
	fb, err := engine.NewFrameBuffer(240, 67)
	if err != nil {
		b.Fatal(err)
	}
	ahead := g.player.Camera().Look()
	x, z := int(math.Floor(g.player.Pos.X)), int(math.Floor(g.player.Pos.Z))
	ground := world.Cell{X: x, Y: g.world.Ground(x, z), Z: z} // a block that is there, for them to stick in
	for i := range 30 {
		at := g.player.Pos.Add(ahead.Scale(3 + float64(i%10))).Add(geometry.Vector3{X: float64(i/10) - 1, Y: 1})
		g.missiles = append(g.missiles, missile{kind: spearThrown, stack: world.Stack{Item: world.Spear, Count: 1}, pos: at, vel: ahead, stuck: ground})
	}
	kb := engine.NewKeyboard()
	for b.Loop() {
		for i := range g.missiles {
			g.missiles[i].landed, g.missiles[i].age = true, 0 // so that they stay to be drawn
		}
		if err := g.Update(1.0/60, kb.Input()); err != nil {
			b.Fatal(err)
		}
		kb.EndFrame()
		fb.Clear()
		g.Draw(fb)
	}
	if len(g.missiles) != 30 {
		b.Fatalf("%d of the 30 spears are left to draw", len(g.missiles))
	}
}

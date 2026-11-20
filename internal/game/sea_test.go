package game

import (
	"math"
	"math/rand/v2"
	"testing"

	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/world"
)

// The sea of the tests: water on the plain from 0 to 30 along x and 0 to 50 along z,
// walled round with stone, with its top at seaTop. It is eight deep, and from z =
// seaShelf on a shelf of stone lies under it, where it is two deep.
const (
	seaTop   = 14.0
	seaShelf = 40
)

// sea is a game in survival at noon of the third day with that sea in it, the beasts
// given, and the player at x, z: afloat with its feet a block under the top where that
// is in the sea, and on the plain where it is not.
func sea(t *testing.T, x, z float64, beasts ...beast) *Game {
	t.Helper()
	g := atHour(t, 0.5, z, beasts...)
	for bz := -1; bz <= 51; bz++ {
		for bx := -1; bx <= 31; bx++ {
			wall := bx < 0 || bx > 30 || bz < 0 || bz > 50
			for y := 6; y < int(seaTop); y++ {
				b := world.Water
				if wall || bz >= seaShelf && y < int(seaTop)-2 {
					b = world.Stone
				}
				g.world.Set(bx, y, bz, b)
			}
		}
	}
	afloat(g, x, z)
	return g
}

// afloat puts the player at x, z: in the sea if that is in it, and on the plain if not.
func afloat(g *Game, x, z float64) {
	at := geometry.Vector3{X: x, Y: 6, Z: z}
	if x > 0 && x < 31 && z > 0 && z < 51 {
		at.Y = seaTop - 1
	}
	stand(g, at)
}

// swimmer is a beast of a kind that swims, at x, z in the sea, as far under the top as
// its kind lies, that goes about as it will.
func swimmer(kind render.Kind, x, z float64) beast {
	c := giant(kind, x, z)
	c.Pos.Y, c.timer = seaTop-breeds[kind].draft, 0
	return c
}

// overDeep is how deep the water under a beast is.
func overDeep(g *Game, c *beast) int {
	return deepAt(g.world, int(math.Floor(c.Pos.X)), int(math.Floor(c.Pos.Z)))
}

func TestSwimmersKeepToWaterDeepEnoughForThem(t *testing.T) {
	for _, kind := range []render.Kind{render.Lakeneck, render.Leapfin, render.Deepmaw} {
		b := breeds[kind]
		if b.moves != gaitSwim || b.deep < 3 || b.draft <= 0 {
			t.Fatalf("a %s moves in way %d, in water %d deep or more, %.1f under the top", kind, b.moves, b.deep, b.draft)
		}
		g := sea(t, 15.5, -20.5, swimmer(kind, 15.5, 20.5))
		c := &g.fauna.all[0]
		from, far, dry := c.Pos, 0.0, 0.0
		for range seconds(90) {
			hold(t, g, 1)
			far = max(far, math.Hypot(c.Pos.X-from.X, c.Pos.Z-from.Z))
			if dry += frame; inWater(g.world, c.Pos) {
				dry = 0
			}
			if deep := overDeep(g, c); deep < b.deep || dry > 1.5 || c.dead {
				t.Fatalf("a %s is at %+v, over water %d deep, and has been out of it for %.1f s: wanted it in water %d deep or more", kind, c.Pos, deep, dry, b.deep)
			}
			if b.leaps == 0 && c.Pos.Y > seaTop-b.draft+0.3 {
				t.Fatalf("a %s that does not leap is at height %.2f: wanted it %.1f under the top or lower", kind, c.Pos.Y, b.draft)
			}
		}
		if far < 5 {
			t.Errorf("in a minute and a half a %s got no farther than %.1f blocks from where it began", kind, far)
		}
	}
	if d := breeds[render.Deepmaw]; d.deep != 6 || d.bite != 10 || d.temper != hunter || d.most != 1 {
		t.Errorf("a deepmaw keeps to water %d deep, bites for %v and there are %d at the most: wanted 6, 10 and 1", d.deep, d.bite, d.most)
	}
	// What is not water is no depth of it, and water is as deep as it is.
	g := sea(t, 15.5, -20.5)
	for _, tc := range []struct{ x, z, want int }{{15, 20, 8}, {15, 45, 2}, {15, -5, 0}, {-1, 20, 0}} {
		if got := deepAt(g.world, tc.x, tc.z); got != tc.want {
			t.Errorf("the water at %d, %d is %d deep, wanted %d", tc.x, tc.z, got, tc.want)
		}
	}
}

func TestALakeneckSinksAndSwimsOffFromAPlayerWhoComesNear(t *testing.T) {
	b := breeds[render.Lakeneck]
	one, two := swimmer(render.Lakeneck, 14.5, 19.5), swimmer(render.Lakeneck, 17.5, 20.5)
	one.herd, two.herd = 1, 1
	one.timer, two.timer = math.Inf(1), math.Inf(1)
	g := sea(t, 15.5, 1.5, one, two) // eighteen blocks from them
	top := seaTop - b.draft
	hold(t, g, seconds(3), ' ')
	for i := range g.fauna.all {
		if c := g.fauna.all[i]; math.Abs(c.Pos.Y-top) > 0.1 || c.flee > 0 {
			t.Fatalf("a lakeneck with a player eighteen blocks off lies at %.2f, running: %v: wanted it at the top, at %.2f", c.Pos.Y, c.flee > 0, top)
		}
	}
	if _, height := render.Size(render.Lakeneck, false); top+height < seaTop+1.5 {
		t.Errorf("at the top a lakeneck's head is %.1f over the water: wanted its neck out of it", top+height-seaTop)
	}

	// The player swims at them. They take alarm, sink and swim off.
	g.player.Yaw = yawOf(geometry.Vector3{Z: 1})
	alarmed, was := -1, [2]float64{}
	for i := range seconds(12) {
		hold(t, g, 1, 'w', ' ')
		if alarmed < 0 && g.fauna.all[0].flee > 0 {
			alarmed = i
			for j := range was {
				was[j] = g.fauna.all[j].Pos.Distance(g.player.Pos)
			}
			if was[0] > b.wary+0.5 && was[1] > b.wary+0.5 {
				t.Fatalf("lakenecks took alarm at a player %.1f and %.1f blocks off, wanted %v", was[0], was[1], b.wary)
			}
		}
		if alarmed >= 0 && i == alarmed+seconds(3) {
			break
		}
	}
	if alarmed < 0 {
		t.Fatalf("lakenecks did not take alarm at a player who swam at them: the nearer is %.1f blocks off", g.fauna.all[0].Pos.Distance(g.player.Pos))
	}
	for j := range was {
		c := g.fauna.all[j]
		if c.Pos.Y > top-2 || !inWater(g.world, c.Pos) {
			t.Errorf("three seconds after it took alarm a lakeneck is at height %.2f: wanted it two blocks or more under %.2f", c.Pos.Y, top)
		}
		if now := c.Pos.Distance(g.player.Pos); now < was[j]+3 {
			t.Errorf("three seconds after it took alarm a lakeneck is %.1f blocks from the player, and was %.1f: wanted it to swim off", now, was[j])
		}
	}
	// Left alone, they come up again.
	afloat(g, 15.5, -20.5)
	hold(t, g, seconds(alarmTime+10))
	for j := range was {
		if c := g.fauna.all[j]; math.Abs(c.Pos.Y-top) > 0.1 || c.dead {
			t.Errorf("left alone, a lakeneck lies at %.2f: wanted it at the top again, at %.2f", c.Pos.Y, top)
		}
	}
	if g.health != maxHealth {
		t.Errorf("lakenecks took %v points from the player", maxHealth-g.health)
	}
}

func TestLeapfinsLeapClearOfTheWaterAndSwimBesideAPlayer(t *testing.T) {
	school := func() []beast {
		var all []beast
		for i := range 5 {
			c := swimmer(render.Leapfin, 20.5+float64(i%3), 30.5+float64(i/3)*1.5)
			c.herd, c.rng = 1, rand.New(rand.NewPCG(7, uint64(100+i)))
			all = append(all, c)
		}
		return all
	}
	// A school by itself for a minute: they leap, clear of the water, and come down in it.
	g := sea(t, 15.5, -20.5, school()...)
	leaps, high := 0, 0.0
	up := make([]bool, 5)
	for range seconds(60) {
		hold(t, g, 1)
		for i := range g.fauna.all {
			c := &g.fauna.all[i]
			clear := c.Pos.Y > seaTop+0.05
			if clear && !up[i] {
				leaps++
			}
			up[i], high = clear, max(high, c.Pos.Y-seaTop)
			if overDeep(g, c) < breeds[render.Leapfin].deep {
				t.Fatalf("a leapfin is at %+v, over water %d deep", c.Pos, overDeep(g, c))
			}
		}
	}
	if leaps < 4 || high > 2.5 {
		t.Errorf("in a minute five leapfins leapt clear of the water %d times, %.1f blocks over it at the most: wanted four leaps or more, and none over 2.5", leaps, high)
	}
	for i := range g.fauna.all {
		if c := g.fauna.all[i]; !inWater(g.world, c.Pos) && c.Pos.Y < seaTop {
			t.Errorf("a leapfin ended at %+v, out of the water", c.Pos)
		}
	}

	// A player who swims by, twenty blocks from them: they come, and keep by it.
	g = sea(t, 4.5, 6.5, school()...)
	g.player.Yaw = yawOf(geometry.Vector3{Z: 1})
	hold(t, g, seconds(6), 'w', ' ')
	near := 0
	for i := range g.fauna.all {
		if g.fauna.all[i].Pos.Distance(g.player.Pos) < 7 {
			near++
		}
	}
	if near < 3 || g.player.Pos.Z < 20 || g.player.Pos.Z > seaShelf-3 || g.health != maxHealth {
		t.Errorf("after six seconds of swimming the player is at z = %.1f with %d of five leapfins within seven blocks, and has lost %v points", g.player.Pos.Z, near, maxHealth-g.health)
	}
	// One on the land they do not come to.
	g = sea(t, 15.5, -4.5, school()...)
	for i := range g.fauna.all {
		g.fauna.all[i].timer = math.Inf(1)
	}
	hold(t, g, seconds(6))
	for i := range g.fauna.all {
		if c := g.fauna.all[i]; c.walking {
			t.Fatalf("a leapfin at %+v swims to a player on the land", c.Pos)
		}
	}
}

func TestADeepmawHuntsInTheDeepAndItsBackBreaksTheWaterBeforeItComes(t *testing.T) {
	b := breeds[render.Deepmaw]
	_, height := render.Size(render.Deepmaw, false)
	lurking := func() beast {
		c := swimmer(render.Deepmaw, 15.5, 34.5)
		c.Pos.Y, c.timer = seaTop-b.draft-b.sinks, math.Inf(1)
		return c
	}

	// A player on the land it takes no notice of, and it keeps under.
	g := sea(t, 15.5, -4.5, lurking())
	c := &g.fauna.all[0]
	for range seconds(6) {
		hold(t, g, 1)
		if c.keen > 0 || c.Pos.Y+height > seaTop-0.5 {
			t.Fatalf("with the player on the land a deepmaw is keen: %v, with its back at %.2f under the top", c.keen > 0, seaTop-c.Pos.Y-height)
		}
	}
	// Nor of one in the deep on the first day.
	g.days = 0.2
	afloat(g, 15.5, 10.5)
	hold(t, g, seconds(6), ' ')
	if c.keen > 0 || g.health != maxHealth {
		t.Fatalf("on the first day a deepmaw went for a player in the deep: keen %v, %v points lost", c.keen > 0, maxHealth-g.health)
	}

	// A player in the deep, 24 blocks off: its back breaks the water far off, and
	// then it comes, and bites for 10.
	g = sea(t, 15.5, 10.5, lurking())
	c = &g.fauna.all[0]
	broke, bitten := -1.0, -1.0
	for i := range seconds(15) {
		hold(t, g, 1, ' ')
		if broke < 0 && c.Pos.Y+height > seaTop {
			broke = c.Pos.Distance(g.player.Pos)
			if g.health != maxHealth {
				t.Fatalf("a deepmaw bit before its back broke the water")
			}
		}
		if overDeep(g, c) < b.deep {
			t.Fatalf("a deepmaw is at %+v, over water %d deep", c.Pos, overDeep(g, c))
		}
		if g.health < maxHealth {
			bitten = float64(i) * frame
			break
		}
	}
	if broke < 8 || bitten < 2 || maxHealth-g.health != b.bite {
		t.Errorf("a deepmaw's back broke the water %.1f blocks from the player, and %.1f s on it had taken %v points: wanted its back out eight blocks off or more, two seconds or more before a bite of %v", broke, bitten, maxHealth-g.health, b.bite)
	}
}

func TestADeepmawHuntsTheBeastsOfTheSeaAndAHunterOfTheLandDoesNot(t *testing.T) {
	// A leapfin that keeps still, ten blocks from a deepmaw that lurks, with the
	// player on the land: the deepmaw comes up, and kills it.
	fish := swimmer(render.Leapfin, 15.5, 20.5)
	fish.timer = math.Inf(1)
	maw := swimmer(render.Deepmaw, 15.5, 8.5)
	maw.Pos.Y, maw.timer = seaTop-breeds[render.Deepmaw].draft-breeds[render.Deepmaw].sinks, math.Inf(1)
	g := sea(t, 15.5, -20.5, fish, maw)
	hold(t, g, seconds(12))
	if left := only(g.fauna.all, render.Leapfin); len(left) != 0 && !left[0].dead {
		t.Errorf("twelve seconds on, a leapfin ten blocks from a deepmaw has %v health, and the deepmaw is at %+v", left[0].health, only(g.fauna.all, render.Deepmaw)[0].Pos)
	}
	// A quickclaw on the bank does not go for a leapfin three blocks out, nor a deepmaw
	// for a nipper on the bank.
	claw, nip := giant(render.Quickclaw, 15.5, -1.5), giant(render.Nipper, 15.5, -1.5)
	claw.Pos.Y, nip.Pos.Y = seaTop, seaTop
	f := &fauna{all: []beast{claw, swimmer(render.Leapfin, 15.5, 2.5)}}
	if p := f.prey(&f.all[0]); p != nil {
		t.Errorf("a quickclaw on the bank goes for a %s in the sea", p.kind)
	}
	f = &fauna{all: []beast{swimmer(render.Deepmaw, 15.5, 4.5), nip}}
	if p := f.prey(&f.all[0]); p != nil {
		t.Errorf("a deepmaw goes for a %s on the bank", p.kind)
	}
}

func TestInTheShallowsAPlayerIsSafeFromADeepmaw(t *testing.T) {
	b := breeds[render.Deepmaw]
	// The player stands on the shelf, four blocks in from the deep, and the deepmaw
	// is ten blocks off.
	g := sea(t, 15.5, seaShelf+4.5, swimmer(render.Deepmaw, 15.5, seaShelf-5.5))
	stand(g, geometry.Vector3{X: 15.5, Y: seaTop - 2, Z: seaShelf + 4.5})
	c := &g.fauna.all[0]
	for range seconds(30) {
		hold(t, g, 1, ' ')
		if c.keen > 0 || g.health != maxHealth || overDeep(g, c) < b.deep {
			t.Fatalf("with the player in the shallows a deepmaw at %+v is keen: %v, over water %d deep, and the player has lost %v points", c.Pos, c.keen > 0, overDeep(g, c), maxHealth-g.health)
		}
	}

	// A player it is after swims for the shelf. Once in the shallows it is not hurt
	// again, and the deepmaw stops at the edge of the deep.
	g = sea(t, 15.5, seaShelf-12.5, swimmer(render.Deepmaw, 15.5, 4.5))
	g.player.Yaw = yawOf(geometry.Vector3{Z: 1})
	c = &g.fauna.all[0]
	keen, safe, came := false, -1.0, 0.0
	for range seconds(30) {
		if g.player.Pos.Z < seaShelf+5 {
			hold(t, g, 1, 'w', ' ')
		} else {
			hold(t, g, 1, ' ') // five blocks into the shallows it stops
		}
		keen, came = keen || c.keen > 0, max(came, c.Pos.Z)
		if overDeep(g, c) < b.deep {
			t.Fatalf("a deepmaw that is after a player is at %+v, over water %d deep", c.Pos, overDeep(g, c))
		}
		p := g.player.Pos
		if shallow := deepAt(g.world, int(math.Floor(p.X)), int(math.Floor(p.Z))) < b.deep; shallow && safe < 0 {
			safe = g.health
		} else if shallow && g.health < safe {
			t.Fatalf("in the shallows, at z = %.1f, the player lost %v points to a deepmaw at %+v", p.Z, safe-g.health, c.Pos)
		}
	}
	if !keen || safe < 0 || g.player.Pos.Z < seaShelf+3 {
		t.Errorf("the deepmaw went for the player: %v, and the player got to z = %.1f, in the shallows: %v", keen, g.player.Pos.Z, safe >= 0)
	}
	if came < 20 {
		t.Errorf("the deepmaw came as far as z = %.1f: wanted it to come after the player, toward the shelf at %d", came, seaShelf)
	}
}

// atSea returns a world of a seed with all of it loaded round a place far out at sea,
// and that place, at the top of the water.
func atSea(t *testing.T, seed int64) (*world.World, geometry.Vector3) {
	t.Helper()
	w := world.Generate(seed)
	for far := 0; far < 9000; far += 60 {
		for turn := range 16 {
			x := int(float64(far) * math.Cos(float64(turn)*math.Pi/8))
			z := int(float64(far) * math.Sin(float64(turn)*math.Pi/8))
			out := true
			for _, d := range [][2]int{{0, 0}, {40, 0}, {-40, 0}, {0, 40}, {0, -40}} {
				out = out && world.BiomeAt(seed, x+d[0], z+d[1]) == world.Sea
			}
			if !out {
				continue
			}
			w.Follow(float64(x), float64(z))
			w.LoadAll()
			if w.At(x, world.SeaLevel, z) != world.Water || deepAt(w, x, z) < 8 {
				continue // ice, or a bank
			}
			return w, geometry.Vector3{X: float64(x) + 0.5, Y: world.SeaLevel + 1, Z: float64(z) + 0.5}
		}
	}
	t.Fatalf("seed %d has no open sea within 9,000 blocks", seed)
	return nil, geometry.Vector3{}
}

func TestTheBeastsOfTheSeaAreFoundInTheSeasOfRealWorlds(t *testing.T) {
	found := map[render.Kind]int{}
	for seed := int64(1); seed <= 5; seed++ {
		w, at := atSea(t, seed)
		f := newFauna(w, at)
		herds := map[render.Kind]map[int]int{}
		for i := range f.all {
			c := &f.all[i]
			b := breeds[c.kind]
			if b.moves != gaitSwim {
				if inWater(w, c.Pos) && b.moves != gaitFly && b.moves != gaitWade {
					t.Errorf("seed %d: a %s was put in the sea, at %+v", seed, c.kind, c.Pos)
				}
				continue
			}
			found[c.kind]++
			if herds[c.kind] == nil {
				herds[c.kind] = map[int]int{}
			}
			herds[c.kind][c.herd]++
			x, z := int(math.Floor(c.Pos.X)), int(math.Floor(c.Pos.Z))
			if deep := deepAt(w, x, z); deep < b.deep || !inWater(w, c.Pos) || math.Abs(c.Pos.Y-(world.SeaLevel+1-b.draft)) > 0.01 {
				t.Errorf("seed %d: a %s was put at %+v, over water %d deep: wanted it at the top of water %d deep or more", seed, c.kind, c.Pos, deep, b.deep)
			}
		}
		for kind, of := range herds {
			b := breeds[kind]
			if f.count(kind) > b.most {
				t.Errorf("seed %d: %d of the %s, and %d at the most", seed, f.count(kind), kind, b.most)
			}
			for herd, n := range of {
				if n < min(b.group[0], 2) || n > b.group[1] {
					t.Errorf("seed %d: school %d of the %s is %d strong, wanted %v", seed, herd, kind, n, b.group)
				}
			}
		}
		// Two minutes with nobody in the water: every one that lives keeps to water
		// deep enough for it, and none is left on a bank. A deepmaw may take some.
		f.timer = math.Inf(1)
		sky := at.Add(geometry.Vector3{Y: 60})
		dry := map[*rand.Rand]float64{}
		for range seconds(120) {
			f.step(w, sky, frame)
			for i := range f.all {
				c := &f.all[i]
				b := breeds[c.kind]
				if b.moves != gaitSwim || c.dead {
					continue
				}
				if dry[c.rng] += frame; inWater(w, c.Pos) {
					dry[c.rng] = 0
				}
				x, z := int(math.Floor(c.Pos.X)), int(math.Floor(c.Pos.Z))
				if deep := deepAt(w, x, z); deep < b.deep || dry[c.rng] > 1.5 {
					t.Fatalf("seed %d: a %s is at %+v, over water %d deep, out of it for %.1f s", seed, c.kind, c.Pos, deep, dry[c.rng])
				}
			}
		}
	}
	for _, kind := range []render.Kind{render.Lakeneck, render.Leapfin, render.Deepmaw} {
		if found[kind] < 2 {
			t.Errorf("%d of the %s were found in the seas of five worlds", found[kind], kind)
		}
	}
}

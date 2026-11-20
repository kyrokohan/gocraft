package game

import (
	"math"
	"math/rand/v2"
	"testing"

	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/world"
)

// atHour is hunting, at an hour of the third day: 0.25 is sunrise and 0.5 noon.
func atHour(t *testing.T, hour float64, z float64, beasts ...beast) *Game {
	t.Helper()
	g := hunting(t, z, beasts...)
	g.days = 2 + hour - firstMorning
	return g
}

func TestASunbackIsSlowInTheMorningAndQuickAtNoon(t *testing.T) {
	// How far one runs at a player 40 blocks off, in five seconds, at an hour of the day.
	ran := func(hour float64, winter bool) float64 {
		g := atHour(t, hour, 0.5, giant(render.Sunback, 0.5, 40.5))
		if winter {
			g.days = arrival + (fallLasts/2+dayLength*(1.2+hour))/dayLength
			g.days = math.Floor(g.days) + hour - firstMorning // the same hour, in the winter
			if !g.winter() || math.Abs(timeOfDay(g.days)-hour) > 1e-9 {
				t.Fatalf("day %.2f is in the winter: %v, at hour %.2f", g.days, g.winter(), timeOfDay(g.days))
			}
		}
		g.fauna.all[0].keen = keenTime // it has seen the player, far off as it is
		for range seconds(5) {
			hold(t, g, 1)
			g.fauna.all[0].keen = keenTime
		}
		return 40 - (g.fauna.all[0].Pos.Z - 0.5)
	}
	dawn, noon, dusk, cold := ran(0.29, false), ran(0.5, false), ran(0.72, false), ran(0.5, true)
	b := breeds[render.Sunback]
	if noon < 0.9*5*b.run || dawn > 0.65*noon || dusk > 0.65*noon || dawn < 0.3*noon {
		t.Errorf("a sunback ran %.1f blocks in five seconds an hour after sunrise, %.1f at noon and %.1f before sunset: wanted all its %v a second at noon, and about half of it at the ends of the day", dawn, noon, dusk, b.run)
	}
	if cold > 0.5*noon {
		t.Errorf("a sunback ran %.1f blocks in five seconds at noon in the winter, and %.1f out of it: wanted it slow", cold, noon)
	}
	// At noon it is quicker than a player, and in the morning it is not.
	if noon/5 <= walkSpeed || dawn/5 >= walkSpeed {
		t.Errorf("a sunback goes %.1f blocks a second at noon and %.1f in the morning, and a player %.1f: wanted it to catch one at noon and not in the morning", noon/5, dawn/5, walkSpeed)
	}

	// It bites, and it lies still in the dark.
	g := atHour(t, 0.5, 0.5, giant(render.Sunback, 0.5, 6.5))
	hold(t, g, seconds(4))
	if lost := maxHealth - g.health; lost < b.bite {
		t.Errorf("a sunback six blocks off at noon took %v points in four seconds, wanted a bite of %v", lost, b.bite)
	}
	g = atHour(t, 0, 0.5, giant(render.Sunback, 0.5, 8.5))
	hold(t, g, seconds(10))
	if c := g.fauna.all[0]; g.health != maxHealth || c.Pos.Z < 8 {
		t.Errorf("at midnight a sunback eight blocks off went to z = %.1f, and the player lost %v points", c.Pos.Z, maxHealth-g.health)
	}
}

// pond is a game in survival at noon of the third day, on the plain, with a pond so many
// blocks deep from 10 to 20 along x and z, its top level with the ground, and a sailjaw
// in the middle of it, on its floor.
func pond(t *testing.T, deep int, px, pz float64) *Game {
	t.Helper()
	g := atHour(t, 0.5, pz)
	g.player.Pos.X = px
	for z := 10; z <= 20; z++ {
		for x := 10; x <= 20; x++ {
			g.world.Set(x, 5-deep, z, world.Stone)
			for y := 6 - deep; y <= 5; y++ {
				g.world.Set(x, y, z, world.Water)
			}
		}
	}
	one := giant(render.Sailjaw, 15.5, 15.5)
	one.Pos.Y, one.timer = float64(6-deep), 0
	g.fauna.all = []beast{one}
	return g
}

func TestASailjawKeepsToItsWaterAndLungesAtWhatComesNear(t *testing.T) {
	// In water one deep, two or four, it lies with its feet one block under the top:
	// its back is under the water, and its sail stands out of it.
	for _, deep := range []int{1, 2, 4} {
		// With nobody near it goes about its pond for a minute, and never leaves it.
		g := pond(t, deep, 60.5, 60.5)
		far := 0.0
		for i := range seconds(60) {
			hold(t, g, 1)
			c := &g.fauna.all[0]
			far = max(far, math.Hypot(c.Pos.X-15.5, c.Pos.Z-15.5))
			if c.Pos.X < 9.9 || c.Pos.X > 21.1 || c.Pos.Z < 9.9 || c.Pos.Z > 21.1 || c.Pos.Y > 5.6 {
				t.Fatalf("in water %d deep a sailjaw with nobody near left its pond: it is at %+v", deep, c.Pos)
			}
			if i > seconds(5) && math.Abs(c.Pos.Y-5) > 0.05 {
				t.Fatalf("in water %d deep a sailjaw's feet are at %.2f: wanted them at 5, one block under the top", deep, c.Pos.Y)
			}
		}
		if far < 1.5 {
			t.Errorf("in a minute in water %d deep a sailjaw got no farther than %.1f blocks from the middle of its pond", deep, far)
		}

		// A player nine blocks off it leaves alone. One who comes within five it lunges at.
		g = pond(t, deep, 15.5, 24.5)
		g.fauna.all[0].timer = math.Inf(1)
		hold(t, g, seconds(8))
		if c := g.fauna.all[0]; g.health != maxHealth || c.leap > 0 || c.keen > 0 {
			t.Fatalf("a sailjaw took notice of a player nine blocks off: keen %.1f, the player has lost %v points", c.keen, maxHealth-g.health)
		}
		g.player.Pos.Z = 20.9 // on the bank, at the water's edge: five and a half blocks from it
		sprang := false
		for range seconds(8) {
			hold(t, g, 1)
			sprang = sprang || g.fauna.all[0].leap > 0
			if g.health < maxHealth {
				break
			}
		}
		b := breeds[render.Sailjaw]
		if lost := maxHealth - g.health; !sprang || lost != b.bite || lost != 9 {
			t.Fatalf("in water %d deep a sailjaw lunged at a player at the edge of its pond: %v, and took %v points: wanted a lunge, and 9", deep, sprang, lost)
		}

		// It does not follow far: the player goes thirty blocks off, and it is home again.
		g.player.Pos = geometry.Vector3{X: 15.5, Y: 6, Z: 55.5}
		g.health = maxHealth
		hold(t, g, seconds(30))
		if c := g.fauna.all[0]; g.health != maxHealth || !inWater(g.world, c.Pos) || c.Pos.Z > 21 {
			t.Errorf("in water %d deep, thirty seconds after the player went, the sailjaw is at %+v, in its water: %v, and the player has lost %v points", deep, c.Pos, inWater(g.world, c.Pos), maxHealth-g.health)
		}
	}
}

func TestSunbacksAndSailjawsAreFoundInTheirLands(t *testing.T) {
	suns, jaws := 0, 0
	for seed := int64(1); seed <= 6; seed++ {
		w, at := onSwamp(t, seed)
		f := newFauna(w, at)
		start := map[*rand.Rand]geometry.Vector3{}
		for i := range f.all {
			c := &f.all[i]
			x, z := int(math.Floor(c.Pos.X)), int(math.Floor(c.Pos.Z))
			switch c.kind {
			case render.Sunback:
				suns++
				if land := world.BiomeAt(seed, x, z); land != world.Swamp && land != world.Badlands {
					t.Errorf("seed %d: a sunback stands in land %d", seed, land)
				}
			case render.Sailjaw:
				jaws++
				start[c.rng] = c.Pos
				if y := int(c.Pos.Y); !inWater(w, c.Pos) || w.At(x, y+1, z) == world.Water || w.At(x, y-2, z) == world.Water {
					t.Errorf("seed %d: a sailjaw lies at %+v, in water: %v, with %s over its feet and %s two under them: wanted it at the top of water one or two deep", seed, c.Pos, inWater(w, c.Pos), w.At(x, y+1, z).Info().Name, w.At(x, y-2, z).Info().Name)
				}
			}
		}
		if f.count(render.Sunback) > breeds[render.Sunback].most || f.count(render.Sailjaw) > breeds[render.Sailjaw].most {
			t.Errorf("seed %d: %d sunbacks and %d sailjaws", seed, f.count(render.Sunback), f.count(render.Sailjaw))
		}
		// A sailjaw in a real pool keeps to it for two minutes: it may lunge out at a
		// beast on the bank, and is soon back.
		f.timer = math.Inf(1)
		sky := at.Add(geometry.Vector3{Y: 60})
		dry := map[*rand.Rand]float64{}
		for range seconds(120) {
			f.step(w, sky, frame)
			for i := range f.all {
				c := &f.all[i]
				from, ok := start[c.rng]
				if !ok || c.dead {
					continue
				}
				if dry[c.rng] += frame; inWater(w, c.Pos) {
					dry[c.rng] = 0
				}
				if far := math.Hypot(c.Pos.X-from.X, c.Pos.Z-from.Z); dry[c.rng] > 20 || far > 2*breeds[render.Sailjaw].tether {
					t.Fatalf("seed %d: a sailjaw has been out of the water for %.0f s, and is %.0f blocks from where it began", seed, dry[c.rng], far)
				}
			}
		}
	}
	if suns < 3 || jaws < 3 {
		t.Errorf("%d sunbacks and %d sailjaws were found in the swamps of six worlds", suns, jaws)
	}
}

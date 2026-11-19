package game

import (
	"math"
	"math/rand/v2"
	"path/filepath"
	"slices"
	"testing"

	"glonk.dev/engine"
	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/sound"
	"github.com/kyrokohan/skyspear/internal/world"
)

// plain is a floor of stone 120 blocks on a side, with room for a herd to run on.
func plain() *world.World {
	w := world.New()
	for z := -40; z < 80; z++ {
		for x := -40; x < 80; x++ {
			w.Set(x, 5, z, world.Stone)
		}
	}
	return w
}

// herdOf is a herd of buglers standing round a point of the plain, two of them young.
func herdOf(n int, x, z float64) *fauna {
	f := &fauna{rng: rand.New(rand.NewPCG(3, 4)), timer: math.Inf(1), herds: 1}
	for i := range n {
		angle := 2 * math.Pi * float64(i) / float64(n)
		f.all = append(f.all, beast{
			kind: render.Bugler, herd: 1, young: i >= n-2, health: breeds[render.Bugler].health,
			Pos:   geometry.Vector3{X: x + 3*math.Cos(angle), Y: 6, Z: z + 3*math.Sin(angle)},
			timer: float64(i) * 0.3, rng: rand.New(rand.NewPCG(5, uint64(i))),
		})
	}
	return f
}

func TestBuglersAreFoundInHerdsOnThePrairieAndTheHighland(t *testing.T) {
	found := 0
	for seed := int64(1); seed <= 8; seed++ {
		w := world.Generate(seed)
		spawn := w.Spawn()
		all, again := newFauna(w, spawn).all, newFauna(w, spawn).all
		if len(all) != len(again) {
			t.Fatalf("seed %d: %d beasts the first time and %d the second", seed, len(all), len(again))
		}

		herds := map[int][]beast{}
		for i, c := range all {
			if again[i].Pos != c.Pos || again[i].kind != c.kind || again[i].young != c.young {
				t.Errorf("seed %d: beast %d is not what and where it was the first time", seed, i)
			}
			if c.kind != render.Bugler {
				continue
			}
			found++
			x, y, z := int(math.Floor(c.Pos.X)), int(c.Pos.Y), int(math.Floor(c.Pos.Z))
			if land := world.BiomeAt(seed, x, z); land != world.Prairie && land != world.Highland {
				t.Errorf("seed %d: a bugler stands in land %d", seed, land)
			}
			if w.At(x, y-1, z) != world.Turf || w.At(x, y+1, z).Info().Solid || w.At(x, y+2, z).Info().Solid {
				t.Errorf("seed %d: a bugler at %v is not on mossy earth with room over it", seed, c.Pos)
			}
			if c.herd == 0 || c.health != breeds[render.Bugler].health {
				t.Errorf("seed %d: a bugler has herd %d and health %v", seed, c.herd, c.health)
			}
			herds[c.herd] = append(herds[c.herd], c)
		}
		if n := len(only(all, render.Bugler)); n > breeds[render.Bugler].most {
			t.Errorf("seed %d: %d buglers, and the most is %d", seed, n, breeds[render.Bugler].most)
		}
		for id, herd := range herds {
			grown := slices.ContainsFunc(herd, func(c beast) bool { return !c.young })
			if len(herd) < 2 || len(herd) > breeds[render.Bugler].group[1] || !grown {
				t.Errorf("seed %d: herd %d has %d buglers, a grown one among them: %v", seed, id, len(herd), grown)
			}
			for _, c := range herd {
				if d := c.Pos.Distance(herd[0].Pos); d > 2*groupReach+2 {
					t.Errorf("seed %d: a bugler of herd %d starts %v from the first of it", seed, id, d)
				}
			}
		}
	}
	if found < 10 {
		t.Errorf("%d buglers in eight worlds, wanted herds in some of them", found)
	}
}

func TestAHerdKeepsTogether(t *testing.T) {
	w, f := plain(), herdOf(7, 20, 20)
	far := geometry.Vector3{X: 20, Y: 6, Z: -30} // near enough that they move, too far to alarm them

	for range 60 * 30 {
		f.step(w, far, frame)
	}

	middle := f.middles()[1]
	for i, c := range f.all {
		if d := math.Hypot(c.Pos.X-middle.X, c.Pos.Z-middle.Z); d > 2.5*herdReach {
			t.Errorf("after a minute bugler %d is %.1f from the middle of its herd, wanted it kept within %.0f", i, d, 2.5*herdReach)
		}
		if !near(c.Pos.Y, 6) {
			t.Errorf("bugler %d ended at height %v, wanted it on the plain", i, c.Pos.Y)
		}
	}
	if len(f.alarms) != 0 || len(f.sounds) != 0 {
		t.Errorf("a herd left alone took alarm: %v", f.alarms)
	}
}

func TestOneAlarmRunsTheWholeHerdFromWhatScaredIt(t *testing.T) {
	w, f := plain(), herdOf(7, 20, 20)
	player := geometry.Vector3{X: 20, Y: 6, Z: 20 - 3 - breeds[render.Bugler].wary + 1} // just inside what the nearest will bear

	before := make([]float64, len(f.all))
	for i, c := range f.all {
		before[i] = c.Pos.Distance(player)
	}
	f.step(w, player, frame)
	if len(f.sounds) != 1 || f.sounds[0].name != sound.Alarm || f.alarms[1].left <= 0 {
		t.Fatalf("a player near a herd: sounds %v, alarms %v; wanted one cry and the herd alarmed", f.sounds, f.alarms)
	}
	for range 2 * 30 {
		f.step(w, player, frame)
	}
	for i, c := range f.all {
		if c.flee <= 0 || c.Pos.Distance(player) < before[i]+4 {
			t.Errorf("bugler %d went from %.1f to %.1f from the player in two seconds, wanted it running away", i, before[i], c.Pos.Distance(player))
		}
	}
	if len(f.sounds) != 0 {
		t.Errorf("the herd cried again while it ran: %v", f.sounds)
	}

	// A crouched player can come nearer.
	f = herdOf(7, 20, 20)
	f.sneaking = true
	f.step(w, player, frame)
	if len(f.alarms) != 0 {
		t.Error("a crouched player at the same distance alarmed the herd")
	}
	player.Z = 20 - 3 - breeds[render.Bugler].wary*sneakPart + 0.5
	f.step(w, player, frame)
	if len(f.alarms) != 1 {
		t.Error("a crouched player right beside the herd did not alarm it")
	}
}

func TestARunningBuglerHurtsThePlayerItRunsInto(t *testing.T) {
	charging := func() beast {
		return beast{
			kind: render.Bugler, health: 16, walking: true, flee: 5, timer: math.Inf(1),
			Pos: geometry.Vector3{X: 20.5, Y: 6, Z: 17}, rng: rand.New(rand.NewPCG(1, 2)),
		}
	}

	g := inSurvival(t) // the player is at 20.5, 20.5, and the bugler runs at it from behind
	g.fauna.all = []beast{charging()}
	from := g.player.Pos
	hold(t, g, 20)
	if g.health != maxHealth-breeds[render.Bugler].bump || g.flash == 0 {
		t.Errorf("health %v after a bugler ran the player down, wanted %v and a flash", g.health, maxHealth-breeds[render.Bugler].bump)
	}
	if g.player.Pos.Z < from.Z+bumpShove/2 {
		t.Errorf("the player was thrown from z=%v to %v, wanted it thrown the way the bugler ran", from.Z, g.player.Pos.Z)
	}

	easy := onFloor(t) // in creative
	easy.fauna.all = []beast{charging()}
	before := easy.health
	hold(t, easy, 20)
	if easy.health != before {
		t.Errorf("a bugler hurt a player in creative: health %v", easy.health)
	}
}

func TestABuglerDiesOfEnoughHitsAndLeavesMeat(t *testing.T) {
	g := inSurvival(t)
	g.fauna.all = []beast{{
		kind: render.Bugler, health: 3, herd: 4, timer: math.Inf(1),
		Pos: geometry.Vector3{X: 20.5, Y: 6, Z: 22.5}, rng: rand.New(rand.NewPCG(1, 2)),
	}}

	for hits := 1; hits <= 3; hits++ { // a bare hand takes a point a hit
		g.strike(prey{i: 0})
		if c := g.fauna.all[0]; c.dead != (hits == 3) {
			t.Fatalf("after %d hits the bugler is dead: %v", hits, c.dead)
		}
	}
	if _, ok := g.aimCreature(g.player.head(), geometry.Vector3{Z: 1}, reach); ok {
		t.Error("a bugler that is falling dead can still be aimed at")
	}

	hold(t, g, int(fallTime*30)+3)
	if len(g.fauna.all) != 0 {
		t.Fatalf("the dead bugler is still there after it fell: %+v", g.fauna.all)
	}
	meat := 0
	for _, d := range g.drops {
		if d.Item == world.Meat {
			meat += d.Count
		}
	}
	if meat != breeds[render.Bugler].meat {
		t.Errorf("the bugler left %d pieces of meat, wanted %d", meat, breeds[render.Bugler].meat)
	}
	if world.Meat.Food() != 5 || world.Meat.Name() != "raw meat" {
		t.Errorf("meat is %q and feeds %d", world.Meat.Name(), world.Meat.Food())
	}
}

func TestBeastsAreSavedWithTheirKindHealthAndHerd(t *testing.T) {
	path := filepath.Join(t.TempDir(), "world")
	g := savedGame(t, path)
	startNew(t, g, true)
	at := g.player.Pos
	g.fauna.all = []beast{
		{kind: render.Bugler, Pos: at.Add(geometry.Vector3{X: 5}), Yaw: 1, young: true, herd: 2, health: 7},
		{kind: render.Mossback, Pos: at.Add(geometry.Vector3{X: -5}), regrow: 40},
		{kind: render.Bugler, Pos: at.Add(geometry.Vector3{Z: 5}), herd: 2, health: 1, dead: true}, // dying: not kept
	}
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}

	back := savedGame(t, path)
	if err := press(back, engine.KeyEnter); err != nil {
		t.Fatal(err)
	}
	all := back.fauna.all
	if len(all) != 2 {
		t.Fatalf("came back with %d beasts, wanted the two that lived", len(all))
	}
	if c := all[0]; c.kind != render.Bugler || !c.young || c.herd != 2 || c.health != 7 || c.Pos != at.Add(geometry.Vector3{X: 5}) || c.Yaw != 1 {
		t.Errorf("the bugler came back as %+v", c)
	}
	if c := all[1]; c.kind != render.Mossback || c.regrow != 40 {
		t.Errorf("the mossback came back as %+v", c)
	}
	if back.fauna.herds < 2 {
		t.Errorf("the next herd would be numbered %d, and herd 2 is taken", back.fauna.herds+1)
	}
}

// A world saved before there were kinds holds mossbacks only, in a list of their own,
// and one saved by a later game may hold a kind this one does not have.
func TestOlderAndNewerSavesOfBeastsOpen(t *testing.T) {
	dir := t.TempDir()
	meta := world.Meta{
		Pos: standing().Pos, Mode: world.Survival, Health: maxHealth, Food: maxFood, Air: maxAir,
		Mossbacks: []world.Mossback{{Pos: geometry.Vector3{X: 25.5, Y: 6, Z: 20.5}, Regrow: 12}},
		Beasts: []world.Beast{
			{Kind: "gryphon", Pos: geometry.Vector3{X: 22.5, Y: 6, Z: 20.5}},
			{Kind: "bugler", Pos: geometry.Vector3{X: 15.5, Y: 6, Z: 20.5}, Health: 99},
		},
	}
	if err := floor().Save(dir, meta); err != nil {
		t.Fatal(err)
	}

	g := savedGame(t, dir)
	if err := press(g, engine.KeyEnter); err != nil {
		t.Fatal(err)
	}
	all := g.fauna.all
	if len(all) != 2 || all[0].kind != render.Mossback || all[0].regrow != 12 || all[1].kind != render.Bugler {
		t.Fatalf("the world opened with %+v, wanted its mossback and its bugler and no gryphon", all)
	}
	if all[1].health != breeds[render.Bugler].health {
		t.Errorf("a bugler saved with health 99 has %v, wanted no more than a bugler has", all[1].health)
	}
}

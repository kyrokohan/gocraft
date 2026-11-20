package game

import (
	"math"
	"math/rand/v2"
	"path/filepath"
	"testing"

	"glonk.dev/engine"
	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/sound"
	"github.com/kyrokohan/skyspear/internal/world"
)

// flockOf is a flock of nippers standing round a point of a floor.
func flockOf(n int, x, z float64) []beast {
	var all []beast
	for i := range n {
		angle := 2 * math.Pi * float64(i) / float64(n)
		all = append(all, beast{
			kind: render.Nipper, herd: 1, health: breeds[render.Nipper].health,
			Pos:   geometry.Vector3{X: x + 1.5*math.Cos(angle), Y: 6, Z: z + 1.5*math.Sin(angle)},
			timer: math.Inf(1), rng: rand.New(rand.NewPCG(7, uint64(i))),
		})
	}
	return all
}

// holding puts an item in the player's hand.
func holding(g *Game, item world.Item, count int) {
	g.items[g.selected] = world.Stack{Item: item, Count: count}
}

func TestNippersAreSmallQuickAndFoundInFlocks(t *testing.T) {
	b := breeds[render.Nipper]
	if w, h := render.Size(render.Nipper, false); w > 0.5 || h > 0.8 {
		t.Errorf("a nipper's box is %v by %v, wanted it the size of a hen", w, h)
	}
	if b.run <= walkSpeed || b.health <= 0 || b.health > 4 || b.meat != 1 || b.temper != thief {
		t.Errorf("a nipper runs at %v with %v health and %d meat, temper %d", b.run, b.health, b.meat, b.temper)
	}

	found := 0
	for seed := int64(1); seed <= 8; seed++ {
		w := realWorld(seed)
		flocks := map[int]int{}
		for _, c := range only(newFauna(w, w.Spawn()).all, render.Nipper) {
			found++
			flocks[c.herd]++
			if c.herd == 0 || c.young {
				t.Errorf("seed %d: a nipper has flock %d, young %v", seed, c.herd, c.young)
			}
		}
		for id, n := range flocks {
			if n < 2 || n > b.group[1] {
				t.Errorf("seed %d: flock %d has %d nippers", seed, id, n)
			}
		}
	}
	if found < 10 {
		t.Errorf("%d nippers in eight worlds, wanted flocks in some of them", found)
	}
}

func TestNippersScatterFromAPlayerWhoWalksAtThem(t *testing.T) {
	g := inSurvival(t) // at 20.5, 20.5, facing +z
	g.fauna.all = flockOf(4, 20.5, 26.5)
	holding(g, world.StonePick, 1)

	hush(g)
	hold(t, g, seconds(1), 'w')
	if h := heard(g); h[sound.Chirp] == 0 {
		t.Errorf("a flock that took alarm was not heard: %v", g.ears.Playing())
	}
	for i, c := range g.fauna.all {
		if c.flee <= 0 {
			t.Errorf("nipper %d does not run from a player who walked at the flock", i)
		}
	}
	hold(t, g, seconds(2))
	for i, c := range g.fauna.all {
		if d := c.Pos.Distance(g.player.Pos); d < 8 {
			t.Errorf("nipper %d is %.1f from the player after three seconds, wanted it well away", i, d)
		}
	}
}

func TestANipperTakesFoodFromTheHandAndNotATool(t *testing.T) {
	g := inSurvival(t)
	g.fauna.all = flockOf(4, 20.5, 26.5)
	holding(g, world.Berries, 5)

	hush(g)
	hold(t, g, seconds(4)) // standing still, with food held out
	if got := g.items[g.selected]; got.Item != world.Berries || got.Count != 4 {
		t.Fatalf("after four seconds near a flock the player holds %d of %s, wanted one berry taken", got.Count, got.Item.Name())
	}
	if h := heard(g); h[sound.Chirp] == 0 {
		t.Errorf("the theft was not heard: %v", g.ears.Playing())
	}
	thieves := 0
	for _, c := range g.fauna.all {
		if c.flee > 0 {
			thieves++
		}
	}
	if thieves == 0 {
		t.Error("no nipper runs off after the theft")
	}

	// They do not come for a tool, and nothing is taken in creative.
	g = inSurvival(t)
	g.fauna.all = flockOf(4, 20.5, 26.5)
	holding(g, world.StonePick, 1)
	hold(t, g, seconds(4))
	if got := g.items[g.selected]; got.Item != world.StonePick || got.Count != 1 {
		t.Errorf("a nipper took a tool: the player holds %d of %s", got.Count, got.Item.Name())
	}
	for i, c := range g.fauna.all {
		if d := c.Pos.Distance(g.player.Pos); d < 3 {
			t.Errorf("nipper %d came to %.1f from a player who holds no food", i, d)
		}
	}

	easy := onFloor(t)
	easy.fauna.all = flockOf(4, 20.5, 26.5)
	holding(easy, world.Berries, 1)
	hold(t, easy, seconds(4))
	if got := easy.items[easy.selected]; got.Item != world.Berries || got.Count != 1 {
		t.Errorf("a nipper took food in creative: the player holds %d of %s", got.Count, got.Item.Name())
	}
}

func TestANipperDiesOfAFewHitsAndLeavesAPieceOfMeat(t *testing.T) {
	g := inSurvival(t)
	g.fauna.all = flockOf(1, 20.5, 22)
	for range 3 { // a bare hand takes a point a hit
		g.strike(prey{i: 0})
	}
	hold(t, g, int(fallTime*30)+3)
	meat := 0
	for _, d := range g.drops {
		if d.Item == world.Meat {
			meat += d.Count
		}
	}
	if len(g.fauna.all) != 0 || meat != 1 {
		t.Errorf("after three hits %d nippers are left and %d pieces of meat lie there, wanted none and one", len(g.fauna.all), meat)
	}
}

// aimedAtMossback is a game in survival whose player aims at a mossback three blocks ahead.
func aimedAtMossback(t *testing.T) *Game {
	t.Helper()
	g := inSurvival(t)
	g.fauna.all = []beast{{
		Pos: geometry.Vector3{X: 20.5, Y: 6, Z: 22.5}, timer: math.Inf(1), rng: rand.New(rand.NewPCG(1, 2)),
	}}
	g.player.Pitch = -0.3
	hold(t, g, 1)
	if !g.preyed {
		t.Fatal("the player does not aim at the mossback")
	}
	return g
}

func TestThreeMealsFromTheHandTameAMossback(t *testing.T) {
	g := aimedAtMossback(t)
	holding(g, world.Berries, 5)
	g.food = maxFood - 6 // hungry, so that the berries could be eaten

	for meal := 1; meal <= 3; meal++ {
		hush(g)
		if err := press(g, 'e'); err != nil {
			t.Fatal(err)
		}
		c := g.fauna.all[0]
		if c.fed != min(meal, 3) || c.tame != (meal == 3) {
			t.Fatalf("after %d meals the mossback has had %d and is tame: %v", meal, c.fed, c.tame)
		}
		if got := g.items[g.selected].Count; got != 5-meal {
			t.Errorf("after %d meals the player has %d berries, wanted %d", meal, got, 5-meal)
		}
		if h := heard(g); h[sound.Eat] != 1 {
			t.Errorf("meal %d was not heard: %v", meal, g.ears.Playing())
		}
		hold(t, g, 12) // E is not held, and the mossback chews
	}
	if g.food != maxFood-6 {
		t.Errorf("the player's food went to %v: it ate the berries it fed the mossback", g.food)
	}
	text, _ := drawn(t, g)
	if want := "trusts you"; !contains(text, want) {
		t.Errorf("the screen does not say the mossback %s:\n%s", want, text)
	}

	// A tame one is fed no more: E with food in hand is the player's own meal again.
	hold(t, g, seconds(4))
	if err := press(g, 'e'); err != nil {
		t.Fatal(err)
	}
	if g.fauna.all[0].fed != 3 || g.food <= maxFood-6 {
		t.Errorf("fed %d, food %v: wanted the player to eat with a tame mossback before it", g.fauna.all[0].fed, g.food)
	}
}

func TestAMossbackTakesBerriesAndFruitAndNothingElse(t *testing.T) {
	for item, takes := range map[world.Item]bool{world.Berries: true, world.Fruit: true, world.Meat: false, world.Stick: false} {
		g := aimedAtMossback(t)
		holding(g, item, 3)
		if err := press(g, 'e'); err != nil {
			t.Fatal(err)
		}
		if fed := g.fauna.all[0].fed == 1; fed != takes {
			t.Errorf("offered %s the mossback ate: %v, wanted %v", item.Name(), fed, takes)
		}
	}
}

func TestATameMossbackFollowsThePlayerAndIsNeverLetGo(t *testing.T) {
	g := inSurvival(t)
	g.world = plain()
	g.player.Pos = geometry.Vector3{X: 0.5, Y: 6, Z: 0.5}
	g.fauna.all = []beast{
		{Pos: geometry.Vector3{X: 0.5, Y: 6, Z: 4.5}, tame: true, fed: 3, timer: math.Inf(1), rng: rand.New(rand.NewPCG(1, 2))},
		{Pos: geometry.Vector3{X: 4.5, Y: 6, Z: 0.5}, timer: math.Inf(1), rng: rand.New(rand.NewPCG(3, 4))},
	}

	// The player walks 40 blocks off. The tame one comes after it, and the wild one does not.
	g.player.Yaw = math.Pi // toward -z
	hold(t, g, seconds(9), 'w')
	hold(t, g, seconds(30))
	tame, wild := g.fauna.all[0], g.fauna.all[1]
	if d := tame.Pos.Distance(g.player.Pos); d > 7 {
		t.Errorf("the tame mossback is %.1f from the player, wanted it to have followed", d)
	}
	if d := wild.Pos.Distance(g.player.Pos); d < 20 {
		t.Errorf("the wild mossback is %.1f from the player: it followed too", d)
	}

	// It stands by a player who stands, and does not crowd it.
	before := tame.Pos
	hold(t, g, seconds(5))
	if moved := g.fauna.all[0].Pos.Distance(before); moved > 1.5 {
		t.Errorf("the tame mossback moved %.1f in five seconds beside a standing player", moved)
	}

	// Left far behind, a wild one is let go and a tame one is kept.
	g.player.Pos.X += 2 * creatureGone
	g.fauna.timer = 0
	g.fauna.step(g.world, g.player.Pos, frame)
	if n := len(g.fauna.all); n == 0 || !g.fauna.all[0].tame {
		t.Fatalf("far from both, %d mossbacks are left, the first tame: %v", n, n > 0 && g.fauna.all[0].tame)
	}
	for _, c := range g.fauna.all {
		if !c.tame && c.Pos.Distance(g.player.Pos) > creatureGone {
			t.Error("a wild mossback left far behind was kept")
		}
	}
}

func TestTameMossbacksAreNotCountedAmongTheWildOnes(t *testing.T) {
	f := &fauna{rng: rand.New(rand.NewPCG(1, 2))}
	for range creatureCount {
		f.all = append(f.all, beast{tame: true})
	}
	if n := f.count(render.Mossback); n != 0 {
		t.Errorf("%d of twelve tame mossbacks count toward the wild ones, wanted none", n)
	}
}

func TestATameMossbackIsSavedAsOne(t *testing.T) {
	path := filepath.Join(t.TempDir(), "world")
	g := savedGame(t, path)
	startNew(t, g, true)
	g.fauna.all = []beast{
		{Pos: g.player.Pos.Add(geometry.Vector3{X: 3}), tame: true, fed: 3},
		{Pos: g.player.Pos.Add(geometry.Vector3{X: -3}), fed: 2},
	}
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}
	back := savedGame(t, path)
	if err := press(back, engine.KeyEnter); err != nil {
		t.Fatal(err)
	}
	all := back.fauna.all
	if len(all) != 2 || !all[0].tame || all[0].fed != 3 || all[1].tame || all[1].fed != 2 {
		t.Errorf("the mossbacks came back as %+v, wanted one tame and one fed twice", all)
	}
}

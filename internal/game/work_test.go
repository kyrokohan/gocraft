package game

import (
	"math"
	"math/rand/v2"
	"path/filepath"
	"slices"
	"testing"

	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/world"
)

// The pot by the hearth on the test floor, 2 blocks from it, and the place for a
// knapping stone, 2 blocks from it the other way.
var (
	floorPot   = world.Cell{X: 8, Y: 6, Z: 10}
	floorStone = world.Cell{X: 4, Y: 6, Z: 10}
)

// atHome is a survival game on the test floor at noon, with the player's hearth and a
// pot by it, and one of the tribe living there with a job.
func atHome(t *testing.T, j job) (*Game, *person) {
	t.Helper()
	g := inSurvival(t)
	g.days = 0.2 // noon
	g.world.Set(floorHearth.X, floorHearth.Y, floorHearth.Z, world.Hearth)
	g.world.Set(floorPot.X, floorPot.Y, floorPot.Z, world.ClayPot)
	one := member(world.Cell{X: 300, Y: 70}, 4)
	one.job = j
	g.people = []person{one}
	return g, &g.people[0]
}

// inPot counts how many of an item a pot holds.
func inPot(p *world.PotStore, item world.Item) (n int) {
	for _, s := range p.Items {
		if s.Item == item {
			n += s.Count
		}
	}
	return n
}

// facingMember is atHome with no job, the one of the tribe standing still 2 blocks
// ahead of the player and facing them.
func facingMember(t *testing.T) (*Game, *person) {
	t.Helper()
	g, one := atHome(t, idle)
	one.Pos, one.timer = geometry.Vector3{X: 20.5, Y: 6, Z: 22.5}, 1000
	hold(t, g, 1)
	if !g.personAimed || &g.people[g.talkTo] != one {
		t.Fatal("the one of the tribe 2 blocks ahead is not aimed at")
	}
	return g, one
}

func TestShowingWoodOrBerriesToOneAtYourFireGivesThemAJob(t *testing.T) {
	g, one := facingMember(t)
	holding(g, world.Stick, 3)
	press(g, 'e')
	if one.job != keeper || g.items[g.selected].Count != 3 {
		t.Fatalf("shown sticks, one of the tribe has job %d, and the player has %d sticks", one.job, g.items[g.selected].Count)
	}
	holding(g, world.Berries, 2)
	press(g, 'e')
	if one.job != gatherer || g.items[g.selected].Count != 2 {
		t.Fatalf("shown berries, one of the tribe has job %d, and the player has %d berries", one.job, g.items[g.selected].Count)
	}
	holding(g, world.Cobble.Item(), 1)
	press(g, 'e')
	if one.job != gatherer || one.say != thanks {
		t.Errorf("shown cobble, one of the tribe has job %d and shows %+v", one.job, one.say)
	}

	one.job, one.follows = idle, true
	holding(g, world.Stick, 3)
	press(g, 'e')
	if one.job != idle {
		t.Error("one who still follows the player took a job")
	}
}

func TestAKeeperKeepsYourHearthBurningFromThePotsByItAndCooksTheMeat(t *testing.T) {
	g, _ := atHome(t, keeper)
	pot := g.pot(floorPot)
	pot.Items[0], pot.Items[1] = world.Stack{Item: world.Stick, Count: 10}, world.Stack{Item: world.Meat, Count: 3}
	far := world.Cell{X: 6, Y: 6, Z: 20} // 10 blocks from the hearth
	g.world.Set(far.X, far.Y, far.Z, world.ClayPot)
	g.pot(far).Items[0] = world.Stack{Item: world.Coal, Count: 5}

	hold(t, g, seconds(30))
	k := g.hearth(floorHearth)
	if k.Burn <= 0 || g.world.At(floorHearth.X, floorHearth.Y, floorHearth.Z) != world.HearthLit {
		t.Errorf("after 30 seconds the hearth has %.1f seconds to burn, and is %v", k.Burn, g.world.At(floorHearth.X, floorHearth.Y, floorHearth.Z))
	}
	if cooked := inPot(pot, world.CookedMeat); cooked != 3 || inPot(pot, world.Meat) != 0 {
		t.Errorf("after 30 seconds the pot holds %d cooked meat and %d raw", cooked, inPot(pot, world.Meat))
	}
	if inPot(pot, world.Stick) >= 10 || inPot(g.pot(far), world.Coal) != 5 {
		t.Errorf("the pot by the hearth has %d sticks, and the one 10 blocks off %d coal", inPot(pot, world.Stick), inPot(g.pot(far), world.Coal))
	}
}

func TestAGathererBringsBerriesAndWhatLiesAboutToAPotByYourFire(t *testing.T) {
	g, _ := atHome(t, gatherer)
	near, far := world.Cell{X: 14, Y: 6, Z: 10}, world.Cell{X: 36, Y: 6, Z: 10} // 8 and 30 blocks from the hearth
	g.world.Set(near.X, near.Y, near.Z, world.Bush)
	g.world.Set(far.X, far.Y, far.Z, world.Bush)
	g.drops = append(g.drops, drop{Drop: world.Drop{Stack: world.Stack{Item: world.Flint, Count: 1}, Pos: geometry.Vector3{X: 6.5, Y: 6, Z: 18.5}, Age: 10}})

	g.days = 1 - firstMorning // midnight
	hold(t, g, seconds(15))
	if g.world.At(near.X, near.Y, near.Z) != world.Bush || len(g.drops) != 1 {
		t.Fatal("a gatherer went out at night")
	}

	g.days = 0.2 // noon
	hold(t, g, seconds(60))
	pot := g.pot(floorPot)
	if inPot(pot, world.Berries) != 2 || inPot(pot, world.Flint) != 1 || len(g.drops) != 0 {
		t.Errorf("after a minute the pot by the fire holds %d berries and %d flint, and %d things lie about", inPot(pot, world.Berries), inPot(pot, world.Flint), len(g.drops))
	}
	if g.world.At(near.X, near.Y, near.Z) == world.Bush || g.world.At(far.X, far.Y, far.Z) != world.Bush {
		t.Error("the bush 8 blocks from the fire was not picked, or the one 30 blocks off was")
	}
}

func TestASaveKeepsEachOnesJobAndWhatTheyCarry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "world")
	g, one := atHome(t, gatherer)
	g.opts.SavePath = path
	one.carry[0], one.picks = world.Stack{Item: world.Berries, Count: 2}, 1
	keeps := member(world.Cell{X: 600, Y: 70}, 5)
	keeps.job = keeper
	guards := member(world.Cell{X: 900, Y: 70}, 6)
	guards.job, guards.carry[0] = sentry, world.Stack{Item: world.HardSpear, Count: 1, Wear: 7}
	g.people = append(g.people, keeps, guards)
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}

	them := ofTribe(savedGame(t, path))
	if len(them) != 3 {
		t.Fatalf("%d of the tribe came back, wanted 3", len(them))
	}
	if them[0].job != gatherer || them[0].carry[0] != (world.Stack{Item: world.Berries, Count: 2}) || them[1].job != keeper {
		t.Errorf("the first of the tribe came back with job %d carrying %+v, the second with job %d", them[0].job, them[0].carry, them[1].job)
	}
	if them[2].job != sentry || them[2].carry[0] != (world.Stack{Item: world.HardSpear, Count: 1, Wear: 7}) {
		t.Errorf("a guard came back with job %d carrying %+v", them[2].job, them[2].carry[0])
	}
}

func TestShowingFlintMakesAKnapperAndGivingASpearAGuard(t *testing.T) {
	g, one := facingMember(t)
	holding(g, world.Flint, 2)
	press(g, 'e')
	if one.job != knapper || g.items[g.selected].Count != 2 {
		t.Fatalf("shown flint, one of the tribe has job %d, and the player has %d flint", one.job, g.items[g.selected].Count)
	}
	holding(g, world.Spear, 1)
	press(g, 'e')
	if one.job != sentry || one.carry[0].Item != world.Spear || g.items[g.selected].Count != 0 {
		t.Fatalf("given a spear, one of the tribe has job %d and carries %+v, and the player has %d spears", one.job, one.carry[0], g.items[g.selected].Count)
	}
	holding(g, world.Stick, 3)
	press(g, 'e')
	if one.job != keeper || one.carry[0].Count != 0 || len(g.drops) != 1 || g.drops[0].Item != world.Spear {
		t.Errorf("shown sticks, a guard has job %d and carries %+v, and %d things lie about", one.job, one.carry[0], len(g.drops))
	}
}

func TestAKnapperMakesToolsAtTheKnappingStoneFromTheFlintInThePots(t *testing.T) {
	g, _ := atHome(t, knapper)
	pot := g.pot(floorPot)
	pot.Items[0], pot.Items[1], pot.Items[2] = world.Stack{Item: world.Flint, Count: 5}, world.Stack{Item: world.Stick, Count: 2}, world.Stack{Item: world.Cord, Count: 1}
	hold(t, g, seconds(30))
	if inPot(pot, world.Flint) != 5 {
		t.Fatal("a knapper made something with no knapping stone by the fire")
	}

	g.world.Set(floorStone.X, floorStone.Y, floorStone.Z, world.KnappingStone)
	hold(t, g, seconds(90))
	if axes, knives := inPot(pot, world.HandAxe), inPot(pot, world.FlintKnife); axes != 2 || knives != 1 || inPot(pot, world.Flint) != 0 {
		t.Errorf("after 90 seconds at the stone the pot holds %d hand axes, %d knives and %d flint", axes, knives, inPot(pot, world.Flint))
	}
}

// guarding is atHome with one of the tribe guarding the fire with a spear.
func guarding(t *testing.T) (*Game, *person) {
	t.Helper()
	g, one := atHome(t, sentry)
	one.carry[0] = world.Stack{Item: world.Spear, Count: 1}
	return g, one
}

func TestAGuardWatchesAtTheEdgeOfTheFiresLightAndFightsAHunterThatComes(t *testing.T) {
	g, one := guarding(t)
	hold(t, g, seconds(10))
	mid := cellMid(floorHearth.X, floorHearth.Y, floorHearth.Z)
	if far := math.Hypot(one.Pos.X-mid.X, one.Pos.Z-mid.Z); math.Abs(far-guardOut) > 1 || one.seated {
		t.Fatalf("a guard with nothing to fight stands %.1f blocks from the fire", far)
	}

	c := cat(6.5, 18.5) // 8 blocks from the fire
	c.rng = rand.New(rand.NewPCG(1, 2))
	g.fauna.all = []beast{c}
	hold(t, g, seconds(10))
	if c := g.fauna.all[0]; c.health >= breeds[render.Daggercat].health {
		t.Error("in 10 seconds the guard did not hurt the daggercat 8 blocks from the fire")
	}
}

func TestGuardsGoOutAfterAKindPaintedByYourFireAndNoOther(t *testing.T) {
	g, _ := guarding(t)
	g.fauna.all = []beast{{
		kind: render.Bugler, Pos: geometry.Vector3{X: 30.5, Y: 6, Z: 30.5}, // 31 blocks from the fire
		health: breeds[render.Bugler].health, rng: rand.New(rand.NewPCG(1, 2)),
	}}
	bugler := world.FirstBeast + world.Picture(render.Bugler)
	far := world.Side{Cell: world.Cell{X: 31, Y: 5, Z: 10}, Face: world.FaceYPos} // 25 blocks from the fire
	g.paintings[far] = world.Painting{Side: far, Picture: bugler, Paint: world.RedPaint}
	hold(t, g, seconds(15))
	if g.fauna.all[0].health < breeds[render.Bugler].health {
		t.Fatal("a guard went out after a bugler painted only 25 blocks from the fire")
	}

	near := world.Side{Cell: world.Cell{X: 10, Y: 5, Z: 14}, Face: world.FaceYPos}
	g.paintings[near] = world.Painting{Side: near, Picture: bugler, Paint: world.RedPaint}
	hold(t, g, seconds(30))
	hurt := len(g.fauna.all) == 1 && g.fauna.all[0].health < breeds[render.Bugler].health
	killed := len(g.fauna.all) == 0 && slices.ContainsFunc(g.drops, func(d drop) bool { return d.Item == world.Meat })
	if !hurt && !killed {
		t.Error("in 30 seconds the guard did not hurt the bugler painted by the fire")
	}
}

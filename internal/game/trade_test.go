package game

import (
	"path/filepath"
	"slices"
	"testing"

	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/world"
)

// facingPerson is a survival game on the test floor with a camp's fire 8 blocks ahead,
// one of its people standing still 2 blocks ahead of the player and facing them, and a
// thing in hand.
func facingPerson(t *testing.T, item world.Item, n int) (*Game, *person) {
	t.Helper()
	g := inSurvival(t)
	fire := world.Cell{X: 20, Y: 6, Z: 28}
	g.world.Set(fire.X, fire.Y, fire.Z, world.Fire)
	g.people = campPeople(g.world, fire)
	one := &g.people[0]
	one.Pos, one.Yaw, one.timer = geometry.Vector3{X: 20.5, Y: 6, Z: 22.5}, 0, 1000 // it stands
	for i := range g.people[1:] {
		g.people[1+i].Pos.Z += 4 // the others keep out of the way
	}
	holding(g, item, n)
	hold(t, g, 1)
	if !g.personAimed || &g.people[g.talkTo] != one {
		t.Fatalf("the person 2 blocks ahead is not aimed at")
	}
	return g, one
}

func TestEWithAThingTheyWantShowsAnOfferAndEAgainTrades(t *testing.T) {
	g, one := facingPerson(t, 0, 0)
	m := campMarket(g.world.Seed, one.fire)
	want := m.wants[0]
	holding(g, want, 2)

	press(g, 'e')
	give, n := m.offer(want)
	if one.say != (saying{item: give, count: n}) || n < 1 || worth[give]*n > worth[want] {
		t.Fatalf("shown %v, a person shows %+v, wanted %d of %v", want, one.say, n, give)
	}
	if g.items[g.selected].Count != 2 {
		t.Fatal("showing a thing gave it away")
	}

	press(g, 'e')
	if g.items[g.selected].Count != 1 || g.items.count([]world.Item{give}) != n {
		t.Errorf("after the trade the player has %d of %v and %d of %v", g.items[g.selected].Count, want, g.items.count([]world.Item{give}), give)
	}
	if g.trust[one.fire] != tradeTrust || one.say != thanks {
		t.Errorf("after a trade the camp's trust is %d and the person shows %+v", g.trust[one.fire], one.say)
	}
}

func TestAnOfferLapses(t *testing.T) {
	g, one := facingPerson(t, 0, 0)
	want := campMarket(g.world.Seed, one.fire).wants[0]
	holding(g, want, 2)
	press(g, 'e')
	hold(t, g, seconds(offerTime+1))
	press(g, 'e')
	if g.items[g.selected].Count != 2 || g.trust[one.fire] != 0 {
		t.Error("E traded on an offer made long before")
	}
}

func TestAThingTheyDoNotWantShowsWhatTheyWant(t *testing.T) {
	g, one := facingPerson(t, world.Cobble.Item(), 3)
	press(g, 'e')
	press(g, 'e')
	if one.say != (saying{item: one.want}) || g.items[g.selected].Count != 3 || g.trust[one.fire] != 0 {
		t.Errorf("shown cobble, a person shows %+v, the player has %d, the trust is %d", one.say, g.items[g.selected].Count, g.trust[one.fire])
	}
}

func TestACrouchedEGivesAGiftAndWinsMoreTrust(t *testing.T) {
	g, one := facingPerson(t, 0, 0)
	m := campMarket(g.world.Seed, one.fire)
	want := m.wants[0]
	holding(g, want, 2)
	g.player.Crouched = true

	press(g, 'e')
	give, _ := m.offer(want)
	if g.items[g.selected].Count != 1 || g.items.count([]world.Item{give}) != 0 {
		t.Errorf("after a gift the player has %d of %v and %d of %v", g.items[g.selected].Count, want, g.items.count([]world.Item{give}), give)
	}
	if g.trust[one.fire] != giftTrust || giftTrust <= tradeTrust || one.say != thanks {
		t.Errorf("after a gift the trust is %d and the person shows %+v", g.trust[one.fire], one.say)
	}
}

func TestKillingAHunterNearACampWinsItsTrust(t *testing.T) {
	g, one := facingPerson(t, world.Cobble.Item(), 1)
	fire := one.fire
	g.fauna.all = []beast{cat(20.5, 32.5)}
	g.wound(0, 1000, 1, false)
	if g.trust[fire] != helpTrust || helpTrust <= giftTrust {
		t.Errorf("a daggercat killed by the camp: its trust is %d", g.trust[fire])
	}

	g.trust[fire] = 0
	g.fauna.all = []beast{cat(20.5, 32.5+helpReach+5)}
	g.wound(0, 1000, 1, false)
	bugler := beast{kind: render.Bugler, Pos: geometry.Vector3{X: 20.5, Y: 6, Z: 32.5}, health: breeds[render.Bugler].health}
	g.fauna.all = []beast{bugler}
	g.wound(0, 1000, 1, false)
	if g.trust[fire] != 0 {
		t.Errorf("a daggercat killed far off and a bugler killed near won the camp's trust: %d", g.trust[fire])
	}
}

func TestEveryCampWantsSomethingPreciousAndGivesSomethingForAnythingItWants(t *testing.T) {
	precious := []world.Item{world.Amber, world.Tusk, world.CrystalShard}
	for seed := range int64(20) {
		for i := range 20 {
			m := campMarket(seed, world.Cell{X: 300 * i, Y: 70, Z: -500 * i})
			if len(m.wants) < 2 || len(m.goods) < 2 || !slices.ContainsFunc(m.wants, func(w world.Item) bool { return slices.Contains(precious, w) }) {
				t.Fatalf("a camp wants %v and gives %v", m.wants, m.goods)
			}
			for _, w := range m.wants {
				if slices.Contains(m.goods, w) {
					t.Errorf("a camp wants and gives %v", w)
				}
				if give, n := m.offer(w); n < 1 || !slices.Contains(m.goods, give) {
					t.Errorf("a camp gives %d of %v for %v", n, give, w)
				}
			}
		}
	}
	for _, w := range precious {
		for _, other := range []world.Item{world.Meat, world.Hide, world.Flint, world.Ochre} {
			if worth[w] <= worth[other] {
				t.Errorf("%v is worth no more than %v", w, other)
			}
		}
	}
}

func TestASaveKeepsEachCampsTrust(t *testing.T) {
	path := filepath.Join(t.TempDir(), "world")
	g := inSurvival(t)
	g.opts.SavePath = path
	g.trust[world.Cell{X: -120, Y: 70, Z: 44}] = 7
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}
	if back := savedGame(t, path); back.trust[world.Cell{X: -120, Y: 70, Z: 44}] != 7 {
		t.Errorf("the camp's trust came back as %v", back.trust)
	}
}

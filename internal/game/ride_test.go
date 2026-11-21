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

// mounted is a game on the wide plain whose player rides a tame grown bugler that stands
// at x = 0.5 and the z given, facing +z. The player got on with E.
func mounted(t *testing.T, z float64) (*Game, *beast) {
	t.Helper()
	return mountedOn(t, render.Bugler, z)
}

// mountedOn is mounted, on a beast of the kind given: the player got on from behind it.
func mountedOn(t *testing.T, kind render.Kind, z float64) (*Game, *beast) {
	t.Helper()
	back, _ := render.Span(kind, false)
	g := among(t, z+back-1.4, raised(kind, 0.5, z, false))
	g.world = wide()
	hold(t, g, 1, 'e')
	m := &g.fauna.all[0]
	if !m.ridden {
		t.Fatalf("E on a tame grown %s just ahead did not get the player on it", kind)
	}
	return g, m
}

// onGround reports whether the player stands on the plain.
func onGround(g *Game) bool {
	return g.player.OnGround && math.Abs(g.player.Pos.Y-6) < 0.01
}

// onBack reports whether the player sits on the back of a beast: over it, as high as its
// back is.
func onBack(g *Game, m *beast) bool {
	b := breeds[m.kind]
	return math.Hypot(g.player.Pos.X-m.Pos.X, g.player.Pos.Z-m.Pos.Z) < 1 &&
		math.Abs(g.player.Pos.Y-(m.Pos.Y+b.back-render.SeatHeight)) < 0.05
}

func TestEMountsATameGrownBuglerAndCGetsDown(t *testing.T) {
	g, m := mounted(t, 10.5)
	if !onBack(g, m) {
		t.Fatalf("the player got on a bugler at %v and is at %v", m.Pos, g.player.Pos)
	}
	// With no key held it stands: it does not follow or wander, and the player stays on.
	at := m.Pos
	hold(t, g, seconds(3))
	if moved := m.Pos.Distance(at); moved > 0.1 || !onBack(g, m) {
		t.Errorf("a ridden bugler with no key held went %.1f blocks; the player is on its back: %v", moved, onBack(g, m))
	}
	// Seen from behind, the player's body sits on it.
	if !g.body().Seated || g.body().Crouched {
		t.Error("the body of a player who rides is not drawn seated")
	}
	// C gets the player down, standing, on the ground beside it.
	hold(t, g, 1, 'c')
	hold(t, g, seconds(1))
	if m.ridden || g.player.Crouched {
		t.Fatalf("C did not get the player down: riding %v, crouched %v", m.ridden, g.player.Crouched)
	}
	if far := math.Hypot(g.player.Pos.X-m.Pos.X, g.player.Pos.Z-m.Pos.Z); far < 0.9 || far > 3 || !onGround(g) || !g.standsIn(g.player.Pos) {
		t.Errorf("the player got down %.1f blocks from the bugler at height %.2f, clear of blocks: %v", far, g.player.Pos.Y, g.standsIn(g.player.Pos))
	}
	if g.health != maxHealth {
		t.Errorf("getting down cost the player %v points", maxHealth-g.health)
	}
	// Off its back it is a tame beast again, that follows.
	g.player.Pos.Z -= 12
	hold(t, g, seconds(2))
	if !m.walking {
		t.Error("a bugler the player got down from does not follow a player twelve blocks off")
	}
}

func TestARiddenBuglerGoesWhereTheKeysSay(t *testing.T) {
	g, m := mounted(t, -60.5)
	b := breeds[render.Bugler]

	// W held: it walks off at the player's walk, and gathers speed to its run of 9.
	from := m.Pos.Z
	hold(t, g, seconds(0.5), 'w')
	if went := m.Pos.Z - from; went < 0.4*walkSpeed || went > 0.5*(walkSpeed+b.ride)/2 {
		t.Errorf("in its first half second with W held a ridden bugler went %.2f blocks: wanted about a walk's", went)
	}
	hold(t, g, seconds(2), 'w')
	from = m.Pos.Z
	hold(t, g, seconds(1), 'w')
	if went := m.Pos.Z - from; b.ride != 9 || math.Abs(went-9) > 0.4 || math.Abs(m.Pos.X-0.5) > 0.1 {
		t.Errorf("a bugler ridden with W held for three seconds went %.1f blocks in the last of them, %.1f aside: wanted 9 ahead", went, m.Pos.X-0.5)
	}
	if !onBack(g, m) {
		t.Fatalf("at a run the player is at %v and the bugler at %v", g.player.Pos, m.Pos)
	}
	// Let go, it stops. D turns it and the player together, and W takes it the new way.
	hold(t, g, seconds(0.5))
	from = m.Pos.Z
	hold(t, g, seconds(0.5))
	if went := m.Pos.Z - from; went > 0.1 {
		t.Errorf("a ridden bugler let go went on %.1f blocks", went)
	}
	hold(t, g, seconds(math.Pi/2/turnSpeed), 'd')
	if math.Abs(math.Remainder(m.Yaw-math.Pi/2, 2*math.Pi)) > 0.1 || math.Abs(math.Remainder(g.player.Yaw-m.Yaw, 2*math.Pi)) > 0.01 {
		t.Errorf("D held for a quarter turn left the bugler facing %.2f and the player %.2f", m.Yaw, g.player.Yaw)
	}
	fromX := m.Pos.X
	hold(t, g, seconds(1), 'w')
	if went := m.Pos.X - fromX; went < walkSpeed {
		t.Errorf("turned to the right, a ridden bugler went %.1f blocks to the right in a second", went)
	}
	// The arrow turns it too, and S backs it.
	hold(t, g, seconds(math.Pi/2/turnSpeed), engine.KeyLeft)
	from = m.Pos.Z
	hold(t, g, seconds(1), 's')
	if went := from - m.Pos.Z; went < 0.3*walkSpeed || went > walkSpeed {
		t.Errorf("S backed a ridden bugler %.1f blocks in a second", went)
	}
	if !m.ridden || !onBack(g, m) {
		t.Error("the player fell off a bugler that went where the keys said")
	}
}

func TestARiddenBuglerHopsAStepAndJumpsTwoBlocks(t *testing.T) {
	for _, tc := range []struct {
		name  string
		high  int
		keys  []engine.Key
		gets  bool // over the wall
		wants string
	}{
		{"a step, by itself", 1, []engine.Key{'w'}, true, "a ridden bugler did not hop a step of one block"},
		{"two high, with no jump", 2, []engine.Key{'w'}, false, "a ridden bugler walked over a wall two high"},
		{"two high, with Space", 2, []engine.Key{'w', ' '}, true, "a ridden bugler did not jump a wall two high"},
		{"three high, with Space", 3, []engine.Key{'w', ' '}, false, "a ridden bugler jumped a wall three high"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, m := mounted(t, 10.5)
			for x := -6; x <= 6; x++ {
				for y := 6; y < 6+tc.high; y++ {
					g.world.Set(x, y, 14, world.Stone)
				}
			}
			hold(t, g, seconds(3), tc.keys...)
			if over := m.Pos.Z > 15; over != tc.gets {
				t.Errorf("%s: it is at z %.1f", tc.wants, m.Pos.Z)
			}
			if !m.ridden || !onBack(g, m) || m.health != breeds[render.Bugler].health {
				t.Errorf("the player is on its back: %v; the bugler has %v health", onBack(g, m), m.health)
			}
		})
	}
}

func TestAWildOrYoungOrLimpingBeastCannotBeRidden(t *testing.T) {
	limping := raised(render.Bugler, 0.5, 2.5, false)
	limping.limping, limping.health = true, 1
	for name, c := range map[string]beast{
		"a wild bugler":         giant(render.Bugler, 0.5, 2.5),
		"a young bugler":        raised(render.Bugler, 0.5, 1.8, true),
		"a bugler that limps":   limping,
		"a tame grown mossback": raised(render.Mossback, 0.5, 2.5, false),
	} {
		g := among(t, 0.5, c)
		hold(t, g, 2, 'e')
		if g.fauna.all[0].ridden || !onGround(g) {
			t.Errorf("E got the player on %s", name)
		}
	}
	// One that loses all its health while it is ridden throws the player off, and limps.
	g, m := mounted(t, 10.5)
	m.hit(g.player.Pos, 1000)
	hold(t, g, seconds(1))
	if m.ridden || !m.limping || !onGround(g) {
		t.Errorf("a ridden bugler that lost all its health is ridden: %v, limping: %v, the player at height %.2f", m.ridden, m.limping, g.player.Pos.Y)
	}
}

func TestARiderGetsDownOnlyWhereThereIsRoom(t *testing.T) {
	// A bugler in a pen of stone just its size: there is nowhere to stand.
	g, m := mounted(t, 10)
	m.Pos.X = 0
	for x := -3; x <= 2; x++ {
		for z := 5; z <= 14; z++ {
			for y := 6; y <= 10; y++ {
				if (x == -1 || x == 0) && (z == 9 || z == 10) {
					continue
				}
				g.world.Set(x, y, z, world.Stone)
			}
		}
	}
	hold(t, g, 1)
	hold(t, g, 1, 'c')
	if !m.ridden || !onBack(g, m) {
		t.Errorf("a player got down where there was no room: riding %v, at %v", m.ridden, g.player.Pos)
	}
	// A wall on its left: the player gets down on its right.
	g, m = mounted(t, 10.5)
	for z := 5; z <= 16; z++ {
		for y := 6; y <= 9; y++ {
			g.world.Set(-1, y, z, world.Stone)
		}
	}
	hold(t, g, 1, 'c')
	hold(t, g, seconds(1))
	if m.ridden || g.player.Pos.X < m.Pos.X || !g.standsIn(g.player.Pos) {
		t.Errorf("by a wall on the left the player got down at x %.2f by a bugler at %.2f: riding %v, clear %v", g.player.Pos.X, m.Pos.X, m.ridden, g.standsIn(g.player.Pos))
	}
}

func TestARiderThrowsNothing(t *testing.T) {
	g, _ := mounted(t, 10.5)
	holding(g, world.Spear, 1)
	hold(t, g, 2, 'r')
	if len(g.missiles) != 0 || carried(g, world.Spear) != 1 {
		t.Errorf("a player who rides threw a spear: %d in the air", len(g.missiles))
	}
}

func TestASaveHoldsARider(t *testing.T) {
	path := filepath.Join(t.TempDir(), "world")
	g := savedGame(t, path)
	startNew(t, g, true)
	c := raised(render.Bugler, g.player.Pos.X, g.player.Pos.Z+2, false)
	c.Pos.Y = g.player.Pos.Y
	g.fauna.all = []beast{c}
	hold(t, g, 1, 'e')
	if !g.fauna.all[0].ridden {
		t.Fatal("E did not get the player on a bugler")
	}
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}
	back := savedGame(t, path)
	back.Play()
	hold(t, back, 2)
	if m := back.fauna.mount(); m == nil || !onBack(back, m) {
		t.Errorf("a save of a player on a bugler opened with the player at %v off its back", back.player.Pos)
	}
}

func TestARiddenShieldbrowWalksAtThreeAndNothingKnocksItBack(t *testing.T) {
	g, m := mountedOn(t, render.Shieldbrow, -60.5)
	hold(t, g, seconds(2), 'w')
	from := m.Pos.Z
	hold(t, g, seconds(1), 'w')
	if went := m.Pos.Z - from; math.Abs(went-3) > 0.2 || !onBack(g, m) {
		t.Errorf("a shieldbrow ridden with W held for three seconds went %.1f blocks in the last of them: wanted 3", went)
	}
	hold(t, g, seconds(0.5))
	at := m.Pos
	m.hit(at.Add(geometry.Vector3{Z: 2}), 20)
	hold(t, g, seconds(1))
	if moved := m.Pos.Distance(at); moved > 0.05 || !m.ridden {
		t.Errorf("a ridden shieldbrow hit hard went %.2f blocks; it is ridden: %v", moved, m.ridden)
	}
}

func TestARiddenShieldbrowChargesForTwoSecondsAndThenRestsForFour(t *testing.T) {
	g, m := mountedOn(t, render.Shieldbrow, -60.5)
	from := m.Pos.Z
	hold(t, g, 1, ' ')
	hold(t, g, seconds(1)-1)
	if went := m.Pos.Z - from; math.Abs(went-8) > 0.6 {
		t.Errorf("in the first second of a charge a ridden shieldbrow went %.1f blocks: wanted 8", went)
	}
	hold(t, g, seconds(1.2))
	from = m.Pos.Z
	hold(t, g, seconds(0.5))
	if went := m.Pos.Z - from; went > 0.1 || m.ire > 0 {
		t.Errorf("a charge of a ridden shieldbrow went on past two seconds: %.1f blocks more", went)
	}
	// From the end of a charge it is four seconds before Space charges again.
	since := 0.7 // seconds since the charge ended
	for _, tc := range []struct {
		after float64 // seconds since the charge ended
		again bool
	}{{0.9, false}, {2.5, false}, {3.7, false}, {4.3, true}} {
		hold(t, g, seconds(tc.after-since))
		hold(t, g, 1, ' ')
		since = tc.after + frame
		if again := m.ire > 0; again != tc.again {
			t.Errorf("Space %.1f s after a charge ended charged: %v", tc.after, again)
		}
	}
	if !m.ridden || !onBack(g, m) || g.health != maxHealth {
		t.Errorf("the rider of a charge is on its back: %v, with %v points lost", onBack(g, m), maxHealth-g.health)
	}
}

func TestARiddenShieldbrowsChargeThrowsAHunterBackAndSparesTheDread(t *testing.T) {
	g, _ := mountedOn(t, render.Shieldbrow, 10.5)
	g.days = midnight
	g.fauna.all = append(g.fauna.all, claw(0.5, 18.5))
	hold(t, g, 1, ' ')
	hold(t, g, seconds(1.5))
	m, q := &g.fauna.all[0], only(g.fauna.all, render.Quickclaw)
	b := breeds[render.Shieldbrow]
	if len(q) != 1 || q[0].health != breeds[render.Quickclaw].health-b.rams || m.ire != 0 {
		t.Fatalf("a quickclaw in the way of a ridden shieldbrow's charge was hit for %v points; the charge goes on: %v", breeds[render.Quickclaw].health-q[0].health, m.ire > 0)
	}
	if far := math.Hypot(q[0].Pos.X-m.Pos.X, q[0].Pos.Z-m.Pos.Z); far < 3 || g.health != maxHealth {
		t.Errorf("a quickclaw a charge met is %.1f blocks from the shieldbrow, and the rider lost %v points", far, maxHealth-g.health)
	}

	g, _ = mountedOn(t, render.Shieldbrow, 10.5)
	g.fauna.all = append(g.fauna.all, dread(0.5, 20.5))
	hold(t, g, 1, ' ')
	hold(t, g, seconds(2.5))
	if d := only(g.fauna.all, render.Dread); len(d) != 1 || d[0].health != breeds[render.Dread].health {
		t.Errorf("a ridden shieldbrow rammed the dread")
	}
}

func TestHuntersAChargeWouldKillKeepOffARiddenShieldbrow(t *testing.T) {
	for _, tc := range []struct {
		kind render.Kind
		away bool
	}{{render.Shieldbrow, true}, {render.Bugler, false}} {
		g, _ := mountedOn(t, tc.kind, 10.5)
		g.days = midnight
		hurt, whole := claw(-2.5, 28.5), claw(3.5, 28.5)
		hurt.health = breeds[render.Shieldbrow].rams - 1
		g.fauna.all = append(g.fauna.all, hurt, whole)
		closest := map[float64]float64{-2.5: math.Inf(1), 3.5: math.Inf(1)}
		for range seconds(8) {
			hold(t, g, 1)
			for _, q := range only(g.fauna.all, render.Quickclaw) {
				d := math.Hypot(q.Pos.X-g.player.Pos.X, q.Pos.Z-g.player.Pos.Z)
				if q.health < breeds[render.Quickclaw].health {
					closest[-2.5] = min(closest[-2.5], d)
				} else {
					closest[3.5] = min(closest[3.5], d)
				}
			}
		}
		if kept := closest[-2.5] > ramKeep-1; kept != tc.away {
			t.Errorf("on a %s a hurt quickclaw came within %.1f blocks of the rider", tc.kind, closest[-2.5])
		}
		if closest[3.5] > 3 {
			t.Errorf("on a %s a whole quickclaw came no nearer than %.1f blocks", tc.kind, closest[3.5])
		}
	}
}

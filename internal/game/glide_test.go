package game

import (
	"math"
	"testing"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/world"
)

// ledge is a game on the wide plain whose player stands at the edge of a ledge so many
// blocks high, at z = -80, facing off it, with a galewing in the air over the ledge:
// grown and the player's, or young, or wild. A glide from twenty blocks up comes down
// on the plain.
func ledge(t *testing.T, high int, tame, young bool) *Game {
	t.Helper()
	g := among(t, -80.5)
	g.world = wide()
	for z := -92; z < -80; z++ {
		for x := -3; x <= 3; x++ {
			for y := 6; y < 6+high; y++ {
				g.world.Set(x, y, z, world.Stone)
			}
		}
	}
	g.player.Pos.Y = float64(6 + high)
	c := giant(render.Galewing, 10.5, -80.5)
	c.tame, c.young, c.aloft = tame, young, true
	c.Pos.Y = g.player.Pos.Y + 14
	g.fauna.all = []beast{c}
	hold(t, g, 1)
	return g
}

// off walks the player off the edge it stands at, and lets go of W once it is off. It
// reports whether a galewing has taken it two seconds later.
func off(t *testing.T, g *Game) bool {
	t.Helper()
	for range seconds(1) {
		if !g.player.OnGround {
			break
		}
		hold(t, g, 1, 'w')
	}
	for range seconds(2) {
		if g.fauna.glider() != nil {
			return true
		}
		hold(t, g, 1)
	}
	return g.fauna.glider() != nil
}

func TestAGalewingOfYoursKeepsOverYou(t *testing.T) {
	for _, tame := range []bool{true, false} {
		g := among(t, -60.5)
		g.world = wide()
		c := giant(render.Galewing, 0.5, -60.5)
		c.tame, c.aloft, c.Pos.Y = tame, true, 20
		g.fauna.all = []beast{c}
		most := 0.0
		for range 50 {
			hold(t, g, seconds(0.5), 'w')
			gale := g.fauna.all[0]
			most = max(most, math.Hypot(gale.Pos.X-g.player.Pos.X, gale.Pos.Z-g.player.Pos.Z))
		}
		if kept := most <= 30; kept != tame {
			t.Errorf("a galewing (the player's: %v) was as far as %.0f blocks from a player who walked for 25 seconds", tame, most)
		}
	}
}

func TestAFallOfSixBlocksWithAGalewingOfYoursNearIsAGlide(t *testing.T) {
	for _, tc := range []struct {
		name               string
		high               int
		tame, young, glide bool
	}{
		{"six blocks", 6, true, false, true},
		{"three blocks", 3, true, false, false},
		{"six blocks, with a wild galewing", 6, false, false, false},
		{"six blocks, with a young galewing", 6, true, true, false},
	} {
		if glided := off(t, ledge(t, tc.high, tc.tame, tc.young)); glided != tc.glide {
			t.Errorf("%s: glided %v", tc.name, glided)
		}
	}
}

func TestAGlideGoesFiveBlocksAheadForOneDown(t *testing.T) {
	g := ledge(t, 20, true, false)
	if !off(t, g) {
		t.Fatal("a galewing of the player's did not take it off a ledge twenty blocks high")
	}
	m := g.fauna.glider()
	if p := g.player.Pos; math.Hypot(m.Pos.X-p.X, m.Pos.Z-p.Z) > 0.01 || m.Pos.Y < p.Y+playerHeight {
		t.Errorf("the galewing is at %v, and not over the player at %v", m.Pos, p)
	}
	from, frames := g.player.Pos, 0
	for ; g.player.Pos.Z < from.Z+40 && frames < seconds(10); frames++ {
		hold(t, g, 1)
	}
	if lost := from.Y - g.player.Pos.Y; math.Abs(lost-8) > 0.2 {
		t.Errorf("a glide of forty blocks ahead lost %.1f blocks of height: wanted 8", lost)
	}
	if took := float64(frames) * frame; math.Abs(took-40.0/9) > 0.1 {
		t.Errorf("a glide of forty blocks took %.2f seconds: wanted %.2f, at 9 blocks a second", took, 40.0/9)
	}
	// W dives, and A and D turn.
	y := g.player.Pos.Y
	if hold(t, g, seconds(1), 'w'); y-g.player.Pos.Y < 4 {
		t.Errorf("a second of a dive lost %.1f blocks of height", y-g.player.Pos.Y)
	}
	yaw := g.player.Yaw
	if hold(t, g, seconds(0.5), 'd'); math.Abs(math.Remainder(g.player.Yaw-yaw, 2*math.Pi)-turnSpeed/2) > 0.05 {
		t.Errorf("half a second of D in a glide turned the player %.2f radians", g.player.Yaw-yaw)
	}
	if g.fauna.glider() == nil {
		t.Error("the glide ended in the air")
	}
}

func TestAGlideSetsThePlayerDownUnhurt(t *testing.T) {
	g := ledge(t, 20, true, false)
	if !off(t, g) {
		t.Fatal("a galewing of the player's did not take it off a ledge twenty blocks high")
	}
	for range seconds(20) {
		if hold(t, g, 1); g.fauna.glider() == nil {
			break
		}
	}
	if !onGround(g) || g.health != maxHealth {
		t.Errorf("a glide from twenty blocks up ended at %v, on the ground %v, with %v health", g.player.Pos, g.player.OnGround, g.health)
	}
	if hold(t, g, seconds(1)); g.fauna.glider() != nil || g.fauna.all[0].Pos.Y < g.player.Pos.Y+playerHeight {
		t.Error("the galewing did not let go of a player on the ground and take to the air")
	}
}

func TestCLetsGoOfAGalewing(t *testing.T) {
	g := ledge(t, 20, true, false)
	if !off(t, g) {
		t.Fatal("a galewing of the player's did not take it off a ledge twenty blocks high")
	}
	hold(t, g, seconds(0.5))
	if hold(t, g, 1, 'c'); g.fauna.glider() != nil || g.player.Crouched {
		t.Fatal("C did not let go of the galewing, or crouched")
	}
	for range seconds(3) {
		if hold(t, g, 1); g.fauna.glider() != nil {
			t.Fatal("a galewing took a player that let go of it again before it was down")
		}
	}
	if !onGround(g) || g.health == maxHealth {
		t.Errorf("a player that let go of a galewing some twenty blocks up is at %v with %v health", g.player.Pos, g.health)
	}
}

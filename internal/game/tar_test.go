package game

import (
	"math"
	"testing"

	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/world"
)

// tarPool is a game in survival on the test floor, with a pool of tar three deep in it
// from 14 to 26 each way: its top is level with the floor, and its middle is at 20, 20.
func tarPool(t *testing.T) *Game {
	t.Helper()
	g := inSurvival(t)
	for z := 14; z <= 26; z++ {
		for x := 14; x <= 26; x++ {
			for y := 3; y <= 5; y++ {
				g.world.Set(x, y, z, world.Tar)
			}
		}
	}
	return g
}

func TestInTarAPlayerCrawlsAndSinksAndItsAirGoesWhenItsHeadIsUnder(t *testing.T) {
	g := tarPool(t)
	hold(t, g, seconds(1)) // it stood on the tar, which is no solid thing
	p := &g.player
	if !inTar(g.world, p.Pos) || p.OnGround {
		t.Fatalf("a player who stood on tar is at height %.2f, in it: %v, on the ground: %v", p.Pos.Y, inTar(g.world, p.Pos), p.OnGround)
	}
	// It crawls: a fifth of a walk.
	from := p.Pos
	hold(t, g, seconds(2), 'w')
	if went, want := math.Hypot(p.Pos.X-from.X, p.Pos.Z-from.Z), 2*walkSpeed/5; math.Abs(went-want) > 0.15 {
		t.Errorf("two seconds of walking in tar took the player %.2f blocks, wanted the %.2f of a fifth of a walk", went, want)
	}
	// It sinks, slowly, whatever it does: a jump does not lift it.
	from = p.Pos
	hold(t, g, seconds(5), ' ')
	if sank := from.Y - p.Pos.Y; math.Abs(sank-5*tarSink) > 0.1 {
		t.Errorf("five seconds in tar, jumping, sank the player %.2f blocks, wanted %.2f", sank, 5*tarSink)
	}
	if g.health != maxHealth || g.air != maxAir {
		t.Errorf("with its head out of the tar the player has lost %v points and %.1f s of air", maxHealth-g.health, maxAir-g.air)
	}
	// In about twenty seconds from the top its head is under, and then its air goes.
	stand(g, geometry.Vector3{X: 20.5, Y: 6, Z: 20.5})
	under := 0.0
	for i := range seconds(40) {
		hold(t, g, 1)
		if g.air < maxAir {
			under = float64(i) * frame
			break
		}
	}
	if under < 15 || under > 25 {
		t.Errorf("the head of a player who stood still in tar was under after %.0f s, wanted about twenty", under)
	}
	hold(t, g, seconds(maxAir+3))
	if g.air != 0 || g.health == maxHealth {
		t.Errorf("%v seconds under tar left %.1f s of air and took %v points", maxAir+3, g.air, maxHealth-g.health)
	}
	drawn(t, g) // with the eye in it

	// A fall into tar does not hurt, and in creative one flies over it.
	g = tarPool(t)
	g.player.Pos.Y, g.peak = 16, 16
	hold(t, g, seconds(3))
	if g.health != maxHealth || !inTar(g.world, g.player.Pos) {
		t.Errorf("a fall of ten blocks into tar took %v points", maxHealth-g.health)
	}
}

func TestAPlayerWhoGetsToTheEdgeOfTheTarClimbsOut(t *testing.T) {
	g := tarPool(t)
	hold(t, g, seconds(4)) // it has sunk a little, in the middle
	g.player.Yaw = yawOf(geometry.Vector3{Z: 1})
	out := -1.0
	for i := range seconds(30) {
		hold(t, g, 1, 'w')
		if !inTar(g.world, g.player.Pos) && g.player.OnGround && g.player.Pos.Z > 27 {
			out = float64(i) * frame
			break
		}
	}
	if out < 0 || math.Abs(g.player.Pos.Y-6) > 0.01 {
		t.Fatalf("thirty seconds of walking to the edge left the player at %+v, in tar: %v", g.player.Pos, inTar(g.world, g.player.Pos))
	}
	if out < 5 || out > 14 {
		t.Errorf("it took %.1f s to crawl six and a half blocks to the edge and climb out, wanted about eight", out)
	}
	if g.health != maxHealth {
		t.Errorf("the player who got out lost %v points", maxHealth-g.health)
	}
}

func TestABeastInTarIsStuckForGoodAndDiesThere(t *testing.T) {
	for _, kind := range []render.Kind{render.Bugler, render.Shieldbrow, render.Dread, render.Nipper} {
		g := tarPool(t)
		g.player.Pos = geometry.Vector3{X: 35.5, Y: 6, Z: 35.5}
		g.days = 0.2
		one := giant(kind, 20.5, 8.5) // on the floor, five blocks from the tar, facing it
		one.walking, one.timer = true, math.Inf(1)
		g.fauna.all, g.fauna.timer = []beast{one}, math.Inf(1)
		c := &g.fauna.all[0]
		hold(t, g, seconds(12))
		if !inTar(g.world, c.Pos) || c.Pos.Z < 13 {
			t.Fatalf("a %s that walked at the tar is at %+v, in it: %v", kind, c.Pos, inTar(g.world, c.Pos))
		}
		at, health := c.Pos, c.health
		hold(t, g, seconds(8))
		if moved := math.Hypot(c.Pos.X-at.X, c.Pos.Z-at.Z); moved > 0.2 || c.Pos.Y > at.Y {
			t.Errorf("a %s in tar moved %.2f blocks in eight seconds, and rose by %.2f: wanted it held", kind, moved, c.Pos.Y-at.Y)
		}
		if c.health >= health {
			t.Errorf("a %s in tar has %v health after eight seconds, of %v", kind, c.health, health)
		}
		hold(t, g, seconds(tarDeath))
		if len(g.fauna.all) != 0 && !g.fauna.all[0].dead {
			t.Errorf("a %s is alive after %v seconds in tar, with %v health", kind, tarDeath+20, g.fauna.all[0].health)
		}
	}
	// One that flies goes over it.
	g := tarPool(t)
	g.player.Pos = geometry.Vector3{X: 35.5, Y: 6, Z: 35.5}
	wing := giant(render.Rattlewing, 20.5, 20.5)
	wing.Pos.Y = 7.6
	g.fauna.all, g.fauna.timer = []beast{wing}, math.Inf(1)
	hold(t, g, seconds(20))
	if c := g.fauna.all[0]; c.dead || inTar(g.world, c.Pos) || c.health != breeds[render.Rattlewing].health {
		t.Errorf("a rattlewing over tar is dead: %v, in it: %v", c.dead, inTar(g.world, c.Pos))
	}
}

func TestTarIsTakenByHandAndBindsATorch(t *testing.T) {
	g := digging(t, world.Tar)
	hold(t, g, seconds(2), 'q')
	if g.world.At(20, 5, 21) != world.Air {
		t.Fatal("two seconds of a hand did not take the tar")
	}
	n := g.items.count([]world.Item{world.Tar.Item()})
	for _, d := range g.drops {
		if d.Stack.Item == world.Tar.Item() {
			n += d.Stack.Count
		}
	}
	if n != 1 {
		t.Errorf("a block of tar left %d of tar", n)
	}

	g = inSurvival(t)
	g.items[0] = world.Stack{Item: world.Stick, Count: 1}
	g.items[1] = world.Stack{Item: world.Tar.Item(), Count: 1}
	r, ok := recipeFor(g, world.UnlitTorch)
	if !ok || !g.craft(r) || g.items.count([]world.Item{world.UnlitTorch}) != 2 || g.items.count([]world.Item{world.Tar.Item(), world.Stick}) != 0 {
		t.Errorf("torches could not be made of a stick and tar: the player carries %v", g.items)
	}
}

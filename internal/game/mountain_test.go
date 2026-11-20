package game

import (
	"math"
	"testing"

	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/world"
)

// pool puts a pool of lava three blocks across in the plain of a game, with its middle
// at x, z and its top at the level of the ground.
func pool(g *Game, x, z int) {
	for dx := -1; dx <= 1; dx++ {
		for dz := -1; dz <= 1; dz++ {
			g.world.Set(x+dx, 4, z+dz, world.Stone)
			g.world.Set(x+dx, 5, z+dz, world.Lava)
		}
	}
}

func TestLavaBurnsThreeTimesAsHardAsFire(t *testing.T) {
	g := among(t, 0.5)
	pool(g, 0, 0)
	g.player.Pos.Y = 5.2 // in it
	hold(t, g, seconds(1))
	if lost := maxHealth - g.health; lost < lavaHurt-1.5 || lost > lavaHurt+1.5 {
		t.Errorf("a second in lava cost %v points, wanted about %v", lost, lavaHurt)
	}
	if lavaHurt != 3*fireHurt {
		t.Errorf("lava takes %v points a second and fire %v, wanted three times", lavaHurt, fireHurt)
	}
	if !inWater(g.world, g.player.Pos) || g.player.Pos.Y > 5.5 {
		t.Errorf("the player in lava is at height %.2f and does not sink and swim as in water", g.player.Pos.Y)
	}
	drawn(t, g) // with the eye in it

	c := onFloor(t) // creative
	c.world = plain()
	c.player.Pos = geometry.Vector3{X: 0.5, Y: 5.2, Z: 0.5}
	pool(c, 0, 0)
	hold(t, c, seconds(2))
	if c.health != maxHealth {
		t.Errorf("in creative lava cost %v points", maxHealth-c.health)
	}
}

func TestATorchIsLitAtLavaAndWhatFallsInItIsBurntUp(t *testing.T) {
	g := among(t, 0.5)
	pool(g, 0, 4)
	g.items[0] = world.Stack{Item: world.UnlitTorch, Count: 2}
	g.player.Pitch = -0.5 // down at the pool ahead
	hold(t, g, 1)
	if !g.aimed || g.target.Block != world.Lava {
		t.Fatalf("aimed %v at %+v, wanted the lava", g.aimed, g.target)
	}
	hold(t, g, 1, 'e')
	if g.items[0] != (world.Stack{Item: world.Torch.Item(), Count: 2}) {
		t.Errorf("E with unlit torches on lava left %+v in hand, wanted them lit", g.items[0])
	}
	hold(t, g, seconds(1), 'q')
	if g.world.At(0, 5, 4) != world.Lava {
		t.Error("lava was broken by hand")
	}

	g.drops = append(g.drops, drop{Drop: world.Drop{Stack: world.Stack{Item: world.Meat, Count: 3}, Pos: geometry.Vector3{X: 0.5, Y: 8, Z: 4.5}}})
	hold(t, g, seconds(3))
	if len(g.drops) != 0 || g.items.count([]world.Item{world.Meat}) != 0 {
		t.Errorf("meat dropped into lava: %d things lie there and the player carries %d", len(g.drops), g.items.count([]world.Item{world.Meat}))
	}
}

func TestABeastKeepsOutOfLava(t *testing.T) {
	g := among(t, -30.5, giant(render.Bugler, 0.5, 2.5))
	for x := -8; x <= 8; x++ { // a ditch of lava across its way
		for z := 8; z <= 10; z++ {
			g.world.Set(x, 4, z, world.Stone)
			g.world.Set(x, 5, z, world.Lava)
		}
	}
	for range seconds(12) {
		g.fauna.all[0].walking, g.fauna.all[0].timer, g.fauna.all[0].Yaw = true, math.Inf(1), 0
		hold(t, g, 1)
	}
	if c := g.fauna.all[0]; c.Pos.Z > 8 || c.Pos.Y < 5.9 {
		t.Errorf("a bugler walked at lava and is at %+v, wanted it stopped at the edge", c.Pos)
	}
}

func TestTheSmokeOfTheMountainIsSeenFromWhereTheWorldBeginsAndRisesFromItsCrater(t *testing.T) {
	g, err := New(Options{Seed: 7})
	if err != nil {
		t.Fatal(err)
	}
	g.Play()
	g.begin(false)
	crater, far, ok := g.world.Mountain(g.player.Pos.X, g.player.Pos.Z)
	if !ok || far < 400 || far > 900 {
		t.Fatalf("the nearest mountain to where seed 7 begins is %.0f blocks off, %v", far, ok)
	}

	// The sky toward it has its smoke in it from the first frame: a column, darker than
	// the sky on either side of it at the same height.
	g.player.Flying = true
	g.player.Pos.Y += 40 // over the trees and the hills
	g.clouds = render.NewClouds(99)
	g.player.Yaw = yawOf(geometry.Vector3{X: float64(crater.X) - g.player.Pos.X, Z: float64(crater.Z) - g.player.Pos.Z})
	g.player.Pitch = 0.3
	_, fb := drawn(t, g)
	band := func(x0 int) (sum int) {
		for y := 15; y < 21; y++ { // low in the sky, where the smoke is thick, and over the land
			for x := x0; x < x0+4; x++ {
				r, gr, b := fb.At(x, y).Background.RGB()
				sum += int(r) + int(gr) + int(b)
			}
		}
		return sum
	}
	if left, middle, right := band(20), band(48), band(76); middle >= left*9/10 || middle >= right*9/10 {
		t.Errorf("the sky toward the mountain sums to %d, and %d and %d to either side: wanted its smoke to darken the middle", middle, left, right)
	}

	// At its foot the crater smokes.
	at := geometry.Vector3{X: float64(crater.X) + 50.5, Z: float64(crater.Z) + 0.5}
	g.world.Follow(at.X, at.Z)
	g.world.LoadAll()
	at.Y = float64(g.world.Ground(int(at.X), int(at.Z))) + 1
	g.player.Pos, g.player.Flying = at, false
	// The beasts that are found places round a player on the ash: none is put on it. One
	// may walk onto it after, so they are looked at as they come.
	g.fauna.all, g.fauna.timer = nil, 0
	for range 5 {
		hold(t, g, 1)
		g.fauna.timer = 0
		for _, c := range g.fauna.all {
			if world.BiomeAt(7, int(math.Floor(c.Pos.X)), int(math.Floor(c.Pos.Z))) == world.Ashlands {
				t.Fatalf("a %s was put in the ashlands", c.kind)
			}
		}
	}
	hold(t, g, seconds(5))
	over := 0
	for _, b := range g.bits {
		if b.smoke && b.pos.Y > float64(crater.Y)-1 && math.Hypot(b.pos.X-float64(crater.X), b.pos.Z-float64(crater.Z)) < 12 {
			over++
		}
	}
	if over < 6 {
		t.Errorf("%d puffs of smoke stand over the crater after five seconds at the mountain's foot", over)
	}
}

func TestAshSoundsAsSandAndDarkRockAsStone(t *testing.T) {
	if materialOf(world.Ash) != materialOf(world.Sand) || materialOf(world.Basalt) != materialOf(world.Stone) {
		t.Error("ash does not sound as sand, or dark rock as stone")
	}
}

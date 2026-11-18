package game

import (
	"math"

	"glonk.dev/geometry"

	"github.com/kyrokohan/gocraft/internal/world"
)

// The player's body in survival. Health and food are points; air is seconds of breath.
// These are first guesses, to be tuned in play.
const (
	maxHealth = 20.0
	maxFood   = float64(world.MaxFood)
	maxAir    = 15.0

	safeFall   = 3.0  // a fall of up to this many blocks does not hurt; each block more costs a point
	drownHurt  = 2.0  // points a second, once the air is gone
	breathBack = 5.0  // seconds of air won back each second out of water
	hungerTime = 30.0 // seconds for a point of food to go
	mendTime   = 4.0  // seconds between a point healed, or lost to hunger
	mendFood   = 18.0 // food at or above which the player heals
	hurtFlash  = 0.3  // seconds the view shows red when the player is hurt
)

// live runs the player's body in survival for dt seconds, after the player has moved:
// falls and drowning hurt, food goes down, and a full stomach heals.
func (g *Game) live(dt float64) {
	p := &g.player
	switch {
	case inWater(g.world, p.Pos):
		g.peak = p.Pos.Y // water breaks a fall
	case p.OnGround:
		if fall := g.peak - p.Pos.Y; fall > safeFall {
			g.hurt(math.Round(fall - safeFall)) // the peak is a frame late, so a hair short
		}
		g.peak = p.Pos.Y
	default:
		g.peak = max(g.peak, p.Pos.Y)
	}

	eye := p.Camera().Position
	if g.world.At(int(math.Floor(eye.X)), int(math.Floor(eye.Y)), int(math.Floor(eye.Z))) == world.Water {
		if g.air -= dt; g.air < 0 {
			g.air = 0
			g.hurt(drownHurt * dt)
		}
	} else {
		g.air = min(g.air+breathBack*dt, maxAir)
	}

	for g.hunger += dt; g.hunger >= hungerTime; g.hunger -= hungerTime {
		g.food = max(g.food-1, 0)
	}
	if g.mend += dt; g.mend >= mendTime {
		g.mend = 0
		switch {
		case g.food >= mendFood:
			g.health = min(g.health+1, maxHealth)
		case g.food == 0 && g.health > 1:
			g.hurt(1) // hunger hurts, and leaves the player one point: it cannot kill
		}
	}
}

// hurt takes points of health from the player. At none the player faints.
func (g *Game) hurt(points float64) {
	if points <= 0 {
		return
	}
	g.health = max(g.health-points, 0)
	g.flash = hurtFlash
	if g.health == 0 {
		g.show(fainted)
	}
}

// renew gives the player a whole body again: full health, food and air.
func (g *Game) renew() {
	g.health, g.food, g.air = maxHealth, maxFood, maxAir
	g.hunger, g.mend, g.flash = 0, 0, 0
	g.peak = g.player.Pos.Y
}

// wake stands the player where a new player stands, whole again and with everything it
// carried: the world is brought in round that place first.
func (g *Game) wake() {
	pos, ok := g.bedPlace()
	if !ok {
		g.world.Follow(0, 0)
		g.world.LoadAll()
		pos = g.world.Spawn()
	}
	g.player = Player{Pos: pos, Yaw: g.player.Yaw}
	g.renew()
	g.aimed, g.preyed, g.digging = false, false, 0
	g.gloams = nil // they lose the player
}

// eat eats one of what is in the slot in use, if it is food and the player has room for
// it, and reports whether it did.
func (g *Game) eat() bool {
	s := g.items[g.selected]
	if !g.survival || s.Item.Food() == 0 || g.food >= maxFood {
		return false
	}
	g.food = min(g.food+float64(s.Item.Food()), maxFood)
	g.items.takeOne(g.selected)
	return true
}

// leave puts down what a block broken in survival leaves, in the middle of its cell.
func (g *Game) leave(x, y, z int, b world.Block) {
	item, n := b.DropAt(x, y, z)
	if !g.survival || n == 0 {
		return
	}
	g.drops = append(g.drops, drop{Drop: world.Drop{
		Stack: world.Stack{Item: item, Count: n},
		Pos:   geometry.Vector3{X: float64(x) + 0.5, Y: float64(y) + 0.5 - dropSize/2, Z: float64(z) + 0.5},
	}})
}

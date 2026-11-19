package game

import (
	"math"

	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/sound"
	"github.com/kyrokohan/skyspear/internal/world"
)

// What a broken block leaves lies on the ground as a small block of its own, which falls
// as a body does and is picked up when the player comes near.
const (
	dropSize  = 0.25  // blocks across
	pickReach = 1.5   // how near the middle of the player a drop is picked up
	pickWait  = 0.5   // seconds before a drop can be picked up, so that it is seen to fall
	dropLife  = 300.0 // seconds a drop lies before it is gone
)

// drop is a stack lying on the ground, and how fast it falls.
type drop struct {
	world.Drop
	velY float64
}

// stepDrops moves what lies on the ground on by dt seconds: it falls where its chunk is
// loaded, is picked up if the player is near and has room for it, and is gone when old.
func (g *Game) stepDrops(dt float64) {
	middle := g.player.Pos.Add(geometry.Vector3{Y: playerHeight / 2})
	kept := g.drops[:0]
	for _, d := range g.drops {
		if d.Age += dt; d.Age > dropLife {
			continue
		}
		if g.world.Loaded(int(math.Floor(d.Pos.X)), int(math.Floor(d.Pos.Z))) {
			d.velY = max(d.velY-gravity*dt, -fallSpeed)
			if inWater(g.world, d.Pos) {
				d.velY = max(d.velY, -sinkSpeed)
			}
			if stopped := travel(g.world, &d.Pos, dropSize, dropSize, geometry.Vector3{Y: d.velY * dt}); stopped[1] {
				d.velY = 0
			}
		}
		at := d.Pos.Add(geometry.Vector3{Y: dropSize / 2})
		if d.Age >= pickWait && at.Sub(middle).Length() <= pickReach {
			if d.Count = g.items.add(d.Stack); d.Count == 0 {
				g.ears.PlayNear(sound.Pickup, 0.6)
				continue
			}
		}
		kept = append(kept, d)
	}
	g.drops = kept
}

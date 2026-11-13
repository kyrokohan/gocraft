package game

import (
	"glonk.dev/engine"

	"github.com/kyrokohan/gocraft/internal/world"
)

// hotbar is the kinds of block the player can place, on the keys 1 to 9.
var hotbar = [9]world.Block{
	world.Grass, world.Dirt, world.Stone, world.Sand, world.Log,
	world.Leaves, world.Planks, world.Brick, world.Glass,
}

// reach is how far away the player can break and place blocks.
const reach = 5.0

// aim finds the block in the middle of the player's view, if one is in reach.
func (g *Game) aim() {
	cam := g.player.Camera()
	g.target, g.aimed = g.world.Target(cam.Position, cam.Look(), reach)
}

// build handles the keys that pick, break and place blocks. A held key repeats, so
// holding Q digs on and holding E builds on.
func (g *Game) build(in *engine.Input) {
	for i := range hotbar {
		if in.Pressed(engine.Key('1' + rune(i))) {
			g.selected = i
		}
	}
	if in.Pressed(engine.KeyTab) {
		g.selected = (g.selected + 1) % len(hotbar)
	}

	g.aim()
	if !g.aimed {
		return
	}
	t := g.target
	switch {
	case keyDown(in.Pressed, 'q'):
		if t.Block.Info().Fixed {
			return
		}
		g.world.Set(t.X, t.Y, t.Z, world.Air)
	case keyDown(in.Pressed, 'e'):
		nx, ny, nz := t.Face.Normal()
		x, y, z := t.X+nx, t.Y+ny, t.Z+nz
		if b := g.world.At(x, y, z); (b != world.Air && b != world.Water) || g.player.overlaps(x, y, z) {
			return
		}
		g.world.Set(x, y, z, hotbar[g.selected])
	default:
		return
	}
	g.aim() // the world changed, so what is aimed at may have too
}

// Package game is the gocraft game itself: what engine.Run drives.
package game

import (
	"math"

	"glonk.dev/engine"

	"github.com/kyrokohan/gocraft/internal/assets"
	"github.com/kyrokohan/gocraft/internal/render"
	"github.com/kyrokohan/gocraft/internal/world"
)

const (
	fieldOfView = math.Pi / 3 // from the top of the view to the bottom
	eyeHeight   = 1.6         // above the player's feet
)

// Game is the whole game. It implements engine.Game.
type Game struct {
	set    *assets.Set
	rend   *render.Renderer
	world  *world.World
	player Player
}

// New returns a game in the world a seed makes, ready for engine.Run.
func New(seed int64) (*Game, error) {
	set, err := assets.Load()
	if err != nil {
		return nil, err
	}
	w := world.Generate(seed)
	return &Game{
		set:    set,
		rend:   render.New(set),
		world:  w,
		player: Player{Pos: w.Spawn()},
	}, nil
}

// Update advances the game by dt seconds.
func (g *Game) Update(dt float64, in *engine.Input) error {
	if in.Pressed(engine.KeyEscape) {
		return engine.ErrQuit
	}
	g.player.Step(g.world, readControls(in), dt)
	return nil
}

// Draw renders the game into fb.
func (g *Game) Draw(fb *engine.FrameBuffer) {
	c := fb.Canvas()
	g.rend.Draw(c, g.world, g.player.Camera())
	c.Compose()
}

// Package game is the gocraft game itself: what engine.Run drives.
package game

import (
	"glonk.dev/engine"
)

// Game is the whole game. It implements engine.Game.
type Game struct{}

// New returns a game ready for engine.Run.
func New() *Game {
	return &Game{}
}

// Update advances the game by dt seconds.
func (g *Game) Update(dt float64, in *engine.Input) error {
	if in.Pressed(engine.KeyEscape) {
		return engine.ErrQuit
	}
	return nil
}

// Draw renders the game into fb.
func (g *Game) Draw(fb *engine.FrameBuffer) {
	c := fb.Canvas()
	c.FillRect(0, 0, c.Width(), c.Height(), engine.RGB(120, 170, 235))
	c.Compose()
	fb.DrawText(1, 0, "gocraft", engine.Color{})
}

package game

import (
	"math"

	"github.com/kyrokohan/gocraft/internal/render"
)

// swingTime is how many seconds a swing of the hand takes.
const swingTime = 0.25

// inHand returns what the player holds as it is drawn: swinging if it has just been used,
// and rising and falling with the player's steps.
func (g *Game) inHand() render.Held {
	h := render.Held{
		Item:  g.items[g.selected].Item,
		Bob:   math.Abs(math.Sin(math.Pi * g.walked / stepLength)),
		Light: g.sky.Brightness(g.lightAt(g.player.Camera().Position)),
	}
	if g.swing > 0 {
		h.Swing = 1 - g.swing/swingTime
	}
	return h
}

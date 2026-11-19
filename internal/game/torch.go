package game

import (
	"math"

	"github.com/kyrokohan/skyspear/internal/world"
)

// torchTime is how many seconds a torch burns in the hand, in survival.
const torchTime = 300.0

// carry burns the torch in the hand for dt seconds, and has it light the place the
// player is in, which hunters keep out of. In survival a torch that has burned for
// torchTime is a stick again, and the next of them in the hand begins to burn. A torch
// that is set down is a block, and burns for ever.
func (g *Game) carry(dt float64) {
	s := &g.items[g.selected]
	lit := s.Count > 0 && s.Item == world.Torch.Item()
	if lit && g.survival {
		if g.torch += dt; g.torch >= torchTime {
			g.torch = 0
			g.items.takeOne(g.selected)
			lit = s.Count > 0
			stick := world.Stack{Item: world.Stick, Count: 1}
			if g.items.add(stick) > 0 {
				g.putDown(stick)
			}
			g.tell("your torch burned out")
		}
	}
	var glow uint8
	if lit {
		glow = world.Torch.Info().Glow
	}
	at := g.player.head()
	g.world.Carry(int(math.Floor(at.X)), int(math.Floor(at.Y)), int(math.Floor(at.Z)), glow)
}

package game

import (
	"math"
	"strconv"
	"strings"

	"glonk.dev/geometry"
	"glonk.dev/mesh"

	"github.com/kyrokohan/skyspear/internal/sound"
	"github.com/kyrokohan/skyspear/internal/world"
)

// Things are made from what the player carries, by a list of what can be made: a few
// things by hand, and more standing near a knapping stone. There are no patterns to lay out.
// What is cooked, hardened or baked, a hearth makes in its own slots (stores.go).

// station is where a recipe is made.
type station uint8

const (
	byHand station = iota
	atBench
)

// stationReach is how near a knapping stone the player stands to use it, in blocks from the
// middle of the player to the block.
const stationReach = 4

// need is a number of things a recipe uses: of one item, or of any of a few.
type need struct {
	name  string // what the player is shown
	items []world.Item
	n     int
}

// of is a need of n of one item.
func of(item world.Item, n int) need {
	return need{name: item.Name(), items: []world.Item{item}, n: n}
}

var (
	logs = []world.Item{
		world.Log.Item(), world.PineLog.Item(), world.PaleLog.Item(),
		world.FernTrunk.Item(), world.RedwoodLog.Item(), world.StumpPalm.Item(),
	}
	anyLog = need{name: "log", items: logs, n: 1}
)

// recipe is a thing that can be made, how many of it, from what, and where.
type recipe struct {
	makes world.Item
	n     int
	needs []need
	at    station
}

// recipes is everything that can be made, in the order the list shows it: what is made
// by hand, of what is gathered, and then what is knapped from flint at a knapping stone.
var recipes = func() []recipe {
	cobble := world.Cobble.Item()
	return []recipe{
		{world.Stick, 4, []need{anyLog}, byHand},
		{world.Cord, 1, []need{of(world.Fiber, 3)}, byHand},
		{world.KnappingStone.Item(), 1, []need{of(cobble, 4)}, byHand},
		{world.DiggingStick, 1, []need{of(world.Stick, 2)}, byHand},
		{world.Club, 1, []need{anyLog}, byHand},
		{world.Spear, 1, []need{of(world.Stick, 3)}, byHand},
		{world.FurBed.Item(), 1, []need{of(world.Hide, 3), of(world.Moss, 6)}, byHand},

		// Fire: a ring of stones to keep it in, a drill to make it with, and torches to
		// carry it. A pot is shaped of clay and then baked in a hearth.
		{world.Hearth.Item(), 1, []need{of(cobble, 8)}, byHand},
		{world.FireDrill, 1, []need{of(world.Stick, 1), anyLog, of(world.Cord, 1)}, byHand},
		{world.UnlitTorch, 2, []need{of(world.Stick, 1), of(world.Resin, 1)}, byHand},
		{world.SoftPot, 1, []need{of(world.Clay.Item(), 4)}, byHand},

		// Flint alone makes a hand axe. A pick and a knife are bound to a handle with cord.
		{world.HandAxe, 1, []need{of(world.Flint, 2)}, atBench},
		{world.FlintKnife, 1, []need{of(world.Flint, 1), of(world.Stick, 1), of(world.Cord, 1)}, atBench},
		{world.FlintPick, 1, []need{of(world.Flint, 3), of(world.Stick, 2), of(world.Cord, 1)}, atBench},
	}
}()

// near reports whether a block of a kind is within stationReach of the player.
func (g *Game) near(b world.Block) bool {
	middle := g.player.Pos.Add(geometry.Vector3{Y: playerHeight / 2})
	x0, y0, z0 := int(math.Floor(middle.X)), int(math.Floor(middle.Y)), int(math.Floor(middle.Z))
	for y := y0 - stationReach; y <= y0+stationReach; y++ {
		for z := z0 - stationReach; z <= z0+stationReach; z++ {
			for x := x0 - stationReach; x <= x0+stationReach; x++ {
				if g.world.At(x, y, z) != b {
					continue
				}
				cell := geometry.Vector3{X: float64(x) + 0.5, Y: float64(y) + 0.5, Z: float64(z) + 0.5}
				if cell.Sub(middle).Length() <= stationReach {
					return true
				}
			}
		}
	}
	return false
}

// makeable returns the recipes that can be made where the player stands, by hand and at
// any workbench near, whether there is enough to make them or not.
func (g *Game) makeable() []recipe {
	at := [2]bool{byHand: true, atBench: g.near(world.KnappingStone)}
	var here []recipe
	for _, r := range recipes {
		if at[r.at] {
			here = append(here, r)
		}
	}
	return here
}

// affords reports whether the player carries enough to make a recipe.
func (g *Game) affords(r recipe) bool {
	for _, n := range r.needs {
		if g.items.count(n.items) < n.n {
			return false
		}
	}
	return true
}

// craft makes a recipe, if the player carries enough for it: what it needs is used up,
// and what it makes goes into the inventory, or in front of the player if there is no
// room for it.
func (g *Game) craft(r recipe) bool {
	if !g.affords(r) {
		return false
	}
	for _, n := range r.needs {
		g.items.use(n.items, n.n)
	}
	made := world.Stack{Item: r.makes, Count: r.n}
	g.ears.PlayNear(sound.Pickup, 0.6)
	if left := g.items.add(made); left > 0 {
		made.Count = left
		g.putDown(made)
	}
	return true
}

// putDown puts a stack down in front of the player, to be picked up again.
func (g *Game) putDown(s world.Stack) {
	ahead := mesh.Camera{Yaw: g.player.Yaw}.Forward()
	at := g.player.Pos.Add(ahead).Add(geometry.Vector3{Y: 1})
	g.drops = append(g.drops, drop{Drop: world.Drop{Stack: s, Pos: at}})
}

// label is a recipe as the list shows it: what it makes, and what from.
func (r recipe) label() (makes, from string) {
	makes = r.makes.Name()
	if r.n > 1 {
		makes += " ×" + strconv.Itoa(r.n)
	}
	var parts []string
	for _, n := range r.needs {
		part := n.name
		if n.n > 1 {
			part += " ×" + strconv.Itoa(n.n)
		}
		parts = append(parts, part)
	}
	return makes, strings.Join(parts, ", ")
}

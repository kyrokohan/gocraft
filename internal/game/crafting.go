package game

import (
	"math"
	"strconv"
	"strings"

	"glonk.dev/geometry"
	"glonk.dev/mesh"

	"github.com/kyrokohan/gocraft/internal/world"
)

// Things are made from what the player carries, by a list of what can be made: a few
// things by hand, and more standing near a workbench. There are no patterns to lay out.
// What is cooked, a kiln makes in its own slots (stores.go).

// station is where a recipe is made.
type station uint8

const (
	byHand station = iota
	atBench
)

// stationReach is how near a workbench the player stands to use it, in blocks from the
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
	logs   = []world.Item{world.Log.Item(), world.PineLog.Item(), world.PaleLog.Item()}
	anyLog = need{name: "log", items: logs, n: 1}
)

// recipe is a thing that can be made, how many of it, from what, and where.
type recipe struct {
	makes world.Item
	n     int
	needs []need
	at    station
}

// tools makes the recipes of the three tools and the blade of a tier, from what their
// heads are made of and sticks.
func tools(head world.Item, pick, axe, shovel, blade world.Item) []recipe {
	return []recipe{
		{pick, 1, []need{of(head, 3), of(world.Stick, 2)}, atBench},
		{axe, 1, []need{of(head, 3), of(world.Stick, 2)}, atBench},
		{shovel, 1, []need{of(head, 1), of(world.Stick, 2)}, atBench},
		{blade, 1, []need{of(head, 2), of(world.Stick, 1)}, atBench},
	}
}

var recipes = func() []recipe {
	planks, cobble := world.Planks.Item(), world.Cobble.Item()
	all := []recipe{
		{planks, 4, []need{anyLog}, byHand},
		{world.Stick, 4, []need{of(planks, 2)}, byHand},
		{world.Workbench.Item(), 1, []need{of(planks, 4)}, byHand},
		{world.Torch.Item(), 4, []need{of(world.Stick, 1), of(world.Coal, 1)}, byHand},
	}
	all = append(all, tools(planks, world.WoodPick, world.WoodAxe, world.WoodShovel, world.WoodBlade)...)
	all = append(all, tools(cobble, world.StonePick, world.StoneAxe, world.StoneShovel, world.StoneBlade)...)
	all = append(all, tools(world.CopperBar, world.CopperPick, world.CopperAxe, world.CopperShovel, world.CopperBlade)...)
	all = append(all, tools(world.IronBar, world.IronPick, world.IronAxe, world.IronShovel, world.IronBlade)...)
	return append(all,
		recipe{world.Kiln.Item(), 1, []need{of(cobble, 8)}, atBench},
		recipe{world.Crate.Item(), 1, []need{of(planks, 8)}, atBench},
		recipe{world.Bedroll.Item(), 1, []need{of(world.Moss, 6), of(planks, 3)}, atBench},
	)
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
	at := [2]bool{byHand: true, atBench: g.near(world.Workbench)}
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

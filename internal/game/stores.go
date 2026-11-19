package game

import (
	"cmp"
	"fmt"
	"slices"

	"glonk.dev/engine"
	"glonk.dev/geometry"
	"glonk.dev/widget"

	"github.com/kyrokohan/skyspear/internal/world"
)

// A pot and a hearth are stores: blocks with slots of their own, which E opens in the bag
// above the player's things, in survival. A pot keeps what is put in it. A hearth cooks
// what is put in it, a piece at a time, as long as it has fuel: an ore to a bar, sand to
// glass, clay to brick, berries to a loaf and moss to a cake. Every hearth cooks wherever it
// is while the world is played, with nobody to watch it.

// cookTime is how many seconds a hearth takes to cook one piece.
const cookTime = 6.0

// cook is what a hearth makes: one of makes from n of from.
type cook struct {
	from  world.Item
	n     int
	makes world.Item
}

var cooks = []cook{
	{world.Sand.Item(), 1, world.Glass.Item()},
	{world.Clay.Item(), 1, world.Brick.Item()},
	{world.Berries, 3, world.BerryLoaf},
	{world.Moss, 3, world.MossCake},
}

// burnTime is how many seconds a piece of an item burns in a hearth: 0 for what is not
// fuel.
func burnTime(item world.Item) float64 {
	switch item {
	case world.Stick:
		return cookTime / 2
	case world.Planks.Item():
		return 1.5 * cookTime
	case world.Log.Item(), world.PineLog.Item(), world.PaleLog.Item():
		return 4 * cookTime
	case world.Coal:
		return 8 * cookTime
	}
	return 0
}

// cooksFrom reports whether a hearth cooks anything from an item.
func cooksFrom(item world.Item) bool {
	return slices.ContainsFunc(cooks, func(c cook) bool { return c.from == item })
}

// cookIn returns what a hearth can cook from what is in it: there is enough of it, and
// room for what it makes.
func cookIn(k *world.HearthStore) (cook, bool) {
	for _, c := range cooks {
		if k.In.Item != c.from {
			continue
		}
		room := k.Out.Count == 0 || k.Out.Item == c.makes && k.Out.Count < c.makes.StackSize()
		return c, k.In.Count >= c.n && room
	}
	return cook{}, false
}

// stepHearth moves a hearth on by dt seconds. While there is something it can cook, it burns
// a piece of fuel after another, and what it has cooked stays cooked when the fuel runs
// out; taking away what it was cooking starts the next one afresh.
func stepHearth(k *world.HearthStore, dt float64) {
	for dt > 0 {
		step := min(dt, 0.25) // so that a long step lights the next piece of fuel in time
		dt -= step
		c, ok := cookIn(k)
		if !ok {
			k.Cooked = 0
		}
		if k.Burn == 0 && ok && burnTime(k.Fuel.Item) > 0 {
			k.Burn, k.Burns = burnTime(k.Fuel.Item), burnTime(k.Fuel.Item)
			takeFrom(&k.Fuel, 1)
		}
		if k.Burn > 0 && ok {
			if k.Cooked += step; k.Cooked >= cookTime {
				k.Cooked = 0
				takeFrom(&k.In, c.n)
				k.Out = world.Stack{Item: c.makes, Count: k.Out.Count + 1}
			}
		}
		k.Burn = max(k.Burn-step, 0)
	}
}

// takeFrom takes n from a stack that has them.
func takeFrom(s *world.Stack, n int) {
	if s.Count -= n; s.Count <= 0 {
		*s = world.Stack{}
	}
}

// stepHearths moves every hearth on by dt seconds.
func (g *Game) stepHearths(dt float64) {
	for _, k := range g.hearths {
		stepHearth(k, dt)
	}
}

// pot returns the store of the pot at a cell, which is empty if nothing was ever put
// in it.
func (g *Game) pot(c world.Cell) *world.PotStore {
	s, ok := g.pots[c]
	if !ok {
		s = &world.PotStore{Cell: c, Items: make([]world.Stack, world.PotSlots)}
		g.pots[c] = s
	}
	return s
}

// hearth returns the store of the hearth at a cell, which is cold and empty if nothing was
// ever put in it.
func (g *Game) hearth(c world.Cell) *world.HearthStore {
	s, ok := g.hearths[c]
	if !ok {
		s = &world.HearthStore{Cell: c}
		g.hearths[c] = s
	}
	return s
}

// openStore opens the bag with the slots of a pot or a hearth over the player's own, and
// the keys on its first slot.
func (g *Game) openStore(b world.Block, c world.Cell) {
	g.openBag()
	g.opened, g.at, g.cursor = b, c, 0
}

// storeRows is how many rows at the top of the bag are the slots of the block opened: a
// pot's two, a hearth's one, or none.
func (g *Game) storeRows() int {
	switch g.opened {
	case world.ClayPot:
		return world.PotSlots / hotbarSlots
	case world.Hearth:
		return 1
	}
	return 0
}

// The columns of a hearth's row: what it cooks, its fuel, and, past a column that shows how
// far it is, what it has made.
const (
	hearthIn   = 0
	hearthFuel = 1
	hearthOut  = 3
)

// storeSlot returns the slot of the block opened at a row and a column of its rows, or
// nil where a hearth's row has none.
func (g *Game) storeSlot(row, col int) *world.Stack {
	if g.opened == world.ClayPot {
		return &g.pot(g.at).Items[row*hotbarSlots+col]
	}
	k := g.hearth(g.at)
	switch col {
	case hearthIn:
		return &k.In
	case hearthFuel:
		return &k.Fuel
	case hearthOut:
		return &k.Out
	}
	return nil
}

// takes reports whether what is in hand can be put down in a slot: a hearth takes only
// what it cooks where it cooks, only fuel where it burns, and nothing where it makes.
func (g *Game) takes(s *world.Stack) bool {
	if g.opened != world.Hearth || g.hand.Count == 0 {
		return true
	}
	k := g.hearth(g.at)
	switch s {
	case &k.In:
		return cooksFrom(g.hand.Item)
	case &k.Fuel:
		return burnTime(g.hand.Item) > 0
	}
	return false
}

// spill puts down what a pot or a hearth held when it is broken, and forgets it.
func (g *Game) spill(c world.Cell) {
	var held []world.Stack
	if s, ok := g.pots[c]; ok {
		held = s.Items
	}
	if k, ok := g.hearths[c]; ok {
		held = []world.Stack{k.In, k.Fuel, k.Out}
	}
	delete(g.pots, c)
	delete(g.hearths, c)
	for _, s := range held {
		if s.Count > 0 && g.survival {
			g.drops = append(g.drops, drop{Drop: world.Drop{Stack: s, Pos: cellMiddle(c)}})
		}
	}
}

// loadStores takes the pots, the hearths and the bed from a save.
func (g *Game) loadStores(s world.Meta) {
	g.pots, g.hearths, g.bed = map[world.Cell]*world.PotStore{}, map[world.Cell]*world.HearthStore{}, s.Bed
	for _, c := range s.Pots {
		items := make([]world.Stack, world.PotSlots)
		copy(items, c.Items)
		g.pots[c.Cell] = &world.PotStore{Cell: c.Cell, Items: items}
	}
	for _, k := range s.Hearths {
		g.hearths[k.Cell] = &k
	}
}

// savedStores returns the pots and the hearths as a save holds them: those that hold
// anything, in the order of their cells, so that a save of the same world is the same.
func (g *Game) savedStores() (pots []world.PotStore, hearths []world.HearthStore) {
	for _, c := range g.pots {
		if slices.ContainsFunc(c.Items, func(s world.Stack) bool { return s.Count > 0 }) {
			pots = append(pots, *c)
		}
	}
	for _, k := range g.hearths {
		if k.In.Count > 0 || k.Fuel.Count > 0 || k.Out.Count > 0 || k.Burn > 0 {
			hearths = append(hearths, *k)
		}
	}
	slices.SortFunc(pots, func(a, b world.PotStore) int { return compareCells(a.Cell, b.Cell) })
	slices.SortFunc(hearths, func(a, b world.HearthStore) int { return compareCells(a.Cell, b.Cell) })
	return pots, hearths
}

// cellMiddle is where a thing put down in a cell lies to begin with, before it falls.
func cellMiddle(c world.Cell) geometry.Vector3 {
	return geometry.Vector3{X: float64(c.X) + 0.5, Y: float64(c.Y) + 0.5 - dropSize/2, Z: float64(c.Z) + 0.5}
}

func compareCells(a, b world.Cell) int {
	return cmp.Or(cmp.Compare(a.X, b.X), cmp.Compare(a.Y, b.Y), cmp.Compare(a.Z, b.Z))
}

// useBlock does what E does to a block aimed at, when that is not to build against it:
// it opens a pot or a hearth, in survival, and lies down on a bedroll. It reports whether
// it did.
func (g *Game) useBlock(t world.Hit) bool {
	c := world.Cell{X: t.X, Y: t.Y, Z: t.Z}
	switch {
	case t.Block == world.FurBed:
		g.lieDown(c)
	case (t.Block == world.ClayPot || t.Block == world.Hearth) && g.survival:
		g.openStore(t.Block, c)
	default:
		return false
	}
	return true
}

// hearthHot is the look of the fire of a hearth and of what it has cooked.
var hearthHot = engine.Cell{Foreground: engine.RGB(244, 140, 52), Background: hudGround}

// drawHearth draws a hearth's row of the bag with its top left corner at (x, y): what it
// cooks and its fuel; then an arrow that fills as a piece cooks, over the fire, which
// goes down as a piece of fuel burns; then what it has made, and what it does.
func (g *Game) drawHearth(fb *engine.FrameBuffer, c *engine.Canvas, x, y int, k *world.HearthStore) {
	g.drawRow(fb, c, x, y, []world.Stack{k.In, k.Fuel})
	g.drawRow(fb, c, x+hearthOut*slotPitch, y, []world.Stack{k.Out})

	mid, w := x+(hearthOut-1)*slotPitch+1, slotPitch-1
	for row := y + 1; row < y+barHeight-1; row++ {
		for i := range w {
			fb.Set(mid+i, row, hudFill)
		}
	}
	bar := func(row int, part float64, full, empty rune) {
		on, off := hearthHot, hudDim
		on.Character, off.Character = full, empty
		widget.Bar{Value: part, Full: on, Empty: off}.Draw(fb, mid, row, w)
	}
	bar(y+1, k.Cooked/cookTime, '━', '─')
	arrow := hudDim
	if k.Burn > 0 {
		arrow = hearthHot
	}
	arrow.Character = '▶'
	fb.Set(mid+w-1, y+1, arrow)
	fire := 0.0
	if k.Burns > 0 {
		fire = k.Burn / k.Burns
	}
	bar(y+3, fire, '▲', '·')

	lines := [3]string{"cooks ore, sand, clay,", "berries and moss", "burns wood or coal"}
	if made, ok := cookIn(k); ok {
		lines[0], lines[1] = "makes "+made.makes.Name(), ""
	} else if k.In.Count > 0 && cooksFrom(k.In.Item) {
		lines[0], lines[1] = "take out what it made", "to make more"
		for _, ck := range cooks {
			if ck.from == k.In.Item && k.In.Count < ck.n {
				lines[0], lines[1] = fmt.Sprintf("needs %d %s", ck.n, ck.from.Name()), ""
			}
		}
	}
	if k.Burn > 0 {
		lines[2] = "burning"
	}
	tx := x + (hearthOut+1)*slotPitch + 2
	for i, line := range lines {
		widget.Label{Text: line, Style: hudText}.Draw(fb, tx, y+1+i, barWidth-(tx-x)-1)
	}
}

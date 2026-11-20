package game

import (
	"cmp"
	"math"
	"slices"

	"glonk.dev/engine"
	"glonk.dev/geometry"
	"glonk.dev/widget"

	"github.com/kyrokohan/skyspear/internal/sound"
	"github.com/kyrokohan/skyspear/internal/world"
)

// A pot and a hearth are stores: blocks with slots of their own, which E opens in the bag
// above the player's things, in survival. A pot keeps what is put in it. A hearth burns
// what it is fed, once it has been lit, and cooks what is put in it while it burns, a
// piece at a time: meat, the seeds of a stump palm, the tip of a spear, and clay to a
// brick or, shaped first, to a pot. Every hearth burns wherever it is while the world is
// played, with nobody to watch it.

// cookTime is how many seconds a hearth takes to cook one piece.
const cookTime = 6.0

// cook is what a hearth makes: one of makes from n of from.
type cook struct {
	from  world.Item
	n     int
	makes world.Item
}

var cooks = []cook{
	{world.Meat, 1, world.CookedMeat},
	{world.PalmSeeds, 1, world.RoastSeeds},
	{world.Spear, 1, world.HardSpear},
	{world.SoftPot, 1, world.ClayPot.Item()},
	{world.Clay.Item(), 1, world.Brick.Item()},
}

// burnTime is how many seconds a piece of an item burns in a hearth: 0 for what is not
// fuel.
func burnTime(item world.Item) float64 {
	switch {
	case item == world.Stick:
		return 15
	case item == world.Planks.Item():
		return 30
	case slices.Contains(logs, item):
		return 80
	case item == world.Peat.Item():
		return 240 // it burns slowly: three times as long as a log
	case item == world.Coal:
		return 160
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

// kindle sets fire to the next piece of a hearth's fuel, and reports whether there was
// one. It is how a hearth is lit, and how one that burns goes on burning.
func kindle(k *world.HearthStore) bool {
	seconds := burnTime(k.Fuel.Item)
	if k.Fuel.Count == 0 || seconds == 0 {
		return false
	}
	k.Burn, k.Burns = seconds, seconds
	takeFrom(&k.Fuel, 1)
	return true
}

// stepHearth moves a hearth on by dt seconds. A cold one stays cold until it is lit. One
// that burns goes on, a piece of fuel after another, whether or not there is anything to
// cook, and goes out when the fuel is gone. What it has cooked stays cooked, and what
// was half cooked when it went out is no further on when it is lit again; taking away
// what it was cooking starts the next one afresh.
func stepHearth(k *world.HearthStore, dt float64) {
	for dt > 0 && k.Burn > 0 {
		step := min(dt, 0.25) // so that a long step lights the next piece of fuel in time
		dt -= step
		if c, ok := cookIn(k); !ok {
			k.Cooked = 0
		} else if k.Cooked += step; k.Cooked >= cookTime {
			k.Cooked = 0
			takeFrom(&k.In, c.n)
			k.Out = world.Stack{Item: c.makes, Count: k.Out.Count + 1}
		}
		if k.Burn = max(k.Burn-step, 0); k.Burn == 0 {
			kindle(k)
		}
	}
}

// takeFrom takes n from a stack that has them.
func takeFrom(s *world.Stack, n int) {
	if s.Count -= n; s.Count <= 0 {
		*s = world.Stack{}
	}
}

// stepHearths moves every hearth on by dt seconds, and has the block of each be the one
// that burns while it does, so that it gives its light.
func (g *Game) stepHearths(dt float64) {
	for c, k := range g.hearths {
		stepHearth(k, dt)
		want := world.Hearth
		if k.Burn > 0 {
			want = world.HearthLit
		}
		if b := g.world.At(c.X, c.Y, c.Z); b != want && isHearth(b) {
			g.world.Set(c.X, c.Y, c.Z, want)
		}
	}
}

// isHearth reports whether a block is a hearth, cold or burning.
func isHearth(b world.Block) bool {
	return b == world.Hearth || b == world.HearthLit
}

// burns reports whether a block is a fire that a torch can be lit at.
func burns(b world.Block) bool {
	return b == world.Fire || b == world.HearthLit || b == world.Torch || b == world.Lava || b == world.HotCore
}

// light lights the cold hearth at a cell, if it has anything to burn, and reports
// whether it did.
func (g *Game) light(c world.Cell) bool {
	if k := g.hearth(c); k.Burn > 0 || !kindle(k) {
		return false
	}
	g.ears.Play(sound.Crackle, cellMid(c.X, c.Y, c.Z), 0.9)
	g.swing = swingTime
	g.tell("the hearth is lit")
	return true
}

// spark strikes fire with the striker in hand, onto the first unlit torches the player
// carries, which are lit. It reports whether it did: there is a striker in hand, and
// there are torches to light.
func (g *Game) spark() bool {
	if g.items[g.selected].Item != world.Striker {
		return false
	}
	for i := range g.items {
		if s := &g.items[i]; s.Item == world.UnlitTorch && s.Count > 0 {
			s.Item = world.Torch.Item()
			g.ears.PlayNear(sound.Crackle, 0.7)
			g.swing = swingTime
			g.tell("the torches are lit")
			g.wearOut()
			return true
		}
	}
	return false
}

// drillTime is how many seconds a fire drill is turned to light a hearth.
const drillTime = 6.0

// drill turns the fire drill in hand in the cold hearth aimed at for dt seconds, while E
// is held, and lights the hearth once it has been turned for drillTime without a rest.
// It reports whether the drill is turning, and then E does nothing else. A hearth with
// nothing to burn is not drilled: E opens it, to feed it.
func (g *Game) drill(holding bool, dt float64) bool {
	c := world.Cell{X: g.target.X, Y: g.target.Y, Z: g.target.Z}
	turning := holding && g.aimed && g.survival && g.target.Block == world.Hearth &&
		g.items[g.selected].Item == world.FireDrill && burnTime(g.hearth(c).Fuel.Item) > 0
	if !turning || c != g.drillAt {
		g.drillAt, g.drilling = c, 0 // a rest, or another hearth: it starts again from nothing
	}
	if !turning {
		return false
	}
	if g.drilling == 0 {
		g.tell("turning the fire drill")
	}
	mid := cellMid(c.X, c.Y, c.Z)
	if math.Floor(g.drilling/digStroke) != math.Floor((g.drilling+dt)/digStroke) || g.drilling == 0 {
		g.ears.Play(sound.Dig(sound.Wood), mid, 0.6)
		g.swing = swingTime
		if g.drilling > drillTime/2 {
			g.puff(mid.Add(geometry.Vector3{Y: 0.6}), 0.12, 1.2) // it begins to smoke
		}
	}
	if g.drilling += dt; g.drilling >= drillTime {
		g.drilling = 0
		g.light(c)
		g.wearOut()
	}
	return true
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
		// One whose block is out of the window may still show as burning there, and is
		// kept so that it is put right when it is seen again.
		if k.In.Count > 0 || k.Fuel.Count > 0 || k.Out.Count > 0 || k.Burn > 0 || !g.world.Loaded(k.X, k.Z) {
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
// it lies down on a fur bed, lights the unlit torches in hand at a fire, lights a cold
// hearth that has fuel with the torch in hand, and opens a pot or a hearth, the last
// three in survival. It reports whether it did.
func (g *Game) useBlock(t world.Hit) bool {
	c := world.Cell{X: t.X, Y: t.Y, Z: t.Z}
	held := &g.items[g.selected]
	switch {
	case t.Block == world.FurBed:
		g.lieDown(c)
	case burns(t.Block) && held.Item == world.UnlitTorch:
		held.Item = world.Torch.Item() // every one of them in the hand
		g.ears.Play(sound.Crackle, cellMid(c.X, c.Y, c.Z), 0.8)
		g.swing = swingTime
		g.tell("the torch is lit")
	case t.Block == world.Hearth && held.Item == world.Torch.Item() && g.survival && g.light(c):
	case t.Block == world.Hearth && held.Item == world.Striker && g.survival && g.light(c):
		g.wearOut() // a striker lights it at once, where a fire drill takes its time
	case isHearth(t.Block) && g.survival:
		g.openStore(world.Hearth, c)
	case t.Block == world.ClayPot && g.survival:
		g.openStore(world.ClayPot, c)
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

	lines := [3]string{"cooks meat, seeds,", "clay, pots, spears", "cold: drill or torch"}
	if made, ok := cookIn(k); ok {
		lines[0], lines[1] = "makes "+made.makes.Name(), ""
	} else if k.In.Count > 0 && cooksFrom(k.In.Item) {
		lines[0], lines[1] = "take out what it made", "to make more"
	}
	switch {
	case k.Burn > 0:
		lines[2] = "burning"
	case burnTime(k.Fuel.Item) == 0:
		lines[2] = "put in wood or coal"
	}
	tx := x + (hearthOut+1)*slotPitch + 2
	for i, line := range lines {
		widget.Label{Text: line, Style: hudText}.Draw(fb, tx, y+1+i, barWidth-(tx-x)-1)
	}
}

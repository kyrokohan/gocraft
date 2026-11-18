package game

import (
	"strconv"

	"glonk.dev/engine"
	"glonk.dev/widget"

	"github.com/kyrokohan/gocraft/internal/world"
)

// The bag is the screen of what the player carries, opened with I: rows of slots like
// the hotbar's, with the hotbar itself at the bottom. A slot is picked with the arrows and
// Enter, or with a click, which takes what is in it in hand, or puts down what is in hand,
// so that things are moved from one slot to another. In creative the rows above the
// hotbar are every block there is, and taking one takes as many as are wanted.
//
// In survival the list of what can be made is at the right of the slots: Right from the
// last column goes into it, and Enter or a click makes one of what is picked there. A
// crate or a kiln that is opened shows its slots in rows over the player's.

// rowPitch is how far down one row of slots is from the one above: they share a line.
const rowPitch = barHeight - 1

// listWidth is how wide the list of what can be made is, with its frame, and listGap the
// cells between it and the slots.
const (
	listWidth = 24
	listGap   = 1
)

const (
	bagHint  = " Arrows: move   Enter, click: take or put   I: close "
	makeHint = " Arrows: move   Enter, click: take, put or make   I: close "
)

// hudDim is the look of a line of the list that cannot be made, for want of something.
var hudDim = engine.Cell{Foreground: engine.RGB(92, 96, 112), Background: hudGround}

// palette is every item a creative player can take: every block, but water, which is
// not built with, and deepstone, which is the floor of the world.
var palette = func() (items []world.Item) {
	for b := world.Grass; b.Known(); b++ {
		if b != world.Water && b != world.Deepstone {
			items = append(items, b.Item())
		}
	}
	return items
}()

// bagRows returns how many rows of slots the bag shows: those of a store that is open,
// the pack's, or in creative as many as every block takes, and then the hotbar's.
func (g *Game) bagRows() int {
	if g.survival {
		return g.storeRows() + packSlots/hotbarSlots + 1
	}
	return (len(palette)+hotbarSlots-1)/hotbarSlots + 1
}

// bagCell returns what is at a row and a column of the bag: a slot, of the player's or
// of a store that is open, or nil and an item of the palette, which may be none. Where a
// kiln's row has no slot there is nil and nothing.
func (g *Game) bagCell(row, col int) (s *world.Stack, item world.Item) {
	if row < g.storeRows() {
		return g.storeSlot(row, col), world.NoItem
	}
	switch {
	case row == g.bagRows()-1:
		return &g.items[col], world.NoItem
	case g.survival:
		return &g.items[hotbarSlots+(row-g.storeRows())*hotbarSlots+col], world.NoItem
	}
	if i := row*hotbarSlots + col; i < len(palette) {
		return nil, palette[i]
	}
	return nil, world.NoItem
}

// bagWidth is how wide the bag is: the slots, and in survival the list beside them.
func (g *Game) bagWidth() int {
	if g.survival {
		return barWidth + listGap + listWidth
	}
	return barWidth
}

// bagOrigin returns the left of the rows of the bag in a frame of a size in cells, and
// the top of its title over them.
func (g *Game) bagOrigin(width, height int) (x, top int) {
	tall := 1 + g.bagRows()*rowPitch + 1 + 1 + 2 // the title, the rows, the gap over the hotbar, and two lines under
	return max((width-g.bagWidth())/2, 0), max((height-tall)/2, 0)
}

// bagRowY returns the top of a row of the bag. The hotbar's is set apart by a line.
func (g *Game) bagRowY(top, row int) int {
	y := top + 1 + row*rowPitch
	if row == g.bagRows()-1 {
		y++
	}
	return y
}

// listBox returns the top left corner of the panel of the list of what can be made, and
// its height, in a frame of a size in cells: beside the rows of slots, from the top of the first to the bottom of the
// hotbar.
func (g *Game) listBox(width, height int) (x, y, h int) {
	x0, top := g.bagOrigin(width, height)
	y = top + 1
	return x0 + barWidth + listGap, y, g.bagRowY(top, g.bagRows()-1) + barHeight - y
}

// openBag shows the bag, with the keys on the slot of the hotbar in use.
func (g *Game) openBag() {
	g.opened = world.Air
	g.cursor = (g.bagRows()-1)*hotbarSlots + g.selected
	g.inList, g.recipe, g.listTop = false, 0, 0
	g.hand, g.digging = world.Stack{}, 0
	g.show(inBag)
}

// closeBag goes back to play. What is still in hand goes back into the inventory, and
// what does not fit is put down in front of the player.
func (g *Game) closeBag() {
	if g.survival && g.hand.Count > 0 {
		if left := g.items.add(g.hand); left > 0 {
			g.putDown(world.Stack{Item: g.hand.Item, Count: left, Wear: g.hand.Wear})
		}
	}
	g.hand, g.opened = world.Stack{}, world.Air
	g.show(playing)
}

// useBag reads the keys and the mouse while the bag is open.
func (g *Game) useBag(in *engine.Input, escape bool) {
	if escape || keyDown(in.Started, 'i') {
		g.closeBag()
		return
	}
	if x, y, ok := in.MousePos(); ok {
		// The pointer moved onto a line of the list picks it, so that what it needs is
		// shown before it is clicked.
		moved := x != g.mouseX || y != g.mouseY
		g.mouseX, g.mouseY = x, y
		i, onList := g.listAt(x, y)
		if onList && moved {
			g.inList, g.recipe = true, i
		}
		if in.MouseStarted(engine.MouseLeft) {
			if r, c, on := g.bagAt(x, y); on && g.isCell(r, c) {
				g.inList, g.cursor = false, r*hotbarSlots+c
				g.pickCell(r, c)
				return
			}
			if onList {
				g.inList, g.recipe = true, i
				g.craft(g.makeable()[i])
				return
			}
		}
	}
	if g.inList {
		g.useList(in)
		return
	}

	rows := g.bagRows()
	row, col := g.cursor/hotbarSlots, g.cursor%hotbarSlots
	switch {
	case in.Pressed(engine.KeyLeft):
		col = (col + hotbarSlots - 1) % hotbarSlots
	case in.Pressed(engine.KeyRight) && g.survival && col == hotbarSlots-1:
		g.inList = true
		return
	case in.Pressed(engine.KeyRight):
		col = (col + 1) % hotbarSlots
	case in.Pressed(engine.KeyUp):
		row = (row + rows - 1) % rows
	case in.Pressed(engine.KeyDown):
		row = (row + 1) % rows
	}
	if !g.isCell(row, col) {
		// A kiln's row has slots in a few of its columns: the keys go on to the next one
		// along, or from another row, to the slot of the kiln over them or the last.
		switch {
		case in.Pressed(engine.KeyLeft) && col == kilnOut-1:
			col = kilnFuel
		case in.Pressed(engine.KeyLeft):
			col = kilnOut
		case in.Pressed(engine.KeyRight) && col == kilnOut-1:
			col = kilnOut
		case in.Pressed(engine.KeyRight):
			col = kilnIn
		default:
			col = kilnOut
		}
	}
	g.cursor = row*hotbarSlots + col
	if in.Started(engine.KeyEnter) || in.Started(' ') {
		g.pickCell(row, col)
	}
}

// useList reads the keys while they are on the list of what can be made. Enter makes
// one, and goes on making them while it is held.
func (g *Game) useList(in *engine.Input) {
	here := g.makeable()
	g.recipe = min(g.recipe, len(here)-1)
	switch {
	case in.Pressed(engine.KeyLeft):
		g.inList = false
		g.cursor = g.cursor/hotbarSlots*hotbarSlots + hotbarSlots - 1
	case in.Pressed(engine.KeyUp):
		g.recipe = (g.recipe + len(here) - 1) % len(here)
	case in.Pressed(engine.KeyDown):
		g.recipe = (g.recipe + 1) % len(here)
	case in.Pressed(engine.KeyEnter) || in.Pressed(' '):
		g.craft(here[g.recipe])
	}

	// The list scrolls to keep the line the keys are on in view.
	_, _, h := g.listBox(g.cols, g.rows)
	lines := h - 2
	g.listTop = min(g.listTop, g.recipe)
	g.listTop = max(g.listTop, g.recipe-lines+1)
}

// bagAt returns the row and the column of the bag a cell of the frame is in, if it is in one.
func (g *Game) bagAt(x, y int) (row, col int, ok bool) {
	x0, top := g.bagOrigin(g.cols, g.rows)
	if x < x0 || x >= x0+barWidth-1 {
		return 0, 0, false
	}
	for r := range g.bagRows() {
		if ry := g.bagRowY(top, r); y >= ry && y < ry+barHeight {
			return r, (x - x0) / slotPitch, true
		}
	}
	return 0, 0, false
}

// listAt returns the line of the list of what can be made a cell of the frame is on, if
// it is on one.
func (g *Game) listAt(x, y int) (int, bool) {
	if !g.survival {
		return 0, false
	}
	lx, ly, h := g.listBox(g.cols, g.rows)
	i := g.listTop + y - (ly + 1)
	if x <= lx || x >= lx+listWidth-1 || y <= ly || y >= ly+h-1 || i >= len(g.makeable()) {
		return 0, false
	}
	return i, true
}

// isCell reports whether a row and a column of the bag are a place the keys can be on: in
// a kiln's row, only its slots.
func (g *Game) isCell(row, col int) bool {
	s, _ := g.bagCell(row, col)
	return s != nil || !g.survival
}

// pickCell takes what is in a cell of the bag in hand, or puts down what is in hand there:
// on the same kind, as many as fit, and in the place of another kind, which is taken up.
// Tools are one to a slot, so a tool put on another changes places with it. What a kiln
// has made can only be taken, and only what it cooks, or burns, goes where it does.
func (g *Game) pickCell(row, col int) {
	s, item := g.bagCell(row, col)
	if s == nil {
		if g.survival {
			return // no slot
		}
		// A block of the palette, which takes the place of what is in hand.
		g.hand = world.Stack{}
		if item != world.NoItem {
			g.hand = world.Stack{Item: item, Count: 1}
		}
		return
	}
	if g.opened == world.Kiln && s == &g.kiln(g.at).Out {
		g.takeMade(s)
		return
	}
	if !g.takes(s) {
		return
	}
	if g.opened == world.Kiln && s == &g.kiln(g.at).In {
		g.kiln(g.at).Cooked = 0 // what was cooking is taken out, or more is put in
	}
	if size := s.Item.StackSize(); g.survival && g.hand.Count > 0 && s.Item == g.hand.Item && size > 1 {
		put := min(g.hand.Count, size-s.Count)
		s.Count += put
		if g.hand.Count -= put; g.hand.Count == 0 {
			g.hand = world.Stack{}
		}
		return
	}
	*s, g.hand = g.hand, *s
}

// takeMade takes what a kiln has made into the hand: all of it into an empty hand, or as
// many as the hand has room for of the same kind.
func (g *Game) takeMade(s *world.Stack) {
	switch {
	case g.hand.Count == 0:
		g.hand, *s = *s, world.Stack{}
	case g.hand.Item == s.Item:
		put := min(s.Count, s.Item.StackSize()-g.hand.Count)
		g.hand.Count += put
		takeFrom(s, put)
	}
}

// drawBag draws the bag over the world, after Canvas.Compose.
func (g *Game) drawBag(fb *engine.FrameBuffer, c *engine.Canvas) {
	x0, top := g.bagOrigin(fb.Width(), fb.Height())
	title := " your things "
	switch {
	case !g.survival:
		title = " every block "
	case g.opened != world.Air:
		title = " " + g.opened.Info().Name + " "
	}
	widget.Label{Text: title, Align: widget.Center, Style: hudText}.Draw(fb, x0, top, barWidth)

	rows := g.bagRows()
	var under int
	for r := range rows {
		y := g.bagRowY(top, r)
		under = y + barHeight
		if g.opened == world.Kiln && r == 0 {
			g.drawKiln(fb, c, x0, y, g.kiln(g.at))
			continue
		}
		row := make([]world.Stack, hotbarSlots)
		for col := range row {
			if s, item := g.bagCell(r, col); s != nil {
				row[col] = *s
			} else if item != world.NoItem {
				row[col] = world.Stack{Item: item, Count: 1}
			}
		}
		g.drawRow(fb, c, x0, y, row)
		if r == g.storeRows() && r > 0 {
			// The player's things are under the store's, and the line between says so, at
			// its right, out of the way of the keys on the store's first slots.
			const mine = " your things "
			widget.Label{Text: mine, Style: hudFrame}.Draw(fb, x0+barWidth-2-len(mine), y, len(mine))
		}
	}
	row, col := g.cursor/hotbarSlots, g.cursor%hotbarSlots
	if !g.inList {
		widget.Panel{Style: hudPick}.Draw(fb, x0+col*slotPitch, g.bagRowY(top, row), slotPitch+1, barHeight)
	}

	// What is in hand, or what the line of the list the keys are on needs, or else the
	// name of what the keys are on.
	text := ""
	switch {
	case g.hand.Count > 0:
		text = " holding " + g.hand.Item.Name() + " "
		if g.survival && g.hand.Count > 1 {
			text = " holding " + strconv.Itoa(g.hand.Count) + " " + g.hand.Item.Name() + " "
		}
	case g.inList:
		if here := g.makeable(); g.recipe < len(here) {
			makes, from := here[g.recipe].label()
			text = " " + makes + " from " + from + " "
		}
	default:
		if s, item := g.bagCell(row, col); s != nil && s.Count > 0 {
			text = " " + s.Item.Name() + " "
		} else if item != world.NoItem {
			text = " " + item.Name() + " "
		} else if g.opened == world.Kiln && row == 0 {
			text = [...]string{kilnIn: " what to cook ", kilnFuel: " fuel ", kilnOut: " what it made "}[col]
		}
	}
	if text != "" {
		widget.Label{Text: text, Align: widget.Center, Style: hudPick}.Draw(fb, x0, under, g.bagWidth())
	}
	hint := bagHint
	if g.survival {
		hint = makeHint
		g.drawList(fb)
	}
	widget.Label{Text: hint, Align: widget.Center, Style: hudText}.Draw(fb, 0, under+1, fb.Width())
}

// drawList draws the list of what can be made where the player stands, dim where there
// is not enough for it. What a line needs is shown under the bag while it is picked.
func (g *Game) drawList(fb *engine.FrameBuffer) {
	x, y, h := g.listBox(fb.Width(), fb.Height())
	title := "make by hand"
	if g.near(world.Workbench) {
		title = "make: workbench"
	}
	widget.Panel{Title: title, Style: hudFrame, Fill: hudFill}.Draw(fb, x, y, listWidth, h)

	here, lines := g.makeable(), h-2
	top := min(g.listTop, max(len(here)-lines, 0))
	for line := range min(lines, len(here)-top) {
		i, row := top+line, y+1+line
		makes, _ := here[i].label()
		style := hudText
		if !g.affords(here[i]) {
			style = hudDim
		}
		if g.inList && i == g.recipe {
			marker := hudPick
			marker.Character = '▶'
			fb.Set(x+1, row, marker)
			if style == hudText {
				style = hudPick
			}
		}
		widget.Label{Text: makes, Style: style}.Draw(fb, x+3, row, listWidth-5)
	}

	// Marks on the frame where there is more of the list than it shows.
	more := hudFrame
	if top > 0 {
		more.Character = '▲'
		fb.Set(x+listWidth-3, y, more)
	}
	if top+lines < len(here) {
		more.Character = '▼'
		fb.Set(x+listWidth-3, y+h-1, more)
	}
}

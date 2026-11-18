package game

import (
	"strconv"

	"glonk.dev/engine"
	"glonk.dev/geometry"
	"glonk.dev/mesh"
	"glonk.dev/widget"

	"github.com/kyrokohan/gocraft/internal/world"
)

// The bag is the screen of what the player carries, opened with I: rows of slots like
// the hotbar's, with the hotbar itself at the bottom. A slot is picked with the arrows and
// Enter, or with a click, which takes what is in it in hand, or puts down what is in hand,
// so that things are moved from one slot to another. In creative the rows above the
// hotbar are every block there is, and taking one takes as many as are wanted.

// rowPitch is how far down one row of slots is from the one above: they share a line.
const rowPitch = barHeight - 1

const bagHint = " Arrows: move   Enter, click: take or put   I: close "

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

// bagRows returns how many rows of slots the bag shows: the pack's, or in creative as
// many as every block takes, and then the hotbar's.
func (g *Game) bagRows() int {
	if g.survival {
		return packSlots/hotbarSlots + 1
	}
	return (len(palette)+hotbarSlots-1)/hotbarSlots + 1
}

// bagCell returns what is at a row and a column of the bag: a slot of the inventory, or
// -1 and an item of the palette, which may be none.
func (g *Game) bagCell(row, col int) (slot int, item world.Item) {
	switch {
	case row == g.bagRows()-1:
		return col, world.NoItem
	case g.survival:
		return hotbarSlots + row*hotbarSlots + col, world.NoItem
	}
	if i := row*hotbarSlots + col; i < len(palette) {
		return -1, palette[i]
	}
	return -1, world.NoItem
}

// bagOrigin returns the left of the rows of the bag in a frame of a size in cells, and
// the top of its title over them.
func (g *Game) bagOrigin(width, height int) (x, top int) {
	tall := 1 + g.bagRows()*rowPitch + 1 + 1 + 2 // the title, the rows, the gap over the hotbar, and two lines under
	return (width - barWidth) / 2, max((height-tall)/2, 0)
}

// bagRowY returns the top of a row of the bag. The hotbar's is set apart by a line.
func (g *Game) bagRowY(top, row int) int {
	y := top + 1 + row*rowPitch
	if row == g.bagRows()-1 {
		y++
	}
	return y
}

// openBag shows the bag, with the keys on the slot of the hotbar in use.
func (g *Game) openBag() {
	g.cursor = (g.bagRows()-1)*hotbarSlots + g.selected
	g.hand, g.digging = world.Stack{}, 0
	g.show(inBag)
}

// closeBag goes back to play. What is still in hand goes back into the inventory, and
// what does not fit is put down in front of the player.
func (g *Game) closeBag() {
	if g.survival && g.hand.Count > 0 {
		if left := g.items.add(g.hand.Item, g.hand.Count); left > 0 {
			ahead := mesh.Camera{Yaw: g.player.Yaw}.Forward()
			at := g.player.Pos.Add(ahead).Add(geometry.Vector3{Y: 1})
			g.drops = append(g.drops, drop{Drop: world.Drop{Stack: world.Stack{Item: g.hand.Item, Count: left}, Pos: at}})
		}
	}
	g.hand = world.Stack{}
	g.show(playing)
}

// useBag reads the keys and the mouse while the bag is open.
func (g *Game) useBag(in *engine.Input, escape bool) {
	if escape || keyDown(in.Started, 'i') {
		g.closeBag()
		return
	}
	rows := g.bagRows()
	row, col := g.cursor/hotbarSlots, g.cursor%hotbarSlots
	switch {
	case in.Pressed(engine.KeyLeft):
		col = (col + hotbarSlots - 1) % hotbarSlots
	case in.Pressed(engine.KeyRight):
		col = (col + 1) % hotbarSlots
	case in.Pressed(engine.KeyUp):
		row = (row + rows - 1) % rows
	case in.Pressed(engine.KeyDown):
		row = (row + 1) % rows
	}
	pick := in.Started(engine.KeyEnter) || in.Started(' ')
	if x, y, ok := in.MousePos(); ok && in.MouseStarted(engine.MouseLeft) {
		if r, c, on := g.bagAt(x, y); on {
			row, col, pick = r, c, true
		}
	}
	g.cursor = row*hotbarSlots + col
	if pick {
		g.pickCell(row, col)
	}
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

// pickCell takes what is in a cell of the bag in hand, or puts down what is in hand there:
// on the same kind, as many as fit, and in the place of another kind, which is taken up.
func (g *Game) pickCell(row, col int) {
	slot, item := g.bagCell(row, col)
	if slot < 0 {
		// A block of the palette, which takes the place of what is in hand.
		g.hand = world.Stack{}
		if item != world.NoItem {
			g.hand = world.Stack{Item: item, Count: 1}
		}
		return
	}
	s := &g.items[slot]
	if g.survival && g.hand.Count > 0 && s.Item == g.hand.Item {
		put := min(g.hand.Count, world.StackSize-s.Count)
		s.Count += put
		if g.hand.Count -= put; g.hand.Count == 0 {
			g.hand = world.Stack{}
		}
		return
	}
	*s, g.hand = g.hand, *s
}

// drawBag draws the bag over the world, after Canvas.Compose.
func (g *Game) drawBag(fb *engine.FrameBuffer, c *engine.Canvas) {
	x0, top := g.bagOrigin(fb.Width(), fb.Height())
	title := " your things "
	if !g.survival {
		title = " every block "
	}
	widget.Label{Text: title, Align: widget.Center, Style: hudText}.Draw(fb, x0, top, barWidth)

	rows := g.bagRows()
	var under int
	for r := range rows {
		row := make([]world.Stack, hotbarSlots)
		for col := range row {
			if slot, item := g.bagCell(r, col); slot >= 0 {
				row[col] = g.items[slot]
			} else if item != world.NoItem {
				row[col] = world.Stack{Item: item, Count: 1}
			}
		}
		y := g.bagRowY(top, r)
		g.drawRow(fb, c, x0, y, row)
		under = y + barHeight
	}
	row, col := g.cursor/hotbarSlots, g.cursor%hotbarSlots
	widget.Panel{Style: hudPick}.Draw(fb, x0+col*slotPitch, g.bagRowY(top, row), slotPitch+1, barHeight)

	// What is in hand, or else the name of what the keys are on.
	text := ""
	if g.hand.Count > 0 {
		text = " holding " + g.hand.Item.Name() + " "
		if g.survival && g.hand.Count > 1 {
			text = " holding " + strconv.Itoa(g.hand.Count) + " " + g.hand.Item.Name() + " "
		}
	} else if slot, item := g.bagCell(row, col); slot >= 0 && g.items[slot].Count > 0 {
		text = " " + g.items[slot].Item.Name() + " "
	} else if item != world.NoItem {
		text = " " + item.Name() + " "
	}
	if text != "" {
		widget.Label{Text: text, Align: widget.Center, Style: hudPick}.Draw(fb, x0, under, barWidth)
	}
	widget.Label{Text: bagHint, Align: widget.Center, Style: hudText}.Draw(fb, 0, under+1, fb.Width())
}

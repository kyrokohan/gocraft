package game

import (
	"cmp"
	"maps"
	"slices"

	"glonk.dev/engine"
	"glonk.dev/widget"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/sound"
	"github.com/kyrokohan/skyspear/internal/world"
)

// Painting on rock. E with a paint in hand on a face of rock opens the picker: the
// pictures the player knows, in a row. One paint paints one, in its color, over the whole
// face. The paintings are kept beside the world, as the pots are, and saved with it.

// The picker: each picture in a slot, with a line between them, framed.
const (
	pickerPitch  = render.PictureSide + 1   // from one picture to the next
	pickerHeight = render.PictureSide/2 + 2 // in cells, with the frame
	pickerHint   = " Left, Right: choose   Enter: paint   Esc: leave "
)

// openPicker shows the picker, to paint the face of rock a ray met with the paint in hand.
func (g *Game) openPicker(t world.Hit) {
	g.wall, g.pick = world.Side{Cell: world.Cell{X: t.X, Y: t.Y, Z: t.Z}, Face: t.Face}, 0
	g.shown = g.shown[:0]
	for _, p := range g.known() {
		g.shown = append(g.shown, g.rend.Picture(world.Painting{Side: g.wall, Picture: p, Paint: g.items[g.selected].Item}, t.Block))
	}
	g.digging = 0
	g.show(picking)
}

// usePicker reads the keys while the picker is open.
func (g *Game) usePicker(in *engine.Input, escape bool) {
	n := len(g.shown)
	switch {
	case escape:
		g.show(playing)
	case in.Pressed(engine.KeyLeft):
		g.pick = (g.pick + n - 1) % n
	case in.Pressed(engine.KeyRight):
		g.pick = (g.pick + 1) % n
	case in.Started(engine.KeyEnter):
		g.paint(g.known()[g.pick])
		g.show(playing)
	}
}

// paint paints a picture on the face the picker was opened on, with the paint in hand.
func (g *Game) paint(p world.Picture) {
	held := &g.items[g.selected]
	if !held.Item.IsPaint() || !world.Paintable(g.world.At(g.wall.X, g.wall.Y, g.wall.Z)) {
		return
	}
	g.paintings[g.wall] = world.Painting{Side: g.wall, Picture: p, Paint: held.Item}
	g.ears.Play(sound.Place(sound.Sand), cellMid(g.wall.X, g.wall.Y, g.wall.Z), 0.5)
	g.swing = swingTime
	if g.survival {
		g.items.takeOne(g.selected)
	}
}

// keepPaintings drops the paintings whose rock is gone: dug, burned or struck away. One
// where the world is not loaded is kept, as its rock cannot be seen to be there.
func (g *Game) keepPaintings() {
	for side := range g.paintings {
		if g.world.Loaded(side.X, side.Z) && !world.Paintable(g.world.At(side.X, side.Y, side.Z)) {
			delete(g.paintings, side)
		}
	}
}

// savedPaintings returns the paintings as a save holds them, in order.
func (g *Game) savedPaintings() []world.Painting {
	return slices.SortedFunc(maps.Values(g.paintings), func(a, b world.Painting) int {
		return cmp.Or(compareCells(a.Cell, b.Cell), cmp.Compare(a.Face, b.Face))
	})
}

// drawPicker draws the picker in the middle of the frame: the pictures in a row, the one
// chosen framed bright, with its name over them and the keys under them. A row wider than
// the frame shows the part of it round the one chosen. It comes after Canvas.Compose, as
// the HUD does.
func (g *Game) drawPicker(fb *engine.FrameBuffer, c *engine.Canvas) {
	fit := max((fb.Width()-1)/pickerPitch, 1)
	first := min(max(g.pick-fit/2, 0), max(len(g.shown)-fit, 0))
	shown := g.shown[first:min(first+fit, len(g.shown))]
	w := len(shown)*pickerPitch + 1
	x, y := (fb.Width()-w)/2, max((fb.Height()-pickerHeight)/2, 1)
	widget.Panel{Style: hudFrame, Fill: hudFill}.Draw(fb, x, y, w, pickerHeight)
	line := hudFrame
	line.Character = '│'
	for i := 1; i < len(shown); i++ {
		for r := 1; r < pickerHeight-1; r++ {
			fb.Set(x+i*pickerPitch, y+r, line)
		}
	}
	widget.Panel{Style: hudPick}.Draw(fb, x+(g.pick-first)*pickerPitch, y, pickerPitch+1, pickerHeight)

	// The pictures are pixels over the cells of the panel, with the world's rubbed out.
	c.FillRect(x, 2*y, w, 2*pickerHeight, engine.Color{})
	for i, pic := range shown {
		pic.Draw(c, x+1+i*pickerPitch, 2*(y+1))
	}
	c.ComposeRect(x, 2*y, w, 2*pickerHeight)

	name := " " + nameOf(g.known()[g.pick]) + " "
	widget.Label{Text: name, Align: widget.Center, Style: hudText}.Draw(fb, 0, y-1, fb.Width())
	widget.Label{Text: pickerHint, Align: widget.Center, Style: hudText}.Draw(fb, 0, y+pickerHeight, fb.Width())
}

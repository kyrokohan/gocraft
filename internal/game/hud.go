package game

import (
	"fmt"
	"math"

	"glonk.dev/engine"
	"glonk.dev/widget"

	"github.com/kyrokohan/gocraft/internal/assets"
	"github.com/kyrokohan/gocraft/internal/tile"
)

// The look of everything drawn over the world: light text and frames on a dark ground,
// and one bright color for what is picked.
var (
	hudGround = engine.RGB(24, 26, 34)
	hudText   = engine.Cell{Foreground: engine.RGB(236, 236, 240), Background: hudGround}
	hudFrame  = engine.Cell{Foreground: engine.RGB(140, 146, 166), Background: hudGround}
	hudPick   = engine.Cell{Foreground: engine.RGB(255, 214, 64), Background: hudGround}
	hudFill   = engine.Cell{Character: ' ', Background: hudGround}
)

// The hotbar: nine slots side by side along the bottom, each with a picture of its
// block. A slot is slotSize cells wide and half as many tall, which is slotSize pixels
// each way, so the picture is square.
const (
	slotSize  = 6
	slotPitch = slotSize + 1                       // from one slot to the next, with the line between
	barWidth  = len(hotbar)*slotPitch + 1          // with the frame
	barHeight = slotSize/2 + 2                     // with the frame
	titleUnit = 4                                  // pixels of the title picture to each little block of its letters
	menuWidth = 22                                 // cells, with the frame
	helpWidth = 46                                 // cells, with the frame
	hint      = " Up, Down: choose   Enter: pick " // under the title's menu
)

// help is the keys, shown on the pause menu.
var help = []string{
	"W A S D   walk         Arrows  look",
	"Space     jump, up     X       down",
	"F         fly on, off  Esc     this menu",
	"Q         break        E       place",
	"1 to 9    pick block   Tab     next block",
}

// barOrigin returns the top left cell of the hotbar: along the bottom, in the middle.
func barOrigin(fb *engine.FrameBuffer) (x, y int) {
	return (fb.Width() - barWidth) / 2, fb.Height() - barHeight
}

// drawHotbarIcons draws the picture of each hotbar block into the pixels of its slot.
// It comes before Canvas.Compose; the frame round the slots is cells, and comes after.
func (g *Game) drawHotbarIcons(fb *engine.FrameBuffer, c *engine.Canvas) {
	x0, y0 := barOrigin(fb)
	for i, b := range hotbar {
		x, y := x0+1+i*slotPitch, 2*(y0+1)
		c.FillRect(x-1, y, slotPitch+1, slotSize, hudGround) // also under the lines between slots
		drawIcon(c, &g.set.Tiles[b.Info().Top], x, y, slotSize)
	}
}

// drawIcon draws a tile size pixels across with its top left pixel at (x, y), leaving
// its holes alone.
func drawIcon(c *engine.Canvas, tex *assets.Texture, x, y, size int) {
	for j := range size {
		for i := range size {
			// The pixel of the tile under the middle of this one.
			p := tex[0][((2*j+1)*tile.Size/(2*size))*tile.Size+(2*i+1)*tile.Size/(2*size)]
			if p>>24 != 0 {
				c.Set(x+i, y+j, engine.RGB(uint8(p>>16), uint8(p>>8), uint8(p)))
			}
		}
	}
}

// drawHUD draws what lies over the world while playing: the crosshair, a line of
// numbers, and the frame of the hotbar with the name of the block in use.
func (g *Game) drawHUD(fb *engine.FrameBuffer) {
	// The crosshair keeps the color of the world under it as its background.
	cx, cy := fb.Width()/2, fb.Height()/2
	cross := fb.At(cx, cy)
	cross.Character, cross.Foreground, cross.Attrs = '+', hudText.Foreground, engine.Bold
	fb.Set(cx, cy, cross)

	mode := ""
	if g.player.Flying {
		mode = "  flying"
	}
	p := g.player.Pos
	status := fmt.Sprintf(" %.0f fps  %d %d %d%s  Esc: menu ", g.fps,
		int(math.Floor(p.X)), int(math.Floor(p.Y)), int(math.Floor(p.Z)), mode)
	widget.Label{Text: status, Style: hudText}.Draw(fb, 0, 0, fb.Width())

	x0, y0 := barOrigin(fb)
	widget.Panel{Style: hudFrame}.Draw(fb, x0, y0, barWidth, barHeight)
	for i := range hotbar {
		x := x0 + i*slotPitch
		if i > 0 {
			line := hudFrame
			line.Character = '│'
			for row := 1; row < barHeight-1; row++ {
				fb.Set(x, y0+row, line)
			}
		}
		number := hudFrame
		number.Character = rune('1' + i)
		fb.Set(x+slotPitch/2, y0, number)
	}

	// The slot in use has a bright frame of its own, and its name over the bar.
	x := x0 + g.selected*slotPitch
	widget.Panel{Style: hudPick}.Draw(fb, x, y0, slotPitch+1, barHeight)
	number := hudPick
	number.Character = rune('1' + g.selected)
	fb.Set(x+slotPitch/2, y0, number)
	name := " " + hotbar[g.selected].Info().Name + " "
	widget.Label{Text: name, Align: widget.Center, Style: hudText}.Draw(fb, x0, y0-1, barWidth)
}

// drawMenu draws the menu in a panel w cells wide with its top left corner at (x, y),
// and returns the row under it.
func (g *Game) drawMenu(fb *engine.FrameBuffer, title string, x, y, w, extra int) int {
	h := len(g.menu.Items) + extra + 2
	widget.Panel{Title: title, Style: hudFrame, Fill: hudFill}.Draw(fb, x, y, w, h)
	g.menu.Draw(fb, x+2, y+1, w-4, len(g.menu.Items))
	return y + h
}

// drawTitle draws the title screen over the world: the name of the game as pixels,
// then the menu under it.
func (g *Game) drawTitle(fb *engine.FrameBuffer, c *engine.Canvas) {
	// The picture is drawn at a whole number of pixels to each little block of its
	// letters, the most that fits, so every letter keeps its shape.
	cols, rows := g.set.Title.Width()/titleUnit, g.set.Title.Height()/titleUnit
	scale := min(max((c.Width()-4)/cols, 1), titleUnit)
	w, h := cols*scale, rows*scale
	top := max(c.Height()/6, 1)
	g.set.Title.DrawScaled(c, (c.Width()-w)/2, top, w, h)
	c.Compose()

	question, width := "", menuWidth
	if g.screen == confirming {
		question = "replace your saved world?"
		width = engine.TextWidth(question) + 8
	}
	y := min((top+h)/2+2, fb.Height()-len(g.menu.Items)-3)
	under := g.drawMenu(fb, question, (fb.Width()-width)/2, y, width, 0)
	widget.Label{Text: hint, Align: widget.Center, Style: hudText}.Draw(fb, 0, under+1, fb.Width())
}

// drawPause draws the pause menu over the world, with the keys under it.
func (g *Game) drawPause(fb *engine.FrameBuffer) {
	h := len(g.menu.Items) + len(help) + 3
	x, y := (fb.Width()-helpWidth)/2, max((fb.Height()-h)/2, 0)
	g.drawMenu(fb, "paused", x, y, helpWidth, len(help)+1)
	for i, line := range help {
		widget.Label{Text: line, Style: hudFrame}.Draw(fb, x+2, y+len(g.menu.Items)+2+i, helpWidth-4)
	}
}

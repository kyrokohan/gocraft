package game

import (
	"fmt"
	"math"

	"glonk.dev/engine"
	"glonk.dev/widget"

	"github.com/kyrokohan/skyspear/internal/world"
)

// A painting is a record you read. E on one shows its card: the picture large, what it
// is called, and what you know of it: a beast's ways, how many nights until the comet
// falls, or of a hand, that you were here.

const (
	cardLine = 44           // the most characters a line of a card has
	cardWide = cardLine + 4 // the card, with its frame
	cardHint = " Esc or E: close "
)

// sideOf returns the face of a block a ray met.
func sideOf(t world.Hit) world.Side {
	return world.Side{Cell: world.Cell{X: t.X, Y: t.Y, Z: t.Z}, Face: t.Face}
}

// openCard shows the card of a painting on a kind of block.
func (g *Game) openCard(p world.Painting, on world.Block) {
	g.card, g.cardPicture = p, g.rend.Picture(p, on, true)
	g.digging = 0
	g.show(reading)
}

// cardLines returns what the card of a picture says under its name.
func (g *Game) cardLines(p world.Picture) []string {
	switch p {
	case world.Hand:
		return []string{"you were here"}
	case world.Spiral:
		return nil
	case world.Comet:
		return []string{g.cometNights()}
	}
	return breeds[p-world.FirstBeast].lore
}

// cometNights tells how many nights it is until the comet falls, by the clock.
func (g *Game) cometNights() string {
	if since := g.greatFall(); since >= 0 && since < fallLasts {
		return "it is falling now"
	}
	_, left := g.comet()
	switch n := int(math.Round(left)); n {
	case 0:
		return "it falls tonight"
	case 1:
		return "it falls next night"
	default:
		return fmt.Sprintf("it falls in %d nights", n)
	}
}

// drawCard draws the card of the painting in the middle of the frame: its name, its
// picture, and its lines, with the keys under it. It comes after Canvas.Compose, as the
// HUD does.
func (g *Game) drawCard(fb *engine.FrameBuffer, c *engine.Canvas) {
	lines := g.cardLines(g.card.Picture)
	side := g.cardPicture.Width() // pixels, which is cells across and twice the cells down
	h := side/2 + len(lines) + 4
	x, y := (fb.Width()-cardWide)/2, max((fb.Height()-h-1)/2, 0)
	widget.Panel{Style: hudFrame, Fill: hudFill}.Draw(fb, x, y, cardWide, h)
	widget.Label{Text: nameOf(g.card.Picture), Align: widget.Center, Style: hudText}.Draw(fb, x+1, y+1, cardWide-2)

	// The picture is pixels over the cells of the panel, with the world's rubbed out.
	px, py := x+(cardWide-side)/2, 2*(y+2)
	c.FillRect(px, py, side, side, engine.Color{})
	g.cardPicture.Draw(c, px, py)
	c.ComposeRect(px, py, side, side)

	for i, line := range lines {
		widget.Label{Text: line, Align: widget.Center, Style: hudText}.Draw(fb, x+1, y+2+side/2+1+i, cardWide-2)
	}
	widget.Label{Text: cardHint, Align: widget.Center, Style: hudText}.Draw(fb, 0, y+h, fb.Width())
}

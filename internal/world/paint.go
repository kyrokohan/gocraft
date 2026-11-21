package world

// Picture is what a painting shows.
type Picture uint8

// The pictures, in the order a save numbers them.
const (
	Hand       Picture = iota // a stencil: the paint blown round a hand, which is left bare
	Spiral                    // a line that winds out from the middle
	Comet                     // the Skyspear: a head and a long tail
	FirstBeast                // a beast side on: one of the game's kind k is FirstBeast + k

	Pictures = FirstBeast + BeastKinds // how many there are
)

// BeastKinds is how many kinds of beast the game has. A new kind is only ever added
// after the last, so that a painting of one keeps its number.
const BeastKinds = 22

// Name returns what a picture is called; for a beast, nothing: the game names its kinds.
func (p Picture) Name() string {
	if p < FirstBeast {
		return [FirstBeast]string{"hand", "spiral", "comet"}[p]
	}
	return ""
}

// Side is one face of the block in a cell.
type Side struct {
	Cell
	Face Face `json:"face"`
}

// Painting is a picture painted on a side of a block of rock, in a paint.
type Painting struct {
	Side
	Picture Picture `json:"picture"`
	Paint   Item    `json:"paint"`
}

// Paintable reports whether a block is rock that can be painted on.
func Paintable(b Block) bool {
	switch b {
	case Stone, Cobble, Redrock, Sandstone, Basalt, ChalkSeam, Deepstone:
		return true
	}
	return false
}

// IsPaint reports whether an item is a paint, which paints in its color.
func (i Item) IsPaint() bool {
	return i == RedPaint || i == BlackPaint || i == WhitePaint
}

// valid reports whether the painting is one this game can make.
func (p Painting) valid() bool {
	return p.Cell.valid() && p.Face <= FaceZPos && p.Picture < Pictures && p.Paint.IsPaint()
}

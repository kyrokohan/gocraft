package world

// Picture is what a painting shows.
type Picture uint8

// The pictures, in the order a save numbers them.
const (
	Hand     Picture = iota // a stencil: the paint blown round a hand, which is left bare
	Spiral                  // a line that winds out from the middle
	Pictures                // how many there are
)

// Name returns what the picture is called.
func (p Picture) Name() string {
	return [Pictures]string{"hand", "spiral"}[p]
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

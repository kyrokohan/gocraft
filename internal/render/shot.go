package render

import (
	"image"
	imagecolor "image/color"

	"glonk.dev/engine"
)

// Picture returns the pixels of a canvas as a picture, each one scale pixels across, so
// that a frame can be saved and looked at without a terminal. Empty pixels are black.
// Text and anything else drawn into the cells of the frame is not in it.
func Picture(c *engine.Canvas, scale int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, c.Width()*scale, c.Height()*scale))
	for y := range c.Height() * scale {
		for x := range c.Width() * scale {
			px := imagecolor.NRGBA{A: 255}
			if p := c.At(x/scale, y/scale); !p.IsDefault() {
				px.R, px.G, px.B = p.RGB()
			}
			img.SetNRGBA(x, y, px)
		}
	}
	return img
}

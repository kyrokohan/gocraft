package world

import "glonk.dev/geometry"

// The size of the world in blocks. x and z run along the ground and y runs up. The sizes
// are powers of two, and fixed, so that finding a block is one index into one list: a
// frame looks up millions.
const (
	SizeX = 256
	SizeY = 64
	SizeZ = 256

	// SeaLevel is the highest layer that holds water.
	SeaLevel = 20
)

// World is the grid of blocks. Outside it there is only air.
type World struct {
	// Seed is the number the terrain was made from.
	Seed int64

	blocks []Block // indexed (y*SizeZ+z)*SizeX + x
	height int     // one above the highest layer that has ever held a block
}

// New returns a world of air.
func New() *World {
	return &World{blocks: make([]Block, SizeX*SizeY*SizeZ)}
}

func inside(x, y, z int) bool {
	return uint(x) < SizeX && uint(y) < SizeY && uint(z) < SizeZ
}

func index(x, y, z int) int {
	return (y*SizeZ+z)*SizeX + x
}

// At returns the block at a cell, which is air outside the world.
func (w *World) At(x, y, z int) Block {
	if !inside(x, y, z) {
		return Air
	}
	return w.blocks[index(x, y, z)]
}

// Set puts a block in a cell. Outside the world it does nothing.
func (w *World) Set(x, y, z int, b Block) {
	if !inside(x, y, z) {
		return
	}
	w.blocks[index(x, y, z)] = b
	if b != Air && y >= w.height {
		w.height = y + 1
	}
}

// Height returns a layer above which there is only air: one above the highest layer
// that has ever held a block. Breaking that block does not lower it.
func (w *World) Height() int {
	return w.height
}

// Surface returns the layer of the highest block in a column, or -1 if it is all air.
func (w *World) Surface(x, z int) int {
	for y := SizeY - 1; y >= 0; y-- {
		if w.At(x, y, z) != Air {
			return y
		}
	}
	return -1
}

// Bytes returns a copy of the blocks in the order they are kept, one byte each.
func (w *World) Bytes() []byte {
	out := make([]byte, len(w.blocks))
	for i, b := range w.blocks {
		out[i] = byte(b)
	}
	return out
}

// Spawn returns where a new player stands: on dry ground with room above, as near the
// middle of the world as there is any. The point is at the player's feet.
func (w *World) Spawn() geometry.Vector3 {
	cx, cz := SizeX/2, SizeZ/2
	for reach := range SizeX / 2 {
		for z := cz - reach; z <= cz+reach; z++ {
			for x := cx - reach; x <= cx+reach; x++ {
				if max(abs(x-cx), abs(z-cz)) != reach {
					continue // only the ring at this distance; the inside was looked at before
				}
				y := w.Surface(x, z)
				switch w.At(x, y, z) {
				case Grass, Sand, Snow, Stone:
					if y+2 < SizeY {
						return geometry.Vector3{X: float64(x) + 0.5, Y: float64(y + 1), Z: float64(z) + 0.5}
					}
				}
			}
		}
	}
	// No dry ground anywhere: stand on top of the middle column, whatever it is.
	return geometry.Vector3{X: float64(cx) + 0.5, Y: float64(w.Surface(cx, cz) + 1), Z: float64(cz) + 0.5}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

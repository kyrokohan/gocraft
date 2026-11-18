package world

import (
	"math"

	"glonk.dev/geometry"
)

// The world has no end along the ground: x and z run on for ever, and y runs up from
// the bottom to Height. It is made and kept a chunk at a time, a column of blocks
// ChunkSize square and Height tall.
const (
	ChunkSize = 16
	Height    = 128

	// SeaLevel is the highest layer that holds water.
	SeaLevel = 48

	chunkBlocks = ChunkSize * ChunkSize * Height
)

// Only the part of the world round the player is held, in a window that slides with
// it. The window is one flat list, because drawing looks up millions of blocks a frame
// and each has to be one index: a block at x, z is kept at x and z cut down to the size
// of the window, so the list wraps round and nothing is copied when the window moves.
const (
	windowChunks = 16                       // chunks along each side of the window
	windowSize   = windowChunks * ChunkSize // blocks along each side: 256
	windowMask   = windowSize - 1

	// squareRadius is how many chunks from the middle chunk the window answers for: a
	// square 15 chunks across, one less than the window, so that no two chunks of the
	// square share a place in the list.
	squareRadius = 7

	// LoadRadius is how many chunks from the middle chunk are kept loaded: 96 blocks
	// and more every way, which is farther than anything is drawn.
	LoadRadius = 6
)

// Generator makes one chunk of a world: it fills in the blocks of the chunk at (cx, cz)
// in chunk coordinates. The blocks are all air when it is called, and the block at x,
// y, z inside the chunk is blocks[(y*ChunkSize+z)*ChunkSize+x]. It must make the same
// chunk every time it is asked, and must not need any other chunk to be there.
type Generator func(cx, cz int, blocks []Block)

// World is the blocks round one place in a world without end. Follow moves the place,
// and Load brings in the chunks round it, from where they were saved if the player ever
// changed them and from the Generator if not.
type World struct {
	// Seed is the number the terrain is made from.
	Seed int64

	gen    Generator
	blocks []Block // the window: indexed y<<16 | (z&255)<<8 | x&255
	height int     // one above the highest layer that has ever held a block

	cx, cz int                              // the chunk in the middle of the window
	owner  [windowChunks * windowChunks]key // the chunk each place in the window holds, if it is loaded
	loaded [windowChunks * windowChunks]bool
	dirty  [windowChunks * windowChunks]bool // changed since it was loaded or saved
	held   [windowChunks * windowChunks]bool // not to be saved: its file could not be read, and is left as it is

	stash map[key][]byte // changed chunks that have left the window, packed, until they are saved
	dir   string         // the folder the world was opened from, where saved chunks are read from; "" for none
	fault error          // the first chunk file that could not be read, for the game to tell the player

	scratch []Block // one chunk, for making and unpacking
}

// key names a chunk by its coordinates.
type key struct{ x, z int }

// New returns a world of air with no end, for tests and for building in.
func New() *World {
	return newWorld(0, nil, "", 0, 0)
}

// newWorld returns a world with the chunks round the column at x, z loaded: made by
// gen, but for those saved in the folder dir, if one is named.
func newWorld(seed int64, gen Generator, dir string, x, z float64) *World {
	w := &World{
		Seed:    seed,
		gen:     gen,
		dir:     dir,
		blocks:  make([]Block, windowSize*windowSize*Height),
		stash:   make(map[key][]byte),
		scratch: make([]Block, chunkBlocks),
	}
	w.Follow(x, z)
	w.LoadAll()
	return w
}

// chunkOf returns the chunk a block coordinate is in. Shifting rounds down, so it is
// right for negative coordinates too.
func chunkOf(v int) int {
	return v >> 4
}

func place(k key) int {
	return (k.z&(windowChunks-1))<<4 | k.x&(windowChunks-1)
}

func index(x, y, z int) int {
	return y<<16 | (z&windowMask)<<8 | x&windowMask
}

// inSquare reports whether a chunk is one the window answers for.
func (w *World) inSquare(k key) bool {
	return k.x-w.cx >= -squareRadius && k.x-w.cx <= squareRadius &&
		k.z-w.cz >= -squareRadius && k.z-w.cz <= squareRadius
}

// At returns the block at a cell. It is air above and below the world, and wherever
// the world is not loaded: far from the place the window follows.
func (w *World) At(x, y, z int) Block {
	if uint(y) >= Height || !w.inSquare(key{chunkOf(x), chunkOf(z)}) {
		return Air
	}
	// A chunk of the square that is not loaded is all air in the list, so it needs no
	// check of its own. That is what lets a ray read the list without asking.
	return w.blocks[index(x, y, z)]
}

// Set puts a block in a cell. It does nothing above and below the world, or where the
// world is not loaded.
func (w *World) Set(x, y, z int, b Block) {
	k := key{chunkOf(x), chunkOf(z)}
	if uint(y) >= Height || !w.inSquare(k) || !w.loaded[place(k)] {
		return
	}
	w.blocks[index(x, y, z)] = b
	w.dirty[place(k)] = true
	if b != Air && y >= w.height {
		w.height = y + 1
	}
}

// Loaded reports whether the chunk that holds the column at x, z is loaded.
func (w *World) Loaded(x, z int) bool {
	k := key{chunkOf(x), chunkOf(z)}
	return w.inSquare(k) && w.loaded[place(k)]
}

// Height returns a layer above which there is only air: one above the highest layer
// that has ever held a block in a loaded chunk. It never comes down.
func (w *World) Height() int {
	return w.height
}

// Surface returns the layer of the highest block in a column, or -1 if it is all air.
func (w *World) Surface(x, z int) int {
	for y := Height - 1; y >= 0; y-- {
		if w.At(x, y, z) != Air {
			return y
		}
	}
	return -1
}

// Fault returns the first problem met in reading a saved chunk, or nil. Such a chunk is
// made afresh from the seed and its file is left as it is; what is built there is not
// saved.
func (w *World) Fault() error {
	return w.fault
}

// Follow moves the middle of the window to the chunk that holds the column at x, z:
// where the player is. Chunks that are now too far away leave the window, and those
// the player changed are kept aside until the world is saved. Call Load afterwards to
// bring in the chunks that are now near.
func (w *World) Follow(x, z float64) {
	cx, cz := chunkOf(int(math.Floor(x))), chunkOf(int(math.Floor(z)))
	if cx == w.cx && cz == w.cz {
		return
	}
	w.cx, w.cz = cx, cz
	for at := range w.loaded {
		if w.loaded[at] && !w.inSquare(w.owner[at]) {
			w.evict(at)
		}
	}
}

// evict takes a chunk out of the window. Its place in the list is cleared to air, which
// is what a place that is not loaded has to hold.
func (w *World) evict(at int) {
	k := w.owner[at]
	if w.dirty[at] && !w.held[at] {
		w.stash[k] = pack(w.copyOut(k))
	}
	for y := range Height {
		for z := range ChunkSize {
			row := index(k.x*ChunkSize, y, k.z*ChunkSize+z)
			clear(w.blocks[row : row+ChunkSize])
		}
	}
	w.loaded[at], w.dirty[at], w.held[at] = false, false, false
}

// copyOut copies a loaded chunk out of the window into the scratch chunk.
func (w *World) copyOut(k key) []Block {
	for y := range Height {
		for z := range ChunkSize {
			row := index(k.x*ChunkSize, y, k.z*ChunkSize+z)
			copy(w.scratch[(y*ChunkSize+z)*ChunkSize:], w.blocks[row:row+ChunkSize])
		}
	}
	return w.scratch
}

// Load brings in up to n of the chunks within LoadRadius of the middle that are not
// loaded yet, the nearest first, and returns how many are still to come. Making a chunk
// takes time, so a game loads a few each frame as the player moves.
func (w *World) Load(n int) (left int) {
	for ring := 0; ring <= LoadRadius; ring++ {
		for dz := -ring; dz <= ring; dz++ {
			for dx := -ring; dx <= ring; dx++ {
				if max(abs(dx), abs(dz)) != ring {
					continue // only the ring at this distance: the inside was looked at before
				}
				k := key{w.cx + dx, w.cz + dz}
				if w.loaded[place(k)] {
					continue
				}
				if n <= 0 {
					left++
					continue
				}
				w.load(k)
				n--
			}
		}
	}
	return left
}

// LoadAll brings in every chunk within LoadRadius of the middle.
func (w *World) LoadAll() {
	w.Load(math.MaxInt)
}

// load brings one chunk into the window: as the player left it, if it was ever changed,
// and from the Generator if not.
func (w *World) load(k key) {
	at := place(k)
	chunk, held := w.source(k)
	for y := range Height {
		for z := range ChunkSize {
			row := index(k.x*ChunkSize, y, k.z*ChunkSize+z)
			copy(w.blocks[row:row+ChunkSize], chunk[(y*ChunkSize+z)*ChunkSize:(y*ChunkSize+z+1)*ChunkSize])
		}
	}
	w.owner[at], w.loaded[at], w.dirty[at], w.held[at] = k, true, false, held

	for i := len(chunk) - 1; i >= 0; i-- {
		if chunk[i] != Air {
			w.height = max(w.height, i/(ChunkSize*ChunkSize)+1)
			break
		}
	}
}

// source returns the blocks of a chunk, in the scratch chunk: from the chunks set aside
// since the last save, from the folder the world was opened from, or from the
// Generator. held is true if the chunk's file is there and could not be read.
func (w *World) source(k key) (chunk []Block, held bool) {
	if packed, ok := w.stash[k]; ok {
		if err := unpack(packed, w.scratch); err == nil {
			return w.scratch, false
		}
		// What this program packed a moment ago cannot fail to unpack; if it somehow
		// does, the chunk is made afresh.
	} else if w.dir != "" {
		switch err := readChunk(w.dir, k, w.scratch); {
		case err == nil:
			return w.scratch, false
		case !isNotExist(err):
			if w.fault == nil {
				w.fault = err
			}
			held = true
		}
	}

	clear(w.scratch)
	if w.gen != nil {
		w.gen(k.x, k.z, w.scratch)
	}
	return w.scratch, held
}

// Spawn returns where a new player stands: on dry ground with room above, as near the
// origin of the world as there is any among the chunks that are loaded. The point is at
// the player's feet.
func (w *World) Spawn() geometry.Vector3 {
	reach := (LoadRadius - 1) * ChunkSize
	for ring := range reach {
		for z := -ring; z <= ring; z++ {
			for x := -ring; x <= ring; x++ {
				if max(abs(x), abs(z)) != ring {
					continue
				}
				y := w.Surface(x, z)
				switch w.At(x, y, z) {
				case Grass, Sand, Snow, Stone:
					if y+2 < Height {
						return geometry.Vector3{X: float64(x) + 0.5, Y: float64(y + 1), Z: float64(z) + 0.5}
					}
				}
			}
		}
	}
	// No dry ground anywhere near: stand on top of the column at the origin, whatever it is.
	return geometry.Vector3{X: 0.5, Y: float64(w.Surface(0, 0) + 1), Z: 0.5}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

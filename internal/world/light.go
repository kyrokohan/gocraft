package world

import "slices"

// Light is how much light there is in a cell: a level from 0 to MaxLight that comes from
// the sky, in the high four bits, and one from blocks that glow, in the low four. The two
// are kept apart because the sky's light changes with the hour and a torch's does not:
// how bright a level looks is for whoever draws it to say.
//
// The sky's light comes straight down through the air at its full level, and a glowing
// block's is at the block's Glow. From there light loses a level for each step it takes
// to the next cell, any way but that straight drop, and a block with Shade takes that
// much from what passes through it, a level at least: a block whose Shade is MaxLight
// lets none in. The light of the window is worked out from its blocks as chunks come in
// and as blocks change, and is not saved.
type Light uint8

// MaxLight is the highest level of light: the open sky's.
const MaxLight = 15

// Sky returns the level of the light that comes from the sky.
func (l Light) Sky() int { return int(l >> 4) }

// Block returns the level of the light that comes from blocks that glow.
func (l Light) Block() int { return int(l & MaxLight) }

// skyLight is the light of a cell open to the sky: the sky's at its full level, and no other.
const skyLight = Light(MaxLight << 4)

// allLight keeps both parts of a Light, for push.
const allLight Light = 0xFF

// lightReach is how far a change to one cell can change the light round it: a level of
// light that has come that far from the cell has nothing left to give.
const lightReach = MaxLight - 1

// shade and glow are Info.Shade and Info.Glow of each kind of block, kept apart for the
// loops that light the world, which look at them millions of times.
var shade, glow = func() (shade, glow [256]uint8) {
	for b := range infos {
		shade[b], glow[b] = min(infos[b].Shade, MaxLight), min(infos[b].Glow, MaxLight)
	}
	return shade, glow
}()

// gives reports whether a cell has light to give the cells beside it: a level of 1 has
// none left after a step.
func gives(l Light) bool {
	return l > 1<<4|MaxLight || l&MaxLight > 1
}

// lightBox is a box of cells whose light may be out of date because blocks changed.
type lightBox struct {
	x0, x1, y0, y1, z0, z1 int
	any                    bool
}

// add takes in the cells whose light can change when the block at x, y, z changes: those
// within lightReach of it, and of the cells under it down to low, whose light from the
// sky changes with it.
func (b *lightBox) add(x, low, y, z int) {
	x0, x1 := x-lightReach, x+lightReach
	z0, z1 := z-lightReach, z+lightReach
	y0, y1 := max(low-lightReach, 0), min(y+lightReach, Height-1)
	if !b.any {
		*b = lightBox{x0, x1, y0, y1, z0, z1, true}
		return
	}
	b.x0, b.x1 = min(b.x0, x0), max(b.x1, x1)
	b.y0, b.y1 = min(b.y0, y0), max(b.y1, y1)
	b.z0, b.z1 = min(b.z0, z0), max(b.z1, z1)
}

// lamp is a light that is carried: it glows in the cell it is in as a block that glows
// does, and is no block. A glow of 0 is no lamp.
type lamp struct {
	x, y, z int
	glow    uint8
}

// Carry puts the light that is carried in a cell, glowing at a level up to MaxLight, in
// the place of wherever it was before: there is one. A glow of 0 puts it out, and so
// does a cell above or below the world. The light round where it was and where it is
// now is worked out again, when it is next asked for.
func (w *World) Carry(x, y, z int, glow uint8) {
	to := lamp{x, y, z, min(glow, MaxLight)}
	if to.glow == 0 || uint(y) >= Height {
		to = lamp{}
	}
	if to == w.lamp {
		return
	}
	for _, l := range [2]lamp{w.lamp, to} {
		if l.glow > 0 {
			w.stale.add(l.x, l.y, l.y, l.z)
		}
	}
	w.lamp = to
}

// LightAt returns the light in a cell. Above the world it is the open sky's, and below
// the world, and wherever the world is not loaded, it is dark.
func (w *World) LightAt(x, y, z int) Light {
	if y >= Height {
		return skyLight
	}
	if y < 0 || !w.inSquare(key{chunkOf(x), chunkOf(z)}) {
		return 0
	}
	w.settle()
	return w.light[index(x, y, z)]
}

// settle brings the light up to date with the blocks that have changed since it was
// last worked out. Changes are gathered and lit together, as a lot of them come at once
// when something is built.
func (w *World) settle() {
	if w.stale.any {
		w.relight()
	}
}

// holds reports whether the chunk k is in the window.
func (w *World) holds(k key) bool {
	return w.inSquare(k) && w.loaded[place(k)]
}

// changed notes that the block at x, y, z is to be b in the place of old, for the light
// to be worked out again round it. It is called before the block is put in.
func (w *World) changed(x, y, z int, old, b Block) {
	if shade[old] == shade[b] && glow[old] == glow[b] {
		return // the light is the same either way
	}
	w.stale.add(x, w.shadowed(x, y, z, old, b), y, z)
}

// shadowed returns the lowest layer of the column at x, z whose light straight from the
// sky changes when the block at y goes from old to b: y itself, if none under it does.
func (w *World) shadowed(x, y, z int, old, b Block) int {
	level := MaxLight
	for above := Height - 1; above > y; above-- {
		level = max(level-int(shade[w.blocks[index(x, above, z)]]), 0)
	}
	was, will, low := level, level, y
	for at := y; at >= 0; at-- {
		before, after := w.blocks[index(x, at, z)], w.blocks[index(x, at, z)]
		if at == y {
			before, after = old, b
		}
		was, will = max(was-int(shade[before]), 0), max(will-int(shade[after]), 0)
		if was == will {
			break // the blocks under are the same, so the light under is too
		}
		low = at
	}
	return low
}

// skyDown works out the light of the cells from the layer y0 up to y1 of the loaded
// columns from x0 to x1 and z0 to z1, as the sky and the blocks there give it before it
// spreads: the sky's light down to the first block that lets none through, dimmed by
// those that take some, and the glow of each block in its own cell. The cells under the
// layer below that have light to give are put in the queue to spread it.
//
// It goes a layer at a time and not a column at a time, as the cells of a layer lie
// together in the window and those of a column far apart.
func (w *World) skyDown(x0, x1, z0, z1, y0, y1, below int) {
	across := x1 - x0 + 1
	w.levels = slices.Grow(w.levels[:0], across*(z1-z0+1))[:across*(z1-z0+1)]
	top := 0 // every loaded column is air from here up
	for z := z0; z <= z1; z++ {
		for x := x0; x <= x1; x++ {
			level := int8(-1) // for a column that is not loaded, and is left alone
			if k := (key{chunkOf(x), chunkOf(z)}); w.holds(k) {
				level, top = MaxLight, max(top, w.tops[place(k)])
			}
			w.levels[(z-z0)*across+x-x0] = level
		}
	}

	for y := Height - 1; y >= y0; y-- {
		open := y >= top
		if open && y > y1 {
			continue // all air, and nothing to write
		}
		for z := z0; z <= z1; z++ {
			row := w.levels[(z-z0)*across : (z-z0+1)*across]
			for x := x0; x <= x1; x++ {
				level := row[x-x0]
				if level < 0 {
					continue
				}
				i := index(x, y, z)
				l := skyLight
				if !open {
					b := w.blocks[i]
					level = max(level-int8(shade[b]), 0)
					row[x-x0] = level
					l = Light(level)<<4 | Light(glow[b])
				}
				if y > y1 {
					continue
				}
				w.light[i] = l
				if y < below && gives(l) {
					w.queue = append(w.queue, int32(i))
				}
			}
		}
	}

	// The light that is carried glows in its cell, whether that is under the blocks or
	// in the open air over them.
	if l := w.lamp; l.glow > 0 && l.x >= x0 && l.x <= x1 && l.z >= z0 && l.z <= z1 &&
		l.y >= y0 && l.y <= y1 && w.Loaded(l.x, l.z) {
		i := index(l.x, l.y, l.z)
		w.light[i] = w.light[i]&^MaxLight | Light(max(w.light[i].Block(), int(l.glow)))
		w.queue = append(w.queue, int32(i))
	}
}

// push puts the cells of the column at x, z from y0 to y1 that have light to give in
// the queue, if the column is loaded. Only the parts of the light that of keeps count.
func (w *World) push(x, z, y0, y1 int, of Light) {
	if !w.Loaded(x, z) {
		return
	}
	for y := max(y0, 0); y <= min(y1, Height-1); y++ {
		if i := index(x, y, z); gives(w.light[i] & of) {
			w.queue = append(w.queue, int32(i))
		}
	}
}

// lightChunk lights a chunk that has just come into the window, and the chunks round it
// that were there before it: its light spreads into theirs, and theirs into it.
func (w *World) lightChunk(k key) {
	x0, z0 := k.x*ChunkSize, k.z*ChunkSize
	// Above the highest block of the chunk and of those beside it, every cell has the
	// open sky above it and beside it: it has nothing to give that its neighbors lack.
	top := w.tops[place(k)]
	for _, d := range [4]key{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
		if n := (key{k.x + d.x, k.z + d.z}); w.holds(n) {
			top = max(top, w.tops[place(n)])
		}
	}
	w.skyDown(x0, x0+ChunkSize-1, z0, z0+ChunkSize-1, 0, Height-1, top)
	// The cells along the chunk's edges in the chunks beside it. Above the tops they have
	// the open sky's light, as the chunk's own cells do there, but they can have the light
	// of a glowing block too, come over low ground from a chunk farther off.
	for i := range ChunkSize {
		for _, c := range [4][2]int{{x0 - 1, z0 + i}, {x0 + ChunkSize, z0 + i}, {x0 + i, z0 - 1}, {x0 + i, z0 + ChunkSize}} {
			w.push(c[0], c[1], 0, top-1, allLight)
			w.push(c[0], c[1], top, Height-1, MaxLight)
		}
	}
	w.spread()
}

// relight works the light out again in the cells that blocks changed round. The light
// of the cells just outside them is right, so it is spread back into them, along with
// what the sky and the glowing blocks give inside them.
func (w *World) relight() {
	b := w.stale
	w.stale = lightBox{}
	w.skyDown(b.x0, b.x1, b.z0, b.z1, b.y0, b.y1, Height)
	for z := b.z0; z <= b.z1; z++ {
		w.push(b.x0-1, z, b.y0, b.y1, allLight)
		w.push(b.x1+1, z, b.y0, b.y1, allLight)
		for x := b.x0; x <= b.x1; x++ {
			w.push(x, z, b.y0-1, b.y0-1, allLight)
			w.push(x, z, b.y1+1, b.y1+1, allLight)
		}
	}
	for x := b.x0; x <= b.x1; x++ {
		w.push(x, b.z0-1, b.y0, b.y1, allLight)
		w.push(x, b.z1+1, b.y0, b.y1, allLight)
	}
	w.spread()
}

// spread passes the light of the cells in the queue on to their neighbors, and theirs on
// to their own, as far as it goes. Light only ever goes up here, so the order the cells
// are taken in does not change where it ends.
func (w *World) spread() {
	for head := 0; head < len(w.queue); head++ {
		i := int(w.queue[head])
		l := w.light[i]
		sky, block := l.Sky(), l.Block()
		x, z := i&windowMask, i>>8&windowMask
		// A step out of the chunk goes into the chunk beside it, which may not be loaded.
		k := w.owner[(z>>4)<<4|x>>4]
		if x&(ChunkSize-1) != ChunkSize-1 || w.holds(key{k.x + 1, k.z}) {
			w.give(i&^windowMask|(x+1)&windowMask, sky, block)
		}
		if x&(ChunkSize-1) != 0 || w.holds(key{k.x - 1, k.z}) {
			w.give(i&^windowMask|(x-1)&windowMask, sky, block)
		}
		if z&(ChunkSize-1) != ChunkSize-1 || w.holds(key{k.x, k.z + 1}) {
			w.give(i&^(windowMask<<8)|(z+1)&windowMask<<8, sky, block)
		}
		if z&(ChunkSize-1) != 0 || w.holds(key{k.x, k.z - 1}) {
			w.give(i&^(windowMask<<8)|(z-1)&windowMask<<8, sky, block)
		}
		if i>>16 < Height-1 {
			w.give(i+1<<16, sky, block)
		}
		if i>>16 > 0 {
			w.give(i-1<<16, sky, block)
		}
	}
	w.queue = w.queue[:0]
}

// give lights the cell n from a neighbor with the given levels, if that makes it
// brighter, and puts it in the queue to pass the light on.
func (w *World) give(n, sky, block int) {
	s := int(shade[w.blocks[n]])
	if s >= MaxLight {
		return
	}
	step := max(s, 1)
	sky, block = sky-step, block-step
	old := w.light[n]
	if sky <= old.Sky() && block <= old.Block() {
		return
	}
	w.light[n] = Light(max(sky, old.Sky())<<4 | max(block, old.Block()))
	w.queue = append(w.queue, int32(n))
}

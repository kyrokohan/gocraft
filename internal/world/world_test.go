package world

import (
	"bytes"
	"testing"
)

func TestSetAndAt(t *testing.T) {
	w := New()

	w.Set(1, 2, 3, Brick)

	if got := w.At(1, 2, 3); got != Brick {
		t.Errorf("got %v, wanted Brick", got)
	}
	if got := w.At(3, 2, 1); got != Air {
		t.Errorf("got %v, wanted Air", got)
	}
}

func TestOutsideTheWorldIsAirAndCannotBeSet(t *testing.T) {
	w := New()

	for _, p := range [][3]int{
		{-1, 0, 0}, {0, -1, 0}, {0, 0, -1},
		{SizeX, 0, 0}, {0, SizeY, 0}, {0, 0, SizeZ},
	} {
		w.Set(p[0], p[1], p[2], Stone)
		if got := w.At(p[0], p[1], p[2]); got != Air {
			t.Errorf("At(%v) is %v, wanted Air", p, got)
		}
	}
}

func TestHeightIsAboveTheHighestBlock(t *testing.T) {
	w := New()
	if got := w.Height(); got != 0 {
		t.Errorf("an empty world is %d high, wanted 0", got)
	}

	w.Set(4, 30, 4, Stone)
	w.Set(4, 7, 4, Stone)

	if got := w.Height(); got != 31 {
		t.Errorf("got %d, wanted 31", got)
	}
}

func TestSurfaceIsTheHighestBlockOfAColumn(t *testing.T) {
	w := New()
	w.Set(2, 5, 2, Stone)
	w.Set(2, 9, 2, Leaves)

	if got := w.Surface(2, 2); got != 9 {
		t.Errorf("got %d, wanted 9", got)
	}
	if got := w.Surface(3, 3); got != -1 {
		t.Errorf("an empty column is %d, wanted -1", got)
	}
}

func TestSameSeedSameWorld(t *testing.T) {
	a, b, c := Generate(7), Generate(7), Generate(8)

	if !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Error("the same seed made two different worlds")
	}
	if bytes.Equal(a.Bytes(), c.Bytes()) {
		t.Error("two seeds made the same world")
	}
}

func TestGeneratedWorldHasGroundWaterAndTrees(t *testing.T) {
	w := Generate(1)

	count := map[Block]int{}
	for z := range SizeZ {
		for x := range SizeX {
			if got := w.At(x, 0, z); got != Deepstone {
				t.Fatalf("the bottom at %d,%d is %v, wanted Deepstone", x, z, got)
			}
			if w.At(x, SizeY-1, z) != Air {
				t.Fatalf("the top layer at %d,%d is not air", x, z)
			}
			for y := 1; y < SizeY; y++ {
				count[w.At(x, y, z)]++
			}
		}
	}
	for _, b := range []Block{Grass, Dirt, Stone, Sand, Water, Log, Leaves} {
		if count[b] == 0 {
			t.Errorf("the world has no %v", b.Info().Name)
		}
	}
	if count[Deepstone] != 0 {
		t.Errorf("deepstone above the bottom layer: %d blocks", count[Deepstone])
	}
}

func TestSpawnStandsOnDryGroundWithRoomAbove(t *testing.T) {
	for seed := int64(1); seed <= 5; seed++ {
		w := Generate(seed)

		p := w.Spawn()

		x, y, z := int(p.X), int(p.Y), int(p.Z)
		if !w.At(x, y-1, z).Info().Solid {
			t.Errorf("seed %d: spawn at %v does not stand on solid ground", seed, p)
		}
		if w.At(x, y, z) != Air || w.At(x, y+1, z) != Air {
			t.Errorf("seed %d: spawn at %v has no room for the player", seed, p)
		}
	}
}

package world

import "testing"

// pigments counts, in the loaded chunks round a column, the ochre and where it lies, the
// chalk near the top of the land, and the flint seams that touch chalk.
func pigments(w *World, x0, z0 int, seed uint32) (ochre, outside, chalk, flint int) {
	span := (LoadRadius - 1) * ChunkSize
	for z := z0 - span; z < z0+span; z++ {
		for x := x0 - span; x < x0+span; x++ {
			l := landAt(x, z, seed)
			for y := 1; y <= l.top; y++ {
				switch w.At(x, y, z) {
				case OchreSeam:
					if ochre++; l.biome != Badlands {
						outside++
					}
				case ChalkSeam:
					chalk++
					for _, d := range [][3]int{{1, 0, 0}, {-1, 0, 0}, {0, 1, 0}, {0, -1, 0}, {0, 0, 1}, {0, 0, -1}} {
						if w.At(x+d[0], y+d[1], z+d[2]) == FlintSeam {
							flint++
							break
						}
					}
				}
			}
		}
	}
	return ochre, outside, chalk, flint
}

func TestOchreLiesInTheBadlandsAndChalkWithFlintInItUnderTheLand(t *testing.T) {
	for seed := int64(1); seed <= 10; seed++ {
		w, x, z := loadedAt(t, seed, Badlands)
		ochre, outside, chalk, flint := pigments(w, x, z, uint32(seed))
		if ochre < 20 || outside > 0 {
			t.Errorf("seed %d: %d blocks of ochre round the badlands, %d of them outside", seed, ochre, outside)
		}
		if chalk < 50 || flint < 3 {
			t.Errorf("seed %d: %d blocks of chalk round the badlands, %d with flint by them", seed, chalk, flint)
		}
		if seed == 1 {
			again := newWorld(seed, Terrain(seed), "", float64(x), float64(z))
			if o, _, c, _ := pigments(again, x, z, uint32(seed)); o != ochre || c != chalk {
				t.Errorf("seed 1 made %d ochre and %d chalk, and again %d and %d", ochre, chalk, o, c)
			}
		}
	}
	// Chalk is not of the badlands only.
	w, x, z := loadedAt(t, 7, Prairie)
	if _, _, chalk, _ := pigments(w, x, z, 7); chalk < 50 {
		t.Errorf("%d blocks of chalk round the prairie", chalk)
	}
}

func TestOchreAndChalkBreakIntoWhatTheyAre(t *testing.T) {
	for b, want := range map[Block]Item{OchreSeam: Ochre, ChalkSeam: Chalk} {
		if item, n := b.Drop(); item != want || n != 1 {
			t.Errorf("%s drops %d of %q, wanted one %q", b.Info().Name, n, item.Name(), want.Name())
		}
		if !b.Info().Solid || b.Info().Tool != Pick {
			t.Errorf("%s is not solid rock that a pick breaks", b.Info().Name)
		}
	}
}

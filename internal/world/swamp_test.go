package world

import "testing"

func TestTheCoalSwampLiesLowAndWetWithPoolsAndScaleTrees(t *testing.T) {
	const seed = 7
	w, x0, z0 := loadedAt(t, seed, Swamp)
	s := uint32(seed)
	span := (LoadRadius - 1) * ChunkSize
	columns, pools, coal, horsetails := 0, 0, 0, 0
	trunks := map[[2]int]int{}
	for z := z0 - span; z < z0+span; z++ {
		for x := x0 - span; x < x0+span; x++ {
			l := landAt(x, z, s)
			if l.biome != Swamp {
				continue
			}
			if again := landAt(x, z, s); again != l {
				t.Fatalf("the land at %d, %d was asked twice, and the answers differ", x, z)
			}
			columns++
			if l.top < SeaLevel-2 || l.top > SeaLevel+2 {
				t.Fatalf("the swamp at %d, %d has its ground at %d, and the sea is at %d: wanted it within two of that", x, z, l.top, SeaLevel)
			}
			if l.top < SeaLevel {
				// A pool: still water to the level of the sea, over peat.
				pools++
				if top, floor := w.At(x, SeaLevel, z), w.At(x, l.top, z); top != Water || floor != Peat || w.At(x, SeaLevel+1, z).Info().Solid {
					t.Fatalf("the pool at %d, %d is %s over %s", x, z, top.Info().Name, floor.Info().Name)
				}
			} else if on := w.At(x, l.top, z); on != Peat && on != Air { // air: the mouth of a cave
				t.Fatalf("the ground of the swamp at %d, %d is %s, wanted peat", x, z, on.Info().Name)
			}
			for y := l.top - 3; y < l.top; y++ {
				if w.At(x, y, z) == CoalOre {
					coal++
				}
			}
			for y := l.top + 1; y < Height; y++ {
				switch w.At(x, y, z) {
				case ScaleTrunk:
					trunks[[2]int{x, z}]++
				case Horsetail:
					horsetails++
				case PineLog, RedwoodLog, FernTrunk, StumpPalm, Snow:
					t.Fatalf("the swamp holds a %s at %d,%d,%d", w.At(x, y, z).Info().Name, x, y, z)
				}
			}
		}
	}
	if columns < 1500 {
		t.Fatalf("only %d columns of swamp were looked at", columns)
	}
	if part := float64(pools) / float64(columns); part < 0.15 || part > 0.5 {
		t.Errorf("%.0f%% of the swamp is pools, wanted a fifth to a half", 100*part)
	}
	if coal*40 < columns || horsetails*30 < columns {
		t.Errorf("%d columns of swamp hold %d coal near the top and %d horsetails: wanted both common", columns, coal, horsetails)
	}
	if len(trunks) < 8 {
		t.Fatalf("%d scale trees stand on %d columns of swamp", len(trunks), columns)
	}
	for at, tall := range trunks {
		x, z := at[0], at[1]
		l := landAt(x, z, s)
		if tall < 7 || tall > 13 || l.top < SeaLevel {
			t.Errorf("the scale tree at %d, %d has a trunk of %d and its foot at %d: wanted 7 to 13, on dry ground", x, z, tall, l.top)
		}
		// Its trunk is bare, and its crown is at the top of it.
		top := l.top + tall
		if w.At(x+1, l.top+2, z) == ScaleCrown || w.At(x+1, top, z) != ScaleCrown || w.At(x, top+1, z) != ScaleCrown {
			t.Errorf("the scale tree at %d, %d: beside its foot is %s, beside its top %s, and over it %s", x, z,
				w.At(x+1, l.top+2, z).Info().Name, w.At(x+1, top, z).Info().Name, w.At(x, top+1, z).Info().Name)
		}
	}
}

func TestPeatIsDugWithAShovelAndAScaleTreeHoldsAmberNowAndThen(t *testing.T) {
	hand, _ := Peat.DigWith(NoItem)
	if spade, yields := Peat.DigWith(DiggingStick); !yields || spade >= hand || Peat.Info().Tool != Shovel {
		t.Errorf("peat is dug in %v s by hand and %v s with a digging stick", hand, spade)
	}
	if item, n := Peat.Drop(); item != Peat.Item() || n != 1 {
		t.Errorf("peat leaves %d of %q", n, item.Name())
	}
	if item, n := ScaleCrown.Drop(); n != 0 {
		t.Errorf("the crown of a scale tree leaves %d of %q", n, item.Name())
	}
	if ScaleTrunk.Info().Tool != Axe || !ScaleTrunk.Info().Solid || ScaleCrown.Info().Shade >= MaxLight {
		t.Errorf("a scale tree's trunk is cut with tool %d, and its crown takes %d of the light", ScaleTrunk.Info().Tool, ScaleCrown.Info().Shade)
	}
	amber := 0
	for i := range 1000 {
		if item, n := ScaleTrunk.DropAt(i, 70, -i*7); item != ScaleTrunk.Item() || n != 1 {
			t.Fatalf("a scale tree's trunk leaves %d of %q", n, item.Name())
		}
		switch item, n := ScaleTrunk.AlsoAt(i, 70, -i*7); {
		case n == 1 && item == Amber:
			amber++
		case n != 0:
			t.Fatalf("a scale tree's trunk also leaves %d of %q", n, item.Name())
		}
	}
	if amber < 40 || amber > 130 {
		t.Errorf("%d of 1,000 blocks of a scale tree's trunk held amber, wanted about one in twelve", amber)
	}
	if _, n := PineLog.AlsoAt(3, 70, 5); n > 1 {
		t.Error("a pine log holds amber")
	}
	if Amber.Name() != "amber" || !Amber.Known() || Amber != Tusk+1 || Peat != SkyIronCore+1 || ScaleCrown != Peat+2 || Swamp != Ashlands+1 {
		t.Errorf("the numbers moved: amber %d, peat %d, crown %d, swamp %d", Amber, Peat, ScaleCrown, Swamp)
	}
}

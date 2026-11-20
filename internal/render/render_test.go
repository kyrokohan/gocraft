package render

import (
	"math"
	"math/rand/v2"
	"testing"

	"glonk.dev/engine"
	"glonk.dev/geometry"
	"glonk.dev/mesh"

	"github.com/kyrokohan/skyspear/internal/assets"
	"github.com/kyrokohan/skyspear/internal/world"
)

func newRenderer(t testing.TB) *Renderer {
	t.Helper()
	set, err := assets.Load()
	if err != nil {
		t.Fatal(err)
	}
	return New(set)
}

func newCanvas(t testing.TB, cols, rows int) *engine.Canvas {
	t.Helper()
	fb, err := engine.NewFrameBuffer(cols, rows)
	if err != nil {
		t.Fatal(err)
	}
	return fb.Canvas()
}

// wall fills a slab of the world one block thick, at layer z, with one kind of block.
func wall(w *world.World, z int, b world.Block) {
	for y := range 24 {
		for x := range 24 {
			w.Set(x, y, z, b)
		}
	}
}

// noon is the sky the tests draw under, unless they say otherwise: the full light of day.
var noon = NewSky(Noon)

// facing is a camera in front of the middle of such a wall, looking at it along +z, with
// the far plane and the fog of the camera the game uses.
var facing = mesh.Camera{
	Position: geometry.Vector3{X: 12.5, Y: 12.5, Z: 2.5},
	FOV:      math.Pi / 3,
	Far:      ViewDistance,
	Fog:      &noon.Fog,
}

// A frame is worked out from three of glonk's rays. What that gives for a pixel must be
// the ray glonk gives for it, or meshes drawn by glonk would not line up with the blocks.
func TestRaysAreGlonksRaysAtEveryPixel(t *testing.T) {
	rng := rand.New(rand.NewPCG(3, 4))
	c := newCanvas(t, 200, 60)

	for i := range 100 {
		cam := mesh.Camera{
			Position: geometry.Vector3{X: rng.Float64() * 20, Y: rng.Float64() * 20, Z: rng.Float64() * 20},
			Yaw:      rng.Float64()*8 - 4,
			Pitch:    rng.Float64()*3 - 1.5,
			FOV:      0.4 + rng.Float64()*2,
		}
		if i%3 == 0 {
			cam.FOV = 0 // glonk's default
		}
		v, ok := newRays(cam, c)
		if !ok {
			t.Fatalf("case %d: no rays for %+v", i, cam)
		}

		for range 20 {
			px, py := rng.IntN(c.Width()), rng.IntN(c.Height())
			origin, want, ok := cam.RayPixels(float64(px)+0.5, float64(py)+0.5, c)
			if got := v.through(px, py); !ok || origin != v.eye || got.Distance(want) > 1e-9 {
				t.Fatalf("case %d, pixel %d,%d: got %v from %v, glonk gives %v from %v (ok %v)",
					i, px, py, got, v.eye, want, origin, ok)
			}
		}
	}
}

func TestABrokenCameraDrawsNothing(t *testing.T) {
	w := world.New()
	wall(w, 5, world.Stone)
	rend := newRenderer(t)

	for _, cam := range []mesh.Camera{
		{FOV: -1}, {FOV: math.Pi}, {FOV: math.NaN()}, {CellAspect: -2}, {CellAspect: math.Inf(1)},
	} {
		cam.Position = facing.Position
		c := newCanvas(t, 40, 20)

		rend.Draw(c, w, cam, noon)

		if !c.At(20, 20).IsDefault() {
			t.Errorf("camera %+v drew %v, wanted nothing", cam, c.At(20, 20))
		}
	}
}

func TestEmptyWorldIsAllSkyWithNoDepth(t *testing.T) {
	c := newCanvas(t, 40, 20)

	newRenderer(t).Draw(c, world.New(), facing, noon)

	for y := range c.Height() {
		for x := range c.Width() {
			r, _, b := c.At(x, y).RGB()
			if c.At(x, y).IsDefault() || b <= r {
				t.Fatalf("pixel %d,%d is %v, wanted sky", x, y, c.At(x, y))
			}
			// The sky has no depth, so anything with one can still be drawn over it.
			if !c.SetDepth(x, y, 1e9, engine.RGB(1, 2, 3)) {
				t.Fatalf("pixel %d,%d of the sky has a depth", x, y)
			}
		}
	}
}

func TestAWallFillsTheViewWithItsTile(t *testing.T) {
	w := world.New()
	wall(w, 5, world.Stone)
	c := newCanvas(t, 40, 20)

	newRenderer(t).Draw(c, w, facing, noon)

	for y := range c.Height() {
		for x := range c.Width() {
			r, g, b := c.At(x, y).RGB()
			// Stone is gray, a little blue, and a side face is drawn darker than the tile.
			if int(b)-int(r) > 14 || int(b) < int(r) || g < 40 || g > 110 {
				t.Fatalf("pixel %d,%d is %v, wanted the gray of a stone side", x, y, c.At(x, y))
			}
		}
	}
}

func TestTheTopOfABlockIsBrighterThanItsSide(t *testing.T) {
	w := world.New()
	for z := range 24 {
		for x := range 24 {
			w.Set(x, 10, z, world.Snow)
		}
	}
	wall(w, 20, world.Snow) // far enough that the lowest rays land on the floor first
	c := newCanvas(t, 40, 20)
	cam := facing
	cam.Position = geometry.Vector3{X: 12.5, Y: 12.5, Z: 9.5}

	newRenderer(t).Draw(c, w, cam, noon)

	_, side, _ := c.At(20, 4).RGB()   // the wall ahead, above the middle
	_, floor, _ := c.At(20, 38).RGB() // the floor, near the bottom
	if floor <= side {
		t.Errorf("floor green %d, wall green %d: the top should be brighter", floor, side)
	}
}

func TestHolesShowWhatIsBehind(t *testing.T) {
	w := world.New()
	wall(w, 4, world.Glass)
	wall(w, 6, world.Brick)
	c := newCanvas(t, 40, 20)

	newRenderer(t).Draw(c, w, facing, noon)

	red := 0
	for y := range c.Height() {
		for x := range c.Width() {
			if r, g, _ := c.At(x, y).RGB(); int(r) > int(g)+25 {
				red++
			}
		}
	}
	if total := c.Width() * c.Height(); red < total/2 {
		t.Errorf("%d of %d pixels show brick through the glass, wanted most", red, total)
	}
}

func TestWaterTintsWhatIsUnderIt(t *testing.T) {
	dry, wet := world.New(), world.New()
	wall(dry, 6, world.Sand)
	wall(wet, 6, world.Sand)
	wall(wet, 4, world.Water)
	wall(wet, 5, world.Water)
	a, b := newCanvas(t, 40, 20), newCanvas(t, 40, 20)
	rend := newRenderer(t)

	rend.Draw(a, dry, facing, noon)
	rend.Draw(b, wet, facing, noon)

	dr, _, db := a.At(20, 20).RGB()
	wr, _, wb := b.At(20, 20).RGB()
	if db >= dr {
		t.Fatalf("dry sand is %v, wanted more red than blue", a.At(20, 20))
	}
	if wb <= wr {
		t.Errorf("sand under water is %v, wanted more blue than red", b.At(20, 20))
	}
	if wr <= 52 {
		t.Errorf("sand under water is %v: the sand does not show through", b.At(20, 20))
	}
}

func TestDistantBlocksFadeIntoTheSky(t *testing.T) {
	near, far := world.New(), world.New()
	wall(near, 10, world.Deepstone)
	wall(far, 2+int(ViewDistance)-2, world.Deepstone)
	a, b := newCanvas(t, 40, 20), newCanvas(t, 40, 20)
	rend := newRenderer(t)

	rend.Draw(a, near, facing, noon)
	rend.Draw(b, far, facing, noon)

	_, _, nb := a.At(20, 20).RGB()
	_, _, fb := b.At(20, 20).RGB()
	if fb < nb+80 {
		t.Errorf("blue of a dark block near %d and far %d: the far one should be nearly sky", nb, fb)
	}
}

func TestNothingIsDrawnBeyondTheFarPlane(t *testing.T) {
	w := world.New()
	wall(w, int(facing.Position.Z+ViewDistance)+2, world.Deepstone)
	c := newCanvas(t, 40, 20)

	newRenderer(t).Draw(c, w, facing, noon)

	// The sky: bright, and more blue than red. The wall would be dark.
	if r, _, b := c.At(20, 20).RGB(); b <= r || b < 200 {
		t.Errorf("the middle of the view is %v, wanted sky", c.At(20, 20))
	}
	// And with no depth, as the sky has none.
	if !c.SetDepth(20, 20, 1e9, engine.RGB(1, 2, 3)) {
		t.Error("a pixel beyond the far plane has a depth")
	}
}

func TestACameraWithNoFogDrawsFarBlocksInTheirOwnColor(t *testing.T) {
	w := world.New()
	wall(w, 2+int(ViewDistance)-2, world.Deepstone)
	c := newCanvas(t, 40, 20)
	cam := facing
	cam.Fog = nil

	newRenderer(t).Draw(c, w, cam, noon)

	if _, _, b := c.At(20, 20).RGB(); b > 90 {
		t.Errorf("a dark block far away is %v with no fog, wanted it dark", c.At(20, 20))
	}
}

// What glonk draws into the canvas after the world is hidden by nearer blocks and shows
// in front of farther ones: the blocks carry glonk's depth.
func TestGlonkMeshesAreHiddenByNearerBlocks(t *testing.T) {
	w := world.New()
	wall(w, 8, world.Stone)
	rend := newRenderer(t)
	marker := engine.RGB(255, 0, 255)

	count := func(z float64) int {
		c := newCanvas(t, 40, 20)
		rend.Draw(c, w, facing, noon)
		cube := mesh.Cube(1)
		cube.Transform.Position = geometry.Vector3{X: 12.5, Y: 12.5, Z: z}
		cube.DrawDepthPixels(c, facing, marker)
		n := 0
		for y := range c.Height() {
			for x := range c.Width() {
				if c.At(x, y) == marker {
					n++
				}
			}
		}
		return n
	}

	if n := count(6); n == 0 {
		t.Error("a cube in front of the wall is not drawn")
	}
	if n := count(11); n != 0 {
		t.Errorf("a cube behind the wall shows through it in %d pixels", n)
	}
}

func TestAViewFromUnderWaterIsBlue(t *testing.T) {
	w := world.New()
	for z := range 24 {
		for y := range 24 {
			for x := range 24 {
				w.Set(x, y, z, world.Water)
			}
		}
	}
	wall(w, 6, world.Sand)
	c := newCanvas(t, 40, 20)

	newRenderer(t).Draw(c, w, facing, noon)

	if r, _, b := c.At(20, 20).RGB(); b <= r {
		t.Errorf("sand seen from under water is %v, wanted more blue than red", c.At(20, 20))
	}
}

// One frame of a generated world at 240 by 67 cells, which is 240 by 134 pixels.
func BenchmarkDraw(b *testing.B) {
	w := world.Generate(1)
	rend := newRenderer(b)
	c := newCanvas(b, 240, 67)
	spawn := w.Spawn()
	cam := mesh.Camera{
		Position: geometry.Vector3{X: spawn.X, Y: spawn.Y + 1.6, Z: spawn.Z},
		Pitch:    -0.2,
		FOV:      math.Pi / 3,
		Far:      ViewDistance,
		Fog:      &noon.Fog,
	}

	for _, view := range []struct {
		name string
		yaw  float64
	}{{"north", 0}, {"east", math.Pi / 2}, {"south", math.Pi}, {"west", -math.Pi / 2}} {
		cam.Yaw = view.yaw
		b.Run(view.name, func(b *testing.B) {
			for b.Loop() {
				c.Clear()
				rend.Draw(c, w, cam, noon)
			}
		})
	}
}

// reds counts the pixels of a canvas that are the red of a flower's petals.
func reds(c *engine.Canvas) int {
	n := 0
	for y := range c.Height() {
		for x := range c.Width() {
			if p := c.At(x, y); !p.IsDefault() {
				if r, g, b := p.RGB(); int(r) > int(g)+70 && int(r) > int(b)+70 {
					n++
				}
			}
		}
	}
	return n
}

// A plant is two pictures that cross in its cell. The ray takes the color of the
// picture it meets, and goes on through the holes round it to what is behind.
func TestAPlantShowsItsPictureAndWhatIsBehindIt(t *testing.T) {
	w := world.New()
	wall(w, 6, world.Stone)
	w.Set(12, 12, 4, world.RedFlower) // straight ahead of the camera, in front of the wall
	c := newCanvas(t, 80, 40)

	newRenderer(t).Draw(c, w, facing, noon)

	petals, gray := reds(c), 0
	for y := range c.Height() {
		for x := range c.Width() {
			if r, g, b := c.At(x, y).RGB(); max(r, g, b)-min(r, g, b) < 24 {
				gray++
			}
		}
	}
	if petals < 20 {
		t.Errorf("%d pixels are the red of the flower, wanted its bloom", petals)
	}
	if total := c.Width() * c.Height(); gray < total*3/4 {
		t.Errorf("%d of %d pixels are the stone behind the flower, wanted most", gray, total)
	}
}

// A mat is one picture lying a little above the floor of its cell: it is seen from above,
// lies over the ground, and from beside it hardly shows.
func TestAMatLiesFlatOnTheGround(t *testing.T) {
	w := world.New()
	for z := 8; z < 17; z++ {
		for x := 8; x < 17; x++ {
			w.Set(x, 5, z, world.Stone)
		}
	}
	w.Set(12, 6, 12, world.FurBed)
	rend := newRenderer(t)
	// The browns of the fur: redder than the grey stone it lies on.
	greens := func(c *engine.Canvas) (n int) {
		for y := range c.Height() {
			for x := range c.Width() {
				if r, _, b := c.At(x, y).RGB(); int(r) > int(b)+25 {
					n++
				}
			}
		}
		return n
	}

	above := newCanvas(t, 80, 40)
	cam := facing
	cam.Position, cam.Pitch = geometry.Vector3{X: 12.5, Y: 9, Z: 10}, -0.9
	rend.Draw(above, w, cam, noon)
	if n := greens(above); n < 30 {
		t.Errorf("from above, %d pixels are the brown of the fur bed", n)
	}

	beside := newCanvas(t, 80, 40)
	cam.Position, cam.Pitch = geometry.Vector3{X: 12.5, Y: 6.0625, Z: 9.5}, 0
	rend.Draw(beside, w, cam, noon)
	if n := greens(beside); n > 0 {
		t.Errorf("edge on, %d pixels are the brown of the fur bed, wanted none", n)
	}
}

func TestAPlantIsSeenFromEverySide(t *testing.T) {
	w := world.New()
	w.Set(12, 12, 12, world.RedFlower)
	rend := newRenderer(t)

	for _, from := range []struct {
		name string
		x, z float64
		yaw  float64
	}{
		{"south", 12.5, 9.5, 0}, {"north", 12.5, 15.5, math.Pi},
		{"west", 9.5, 12.5, math.Pi / 2}, {"east", 15.5, 12.5, -math.Pi / 2},
		{"a corner", 10.5, 10.5, math.Pi / 4},
	} {
		c := newCanvas(t, 80, 40)
		cam := facing
		cam.Position, cam.Yaw = geometry.Vector3{X: from.x, Y: 12.5, Z: from.z}, from.yaw

		rend.Draw(c, w, cam, noon)

		if n := reds(c); n < 10 {
			t.Errorf("from the %s, %d pixels are the red of the flower", from.name, n)
		}
	}
}

// A plant stores the depth of its own picture, so a thing behind it is hidden by its
// petals and shows through its holes.
func TestAPlantHidesWhatIsBehindItsPictureOnly(t *testing.T) {
	w := world.New()
	w.Set(12, 12, 4, world.RedFlower)
	rend := newRenderer(t)
	c := newCanvas(t, 80, 40)
	rend.Draw(c, w, facing, noon)
	before := reds(c)

	// A sheet of pixels at a depth just behind the plant, over the whole canvas.
	marker, shown := engine.RGB(0, 255, 255), 0
	for y := range c.Height() {
		for x := range c.Width() {
			if c.SetDepth(x, y, 2.6, marker) {
				shown++
			}
		}
	}

	if reds(c) != before {
		t.Errorf("%d pixels of the flower are left of %d after a thing was drawn behind it", reds(c), before)
	}
	if shown == 0 || shown == c.Width()*c.Height() {
		t.Errorf("the thing behind shows in %d pixels of %d, wanted it round the flower and not over it", shown, c.Width()*c.Height())
	}
}

// Frost whitens the tops of blocks that are open to the sky, and nothing else: not their
// sides, and not what a roof is over.
func TestFrostWhitensTheOpenGroundAndNotSidesOrWhatIsRoofed(t *testing.T) {
	scene := func(roofed bool, frost float64) *engine.Canvas {
		w := world.New()
		for z := range 24 {
			for x := range 24 {
				w.Set(x, 10, z, world.Turf)
				if roofed {
					w.Set(x, 16, z, world.Stone)
				}
			}
		}
		wall(w, 20, world.Turf)
		c := newCanvas(t, 40, 20)
		cam := facing
		cam.Position = geometry.Vector3{X: 12.5, Y: 12.5, Z: 9.5}
		r := newRenderer(t)
		r.Frost = frost
		r.Draw(c, w, cam, noon)
		return c
	}
	pale := func(c engine.Color) int {
		r, g, b := c.RGB()
		return int(min(r, g, b))
	}
	green, white := scene(false, 0), scene(false, 1)
	if was, is := pale(green.At(20, 38)), pale(white.At(20, 38)); is < was+60 {
		t.Errorf("the open ground is %v with no frost and %v with all of it: wanted it white", green.At(20, 38), white.At(20, 38))
	}
	if green.At(20, 4) != white.At(20, 4) {
		t.Errorf("the side of a wall is %v with no frost and %v with it: wanted it as it was", green.At(20, 4), white.At(20, 4))
	}
	half := scene(false, 0.5)
	if was, mid, is := pale(green.At(20, 38)), pale(half.At(20, 38)), pale(white.At(20, 38)); mid <= was || mid >= is {
		t.Errorf("the open ground with half the frost is %v, wanted between %v and %v", half.At(20, 38), green.At(20, 38), white.At(20, 38))
	}
	under, frosted := scene(true, 0), scene(true, 1)
	for y := range 40 {
		if under.At(20, y) != frosted.At(20, y) {
			t.Fatalf("under a roof the pixel at row %d is %v with no frost and %v with it", y, under.At(20, y), frosted.At(20, y))
		}
	}
}

// What glonk draws under the top of water is seen through the water, as the ground under
// it is. What it draws in front of the water is not.
func TestWaterIsOverWhatIsDrawnUnderIt(t *testing.T) {
	w := world.New()
	wall(w, 8, world.Sand)
	for z := 5; z <= 7; z++ {
		wall(w, z, world.Water)
	}
	rend := newRenderer(t)
	marker := engine.RGB(255, 0, 255)

	// The view with the edges of a cube in the marker's color at z, washed so many times.
	view := func(z float64, washes int) *engine.Canvas {
		c := newCanvas(t, 40, 20)
		rend.Draw(c, w, facing, noon)
		if z > 0 {
			cube := mesh.Cube(1)
			cube.Transform.Position = geometry.Vector3{X: 12.5, Y: 12.5, Z: z}
			cube.DrawDepthPixels(c, facing, marker)
		}
		for range washes {
			rend.Wash(c)
		}
		return c
	}
	differ := func(a, b *engine.Canvas) (n, marked int) { return changed(a, b, marker) }

	bare := view(0, 0)
	if n, _ := differ(view(0, 1), bare); n != 0 {
		t.Errorf("a wash changed %d pixels of water with nothing in it", n)
	}
	if n, marked := differ(view(3.9, 1), bare); n == 0 || marked != n {
		t.Errorf("of the %d pixels of a cube in front of the water, %d are as they were drawn", n, marked)
	}
	drawn, _ := differ(view(6.5, 0), bare)
	under := view(6.5, 1)
	if n, marked := differ(under, bare); drawn == 0 || n != drawn || marked != 0 {
		t.Fatalf("a cube under the water was drawn in %d pixels: washed, %d are not the water's, and %d are as they were drawn", drawn, n, marked)
	}
	for y := range under.Height() {
		for x := range under.Width() {
			red, _, blue := under.At(x, y).RGB()
			if was, _, _ := bare.At(x, y).RGB(); under.At(x, y) != bare.At(x, y) && (red <= was || red > 150 || blue < 100) {
				t.Fatalf("a pixel of a cube under the water is %v: wanted it between its own %v and the water's %v", under.At(x, y), marker, bare.At(x, y))
			}
		}
	}
	if n, _ := differ(view(6.5, 2), under); n != 0 {
		t.Errorf("a second wash changed %d pixels", n)
	}
}

// changed returns how many pixels of a are not as they are in b, and how many of those
// are of a color.
func changed(a, b *engine.Canvas, of engine.Color) (n, marked int) {
	for y := range a.Height() {
		for x := range a.Width() {
			if a.At(x, y) != b.At(x, y) {
				n++
				if a.At(x, y) == of {
					marked++
				}
			}
		}
	}
	return n, marked
}

// Water is seen into ten blocks, and no farther: what lies deeper in it than that does
// not show, and a thing drawn that deep in it is hidden.
func TestWaterHidesWhatLiesDeepInIt(t *testing.T) {
	rend := newRenderer(t)
	marker := engine.RGB(255, 0, 255)
	// Water so many blocks thick from z = 5, with sand behind it at a layer and the
	// edges of a cube in it at a place, if they are over 0.
	view := func(thick, sand int, cube float64) *engine.Canvas {
		w := world.New()
		for z := 5; z < 5+thick; z++ {
			wall(w, z, world.Water)
		}
		if sand > 0 {
			wall(w, sand, world.Sand)
		}
		c := newCanvas(t, 40, 20)
		rend.Draw(c, w, facing, noon)
		if cube > 0 {
			m := mesh.Cube(1)
			m.Transform.Position = geometry.Vector3{X: 12.5, Y: 12.5, Z: cube}
			m.DrawDepthPixels(c, facing, marker)
			rend.Wash(c)
		}
		return c
	}
	deep := view(14, 0, 0)
	if n, _ := changed(view(14, 19, 0), deep, marker); n != 0 {
		t.Errorf("sand under fourteen blocks of water shows in %d pixels", n)
	}
	if n, _ := changed(view(14, 0, 17.5), deep, marker); n != 0 {
		t.Errorf("a cube twelve blocks deep in water shows in %d pixels", n)
	}
	if n, marked := changed(view(14, 0, 9.5), deep, marker); n == 0 || marked != 0 {
		t.Errorf("a cube four blocks deep in water shows in %d pixels, %d of them not washed: wanted it seen through the water", n, marked)
	}
	if n, _ := changed(view(3, 8, 0), view(3, 0, 0), marker); n == 0 {
		t.Error("sand under three blocks of water does not show")
	}
	// The water thickens evenly: sand under nine blocks of it shows a little, and less
	// than sand under six.
	far := func(c *engine.Canvas) float64 {
		r, g, b := c.At(20, 20).RGB()
		dr, dg, db := deep.At(20, 20).RGB()
		return math.Abs(float64(r)-float64(dr)) + math.Abs(float64(g)-float64(dg)) + math.Abs(float64(b)-float64(db))
	}
	six, nine := far(view(6, 11, 0)), far(view(9, 14, 0))
	if nine <= 0 || nine >= six {
		t.Errorf("sand under six blocks of water is %v from the color of deep water, and under nine %v: wanted it to fade as the water deepens, and not be gone", six, nine)
	}
}

// An eye under water sees twenty blocks in it and no farther, and what is drawn in the
// water round it is seen through the water.
func TestFromUnderWaterTheEyeSeesTwentyBlocksAndWhatIsDrawnIsWashed(t *testing.T) {
	rend := newRenderer(t)
	marker := engine.RGB(255, 0, 255)
	// The view from in water forty blocks thick, with sand in it at a layer and the
	// edges of a cube at a place, if they are over 0, washed or not.
	view := func(sand int, cube float64, wash bool) *engine.Canvas {
		w := world.New()
		for z := range 40 {
			wall(w, z, world.Water)
		}
		if sand > 0 {
			wall(w, sand, world.Sand)
		}
		c := newCanvas(t, 40, 20)
		rend.Draw(c, w, facing, noon)
		if cube > 0 {
			m := mesh.Cube(1)
			m.Transform.Position = geometry.Vector3{X: 12.5, Y: 12.5, Z: cube}
			m.DrawDepthPixels(c, facing, marker)
		}
		if wash {
			rend.Wash(c)
			rend.Wash(c) // a second changes nothing
		}
		return c
	}
	open := view(0, 0, false)
	if n, _ := changed(view(30, 0, false), open, marker); n != 0 {
		t.Errorf("sand 27 blocks off under water shows in %d pixels", n)
	}
	if n, _ := changed(view(10, 0, false), open, marker); n == 0 {
		t.Error("sand 7 blocks off under water does not show")
	}
	if n, _ := changed(view(0, 30, true), open, marker); n != 0 {
		t.Errorf("a cube 27 blocks off under water shows in %d pixels", n)
	}
	drawn, marked := changed(view(0, 8.5, false), open, marker)
	if drawn == 0 || marked != drawn {
		t.Fatalf("a cube 6 blocks off under water was drawn in %d pixels, %d of them its own color", drawn, marked)
	}
	washed := view(0, 8.5, true)
	if n, marked := changed(washed, open, marker); n != drawn || marked != 0 {
		t.Errorf("of the %d pixels of a cube under water, %d show after the wash, and %d are as they were drawn: wanted all seen through the water", drawn, n, marked)
	}
	for y := range washed.Height() {
		for x := range washed.Width() {
			if red, _, blue := washed.At(x, y).RGB(); washed.At(x, y) != open.At(x, y) && (red > 150 || red < 30 || blue < 100) {
				t.Fatalf("a pixel of a cube under water is %v: wanted it between its own %v and the water's %v", washed.At(x, y), marker, open.At(x, y))
			}
		}
	}
}

// A great flier that is too far off to draw as it is stands in the sky as a small dark
// shape, wider the nearer it is, and nowhere else.
func TestTheSkyDrawsAFlierThatIsFarOff(t *testing.T) {
	w := world.New()
	rend := newRenderer(t)
	shape := func(ways ...geometry.Vector3) (n, left, right int) {
		sky := NewSky(Noon)
		plain := newCanvas(t, 120, 40)
		rend.Draw(plain, w, facing, sky)
		sky.SetWings(ways)
		c := newCanvas(t, 120, 40)
		rend.Draw(c, w, facing, sky)
		left, right = c.Width(), -1
		for y := range c.Height() {
			for x := range c.Width() {
				if c.At(x, y) == plain.At(x, y) {
					continue
				}
				n, left, right = n+1, min(left, x), max(right, x)
				red, green, blue := c.At(x, y).RGB()
				if was, _, _ := plain.At(x, y).RGB(); red >= was || int(red)+int(green)+int(blue) > 420 {
					t.Fatalf("a pixel of a far flier is %v against a sky of %v: wanted it dark", c.At(x, y), plain.At(x, y))
				}
			}
		}
		return n, left, right
	}
	if n, _, _ := shape(); n != 0 {
		t.Errorf("with no flier the sky is changed in %d pixels", n)
	}
	// One 80 blocks off, a little up and to the right of the middle of the view.
	n, left, right := shape(geometry.Vector3{X: 8, Y: 10, Z: 80})
	if n < 6 || n > 200 || left < 60 || right > 100 || right-left < 6 {
		t.Errorf("a flier 80 blocks off is drawn in %d pixels, from %d to %d across: wanted a small shape to the right of the middle, 60", n, left, right)
	}
	if _, l, r := shape(geometry.Vector3{X: 4, Y: 5, Z: 40}); r-l <= right-left {
		t.Errorf("a flier 40 blocks off is %d pixels wide, and one 80 off %d: wanted the nearer one wider", r-l, right-left)
	}
	// Behind the eye there is none.
	if n, _, _ := shape(geometry.Vector3{X: 8, Y: 10, Z: -80}); n != 0 {
		t.Errorf("a flier behind the eye is drawn in %d pixels", n)
	}
}

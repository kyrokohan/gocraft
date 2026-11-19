package render

import (
	"math"
	"testing"

	"glonk.dev/engine"
	"glonk.dev/geometry"
	"glonk.dev/mesh"

	"github.com/kyrokohan/skyspear/internal/world"
)

// underACloud is a camera on the ground under the middle of one cloud of cl, looking
// straight up, with the far plane and fog of the game's camera.
func underACloud(t *testing.T, cl *Clouds) mesh.Camera {
	t.Helper()
	for j := range cloudCells {
		for i := range cloudCells {
			x, z := cloudMiddle(i, j)
			if math.Abs(x) > 60 || math.Abs(z) > 60 {
				continue // the tests build under the cloud, where their world is loaded
			}
			if cl.at(i, j) && cl.at(i+1, j) && cl.at(i, j+1) && cl.at(i-1, j) && cl.at(i, j-1) {
				return mesh.Camera{
					Position: geometry.Vector3{X: x, Y: cloudBottom - 42, Z: z},
					Pitch:    1.5,
					FOV:      math.Pi / 3,
					Far:      ViewDistance,
					Fog:      &noon.Fog,
				}
			}
		}
	}
	t.Fatal("no cloud with clouds on all four sides")
	return mesh.Camera{}
}

// differing counts the pixels in which two canvases differ.
func differing(a, b *engine.Canvas) int {
	n := 0
	for y := range a.Height() {
		for x := range a.Width() {
			if a.At(x, y) != b.At(x, y) {
				n++
			}
		}
	}
	return n
}

// frames draws a world twice from one camera, without and with the clouds.
func frames(t *testing.T, w *world.World, cl *Clouds, cam mesh.Camera) (bare, clouded *engine.Canvas) {
	t.Helper()
	rend := newRenderer(t)
	bare, clouded = newCanvas(t, 60, 30), newCanvas(t, 60, 30)
	rend.Draw(bare, w, cam, noon)
	rend.Draw(clouded, w, cam, noon)
	cl.Draw(clouded, cam, noon)
	return bare, clouded
}

func TestTheSameSeedMakesTheSameClouds(t *testing.T) {
	a, b, c := NewClouds(5), NewClouds(5), NewClouds(6)

	same, other := true, true
	for j := range cloudCells {
		for i := range cloudCells {
			same = same && a.at(i, j) == b.at(i, j)
			other = other && a.at(i, j) == c.at(i, j)
		}
	}
	if !same {
		t.Error("one seed made two skies")
	}
	if other {
		t.Error("two seeds made the same sky")
	}
}

func TestSomeOfTheSkyIsCloudAndMostIsNot(t *testing.T) {
	cl := NewClouds(1)

	n := 0
	for j := range cloudCells {
		for i := range cloudCells {
			if cl.at(i, j) {
				n++
			}
		}
	}

	if share := float64(n) / (cloudCells * cloudCells); share < 0.12 || share > 0.5 {
		t.Errorf("%.0f%% of the sky is cloud, wanted between 12 and 50", share*100)
	}
}

func TestCloudsShowOverheadAndAreLighterThanTheSky(t *testing.T) {
	cl := NewClouds(1)
	cam := underACloud(t, cl)

	bare, clouded := frames(t, world.New(), cl, cam)

	if n := differing(bare, clouded); n < bare.Width()*bare.Height()/4 {
		t.Fatalf("clouds changed %d pixels of %d looking straight up under one", n, bare.Width()*bare.Height())
	}
	// The middle of the view is the underside of the cloud: nearer white than the sky is.
	sr, _, _ := bare.At(30, 30).RGB()
	cr, cg, cb := clouded.At(30, 30).RGB()
	if cr <= sr || cg < 150 || cb < 150 {
		t.Errorf("the cloud overhead is %v over a sky of %v, wanted it lighter", clouded.At(30, 30), bare.At(30, 30))
	}
}

func TestCloudsShowFromAboveToo(t *testing.T) {
	cl := NewClouds(1)
	cam := underACloud(t, cl)
	cam.Position.Y = cloudTop + 12
	cam.Pitch = -1.5

	bare, clouded := frames(t, world.New(), cl, cam)

	if n := differing(bare, clouded); n < bare.Width()*bare.Height()/8 {
		t.Errorf("clouds changed %d pixels looking down on one, wanted its top to show", n)
	}
}

func TestBlocksHideTheClouds(t *testing.T) {
	cl := NewClouds(1)
	cam := underACloud(t, cl)
	w := world.New()
	for z := -12; z <= 12; z++ {
		for x := -12; x <= 12; x++ {
			w.Set(int(math.Floor(cam.Position.X))+x, int(cam.Position.Y)+10, int(math.Floor(cam.Position.Z))+z, world.Planks) // a roof
		}
	}

	bare, clouded := frames(t, w, cl, cam)

	if n := differing(bare, clouded); n != 0 {
		t.Errorf("clouds show through a roof in %d pixels", n)
	}
}

// Clouds are seen through, so they store no depth: what is drawn after them is not
// hidden by them.
func TestCloudsStoreNoDepth(t *testing.T) {
	cl := NewClouds(1)
	cam := underACloud(t, cl)

	_, clouded := frames(t, world.New(), cl, cam)

	if !clouded.SetDepth(30, 30, 1e9, engine.RGB(1, 2, 3)) {
		t.Error("a pixel of cloud over open sky has a depth")
	}
}

func TestCloudsDriftAndComeRoundAgain(t *testing.T) {
	cl := NewClouds(1)
	cam := underACloud(t, cl)
	cam.Pitch = 0.9 // looking up and ahead, to see the edges of clouds move
	w := world.New()

	_, start := frames(t, w, cl, cam)
	cl.Advance(4)
	_, later := frames(t, w, cl, cam)
	if differing(start, later) == 0 {
		t.Error("the clouds did not move in four seconds")
	}

	// The sky repeats: once the clouds have gone its whole width they are where they began.
	cl.Advance(cloudField/cloudSpeed - 4)
	_, round := frames(t, w, cl, cam)
	if n := differing(start, round); n != 0 {
		t.Errorf("after going once round, the clouds differ from the start in %d pixels", n)
	}
}

// The clouds of a whole sky, over a frame of 240 by 67 cells, looking up and ahead.
func BenchmarkClouds(b *testing.B) {
	cl := NewClouds(1)
	c := newCanvas(b, 240, 67)
	cam := mesh.Camera{
		Position: geometry.Vector3{X: 0, Y: cloudBottom - 60, Z: 0},
		Pitch:    0.5,
		FOV:      math.Pi / 3,
		Far:      ViewDistance,
		Fog:      &noon.Fog,
	}

	for b.Loop() {
		c.Clear()
		cl.Draw(c, cam, noon)
	}
}

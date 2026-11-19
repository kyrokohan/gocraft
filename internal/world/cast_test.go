package world

import (
	"math"
	"math/rand/v2"
	"testing"

	"glonk.dev/geometry"
)

func vec(x, y, z float64) geometry.Vector3 {
	return geometry.Vector3{X: x, Y: y, Z: z}
}

// first returns the first block a ray meets.
func first(w *World, o, d geometry.Vector3, maxT float64) (Hit, bool) {
	r := NewRay(o, d)
	return w.Next(&r, maxT)
}

func TestRayAlongAnAxisHitsTheNearFace(t *testing.T) {
	w := New()
	w.Set(5, 10, 0, Stone)

	hit, ok := first(w, vec(0.5, 10.5, 0.5), vec(1, 0, 0), 100)

	want := Hit{X: 5, Y: 10, Z: 0, Face: FaceXNeg, T: 4.5, Block: Stone}
	if !ok || hit != want {
		t.Errorf("got %+v %v, wanted %+v", hit, ok, want)
	}
}

func TestRayFromAboveHitsTheTopFace(t *testing.T) {
	w := New()
	w.Set(3, 4, 3, Turf)

	hit, ok := first(w, vec(3.25, 9, 3.75), vec(0, -2, 0), 100)

	// The direction is 2 long, so 4 blocks down is 2 along the ray.
	want := Hit{X: 3, Y: 4, Z: 3, Face: FaceYPos, T: 2, Block: Turf}
	if !ok || hit != want {
		t.Errorf("got %+v %v, wanted %+v", hit, ok, want)
	}
}

func TestEveryFaceIsNamedByTheSideTheRayComesFrom(t *testing.T) {
	w := New()
	w.Set(10, 10, 10, Brick)
	center := vec(10.5, 10.5, 10.5)

	for _, tc := range []struct {
		from geometry.Vector3
		want Face
	}{
		{vec(-3, 0, 0), FaceXNeg}, {vec(3, 0, 0), FaceXPos},
		{vec(0, -3, 0), FaceYNeg}, {vec(0, 3, 0), FaceYPos},
		{vec(0, 0, -3), FaceZNeg}, {vec(0, 0, 3), FaceZPos},
	} {
		hit, ok := first(w, center.Add(tc.from), tc.from.Neg(), 100)
		if !ok || hit.Face != tc.want {
			t.Errorf("from %v: got face %v %v, wanted %v", tc.from, hit.Face, ok, tc.want)
		}
		nx, ny, nz := hit.Face.Normal()
		if got := vec(float64(nx), float64(ny), float64(nz)); got != tc.from.Scale(1.0/3) {
			t.Errorf("face %v points %v, wanted %v", hit.Face, got, tc.from.Scale(1.0/3))
		}
	}
}

func TestNextGoesOnFromTheLastHit(t *testing.T) {
	w := New()
	w.Set(4, 1, 1, Glass)
	w.Set(5, 1, 1, Water)
	w.Set(8, 1, 1, Stone)
	r := NewRay(vec(1.5, 1.5, 1.5), vec(1, 0, 0))

	var got []Block
	var dist []float64
	for {
		hit, ok := w.Next(&r, 100)
		if !ok {
			break
		}
		got = append(got, hit.Block)
		dist = append(dist, hit.T)
	}

	want, wantDist := []Block{Glass, Water, Stone}, []float64{2.5, 3.5, 6.5}
	if len(got) != len(want) {
		t.Fatalf("got %v, wanted %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] || dist[i] != wantDist[i] {
			t.Errorf("hit %d is %v at %v, wanted %v at %v", i, got[i], dist[i], want[i], wantDist[i])
		}
	}
}

func TestBlockBeyondTheReachIsNotHit(t *testing.T) {
	w := New()
	w.Set(5, 10, 0, Stone)

	if hit, ok := first(w, vec(0.5, 10.5, 0.5), vec(1, 0, 0), 4.4); ok {
		t.Errorf("hit %+v beyond the reach", hit)
	}
	if _, ok := first(w, vec(0.5, 10.5, 0.5), vec(1, 0, 0), 4.5); !ok {
		t.Error("a block exactly at the reach should be hit")
	}
}

func TestTheBlockTheRayStartsInIsNotReported(t *testing.T) {
	w := New()
	w.Set(5, 5, 5, Water)
	w.Set(7, 5, 5, Stone)

	hit, ok := first(w, vec(5.5, 5.5, 5.5), vec(1, 0, 0), 100)

	if !ok || hit.Block != Stone || hit.X != 7 {
		t.Errorf("got %+v %v, wanted the stone at x=7", hit, ok)
	}
}

func TestRayFromOutsideTheWorldComesIn(t *testing.T) {
	w := New()
	w.Set(0, 10, 0, Stone)

	hit, ok := first(w, vec(-5, 10.5, 0.5), vec(1, 0, 0), 100)

	if !ok || hit.X != 0 || hit.T != 5 {
		t.Errorf("got %+v %v, wanted the block at x=0, 5 away", hit, ok)
	}
}

func TestRaysThatMeetNothingEnd(t *testing.T) {
	w := New()
	w.Set(5, 5, 5, Stone)
	nan, inf := math.NaN(), math.Inf(1)

	for _, tc := range []struct{ o, d geometry.Vector3 }{
		{vec(1.5, 1.5, 1.5), vec(0, 1, 0)},       // out through the top
		{vec(1.5, 1.5, 1.5), vec(-1, -1, -1)},    // out through a corner
		{vec(-5, 80, -5), vec(-1, 1, -1)},        // outside and leaving
		{vec(1.5, 1.5, 1.5), vec(0, 0, 0)},       // going nowhere
		{vec(1.5, 1.5, 1.5), vec(nan, nan, nan)}, // nonsense
		{vec(nan, 1.5, 1.5), vec(1, 0, 0)},
		{vec(1.5, 1.5, 1.5), vec(inf, 0, 0)},
		{vec(1.5, 1.5, 1.5), vec(1, 0, 0)}, // no reach: the edge of the world ends it
	} {
		if hit, ok := first(w, tc.o, tc.d, math.Inf(1)); ok {
			t.Errorf("ray from %v along %v hit %+v", tc.o, tc.d, hit)
		}
	}
}

// Whatever the ray, the hit is on the face it names and nothing was skipped on the way.
func TestRandomRaysHitTheFirstBlockOnTheirWay(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	w := New()
	for range 4000 {
		w.Set(rng.IntN(24), rng.IntN(24), rng.IntN(24), Stone)
	}

	for i := range 500 {
		o := vec(rng.Float64()*24, rng.Float64()*24, rng.Float64()*24)
		if w.At(int(o.X), int(o.Y), int(o.Z)) != Air {
			continue
		}
		d := vec(rng.Float64()*2-1, rng.Float64()*2-1, rng.Float64()*2-1)
		hit, ok := first(w, o, d, 40)
		end := 40.0
		if ok {
			end = hit.T
			p := o.Add(d.Scale(hit.T))
			nx, ny, nz := hit.Face.Normal()
			// The point lies on the named face: on its plane, and inside the block otherwise.
			plane := float64(hit.X)*math.Abs(float64(nx)) + float64(hit.Y)*math.Abs(float64(ny)) + float64(hit.Z)*math.Abs(float64(nz))
			at := p.X*math.Abs(float64(nx)) + p.Y*math.Abs(float64(ny)) + p.Z*math.Abs(float64(nz))
			if nx+ny+nz > 0 {
				plane++
			}
			if math.Abs(at-plane) > 1e-9 {
				t.Errorf("ray %d: hit point %v is not on face %v of block %d,%d,%d", i, p, hit.Face, hit.X, hit.Y, hit.Z)
			}
		}
		for s := range 400 {
			p := o.Add(d.Scale((end - 1e-6) * float64(s) / 400))
			if b := w.At(int(math.Floor(p.X)), int(math.Floor(p.Y)), int(math.Floor(p.Z))); b != Air {
				t.Errorf("ray %d: passed through a block at %v before its hit at %v", i, p, end)
				break
			}
		}
	}
}

func TestTargetPassesThroughWater(t *testing.T) {
	w := New()
	w.Set(3, 1, 1, Water)
	w.Set(4, 1, 1, Water)
	w.Set(5, 1, 1, Sand)

	hit, ok := w.Target(vec(1.5, 1.5, 1.5), vec(1, 0, 0), 5)

	if !ok || hit.Block != Sand || hit.T != 3.5 {
		t.Errorf("got %+v %v, wanted the sand 3.5 away", hit, ok)
	}
}

func TestTargetIsNothingOverOpenWater(t *testing.T) {
	w := New()
	w.Set(3, 1, 1, Water)

	if hit, ok := w.Target(vec(1.5, 1.5, 1.5), vec(1, 0, 0), 5); ok {
		t.Errorf("got %+v, wanted nothing", hit)
	}
}

// A plant does not stop a body, but it can be aimed at, to pick it or build in its place.
func TestTargetStopsAtAPlant(t *testing.T) {
	w := New()
	w.Set(3, 1, 1, RedFlower)
	w.Set(5, 1, 1, Stone)

	hit, ok := w.Target(vec(1.5, 1.5, 1.5), vec(1, 0, 0), 5)

	if !ok || hit.Block != RedFlower || hit.X != 3 {
		t.Errorf("got %+v %v, wanted the flower at x=3", hit, ok)
	}
}

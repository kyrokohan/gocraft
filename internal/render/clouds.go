package render

import (
	"math"
	"slices"

	"glonk.dev/engine"
	"glonk.dev/geometry"
	"glonk.dev/mesh"
)

// The sky is a square field of cells, each a block of cloud or nothing. The field
// repeats in every direction, as the world below it has no end, so wherever the camera
// is there are clouds round it, and those that drift off one side come in at the other.
const (
	cloudCells  = 48                       // cells along each side of the field
	cloudCell   = 8.0                      // blocks along each side of a cell
	cloudField  = cloudCells * cloudCell   // blocks along each side of the field
	cloudBottom = 136.0                    // above the top of the world, so no block is ever in a cloud
	cloudTop    = cloudBottom + 4          // a cloud is a slab this thick
	cloudSpeed  = 1.5                      // blocks a second, along x
	cloudGroups = 6                        // groups along each side: glonk skips a group that is off screen
	groupCells  = cloudCells / cloudGroups // cells along each side of a group
	groupSize   = groupCells * cloudCell   // blocks along each side of a group
	cloudOrigin = -64.0                    // where the field starts on x and z: it is centered on the world
)

// Clouds are slabs of cloud that drift over the world. They are glonk meshes, drawn
// into the canvas after the blocks: glonk lights them, fades them in fog, lets the sky
// show through them, and hides them behind any block that is nearer.
type Clouds struct {
	cells  [cloudCells * cloudCells]bool
	groups []mesh.Mesh
	homes  []geometry.Vector3 // where the middle of each group is before any drift
	order  []int              // the groups from far to near, worked out for each frame
	drift  float64            // how far the clouds have moved along x, less than the field
	lights [2]mesh.Light
}

// puff returns a number from 0 up to 1 fixed by a point of a grid and a seed.
func puff(x, z int, seed uint32) float64 {
	h := uint32(x)*0x9E3779B1 ^ uint32(z)*0x85EBCA77 ^ seed*0xC2B2AE3D
	h ^= h >> 15
	h *= 0x2C1B3C6D
	h ^= h >> 12
	h *= 0x297A2D39
	h ^= h >> 15
	return float64(h) / (1 << 32)
}

// billow returns soft noise from 0 to 1 over the field of cells that changes over
// about size cells and repeats at the edge of the field.
func billow(i, j, size int, seed uint32) float64 {
	n := cloudCells / size
	x, z := float64(i)/float64(size), float64(j)/float64(size)
	x0, z0 := int(x), int(z)
	tx, tz := x-float64(x0), z-float64(z0)
	tx, tz = tx*tx*(3-2*tx), tz*tz*(3-2*tz)
	at := func(a, b int) float64 { return puff(a%n, b%n, seed) }
	near := at(x0, z0)*(1-tx) + at(x0+1, z0)*tx
	far := at(x0, z0+1)*(1-tx) + at(x0+1, z0+1)*tx
	return near*(1-tz) + far*tz
}

// NewClouds returns the sky a seed makes. The same seed always makes the same sky.
func NewClouds(seed int64) *Clouds {
	cl := &Clouds{}
	s := uint32(seed) ^ uint32(seed>>32)
	for j := range cloudCells {
		for i := range cloudCells {
			// Big shapes, with their edges broken up by smaller ones.
			cl.cells[j*cloudCells+i] = billow(i, j, 6, s+11)*0.7+billow(i, j, 2, s+12)*0.3 > 0.56
		}
	}

	for gj := range cloudGroups {
		for gi := range cloudGroups {
			if m, ok := cl.group(gi, gj); ok {
				home := geometry.Vector3{
					X: cloudOrigin + (float64(gi)+0.5)*groupSize,
					Y: (cloudBottom + cloudTop) / 2,
					Z: cloudOrigin + (float64(gj)+0.5)*groupSize,
				}
				cl.groups, cl.homes = append(cl.groups, m), append(cl.homes, home)
				cl.order = append(cl.order, len(cl.order))
			}
		}
	}
	return cl
}

// at reports whether a cell of the field is cloud. The field repeats, so any cell can
// be asked.
func (cl *Clouds) at(i, j int) bool {
	i, j = (i%cloudCells+cloudCells)%cloudCells, (j%cloudCells+cloudCells)%cloudCells
	return cl.cells[j*cloudCells+i]
}

// cloudMiddle returns where the middle of a cell of the field is, on x and z, before
// any drift.
func cloudMiddle(i, j int) (x, z float64) {
	return cloudOrigin + (float64(i)+0.5)*cloudCell, cloudOrigin + (float64(j)+0.5)*cloudCell
}

// group builds the mesh of one group of cells, measured from the middle of the group.
// Only the faces on the outside of a cloud are built: a face between two cells of cloud
// would show through the cloud as a seam, since the cloud is seen through. ok is false
// for a group with no cloud in it.
func (cl *Clouds) group(gi, gj int) (m mesh.Mesh, ok bool) {
	// quad adds a face with corners a, b, c, d, in the order that makes it face out.
	quad := func(a, b, c, d geometry.Vector3) {
		n := len(m.Vertices)
		m.Vertices = append(m.Vertices, a, b, c, d)
		m.Faces = append(m.Faces, [3]int{n, n + 1, n + 2}, [3]int{n, n + 2, n + 3})
	}
	at := func(x, y, z float64) geometry.Vector3 { return geometry.Vector3{X: x, Y: y, Z: z} }

	half, lo, hi := groupSize/2, -(cloudTop-cloudBottom)/2, (cloudTop-cloudBottom)/2
	for cj := range groupCells {
		for ci := range groupCells {
			i, j := gi*groupCells+ci, gj*groupCells+cj
			if !cl.at(i, j) {
				continue
			}
			x0, z0 := float64(ci)*cloudCell-half, float64(cj)*cloudCell-half
			x1, z1 := x0+cloudCell, z0+cloudCell

			quad(at(x0, hi, z0), at(x0, hi, z1), at(x1, hi, z1), at(x1, hi, z0)) // top
			quad(at(x0, lo, z0), at(x1, lo, z0), at(x1, lo, z1), at(x0, lo, z1)) // underside
			if !cl.at(i-1, j) {
				quad(at(x0, lo, z0), at(x0, lo, z1), at(x0, hi, z1), at(x0, hi, z0))
			}
			if !cl.at(i+1, j) {
				quad(at(x1, lo, z0), at(x1, hi, z0), at(x1, hi, z1), at(x1, lo, z1))
			}
			if !cl.at(i, j-1) {
				quad(at(x0, lo, z0), at(x0, hi, z0), at(x1, hi, z0), at(x1, lo, z0))
			}
			if !cl.at(i, j+1) {
				quad(at(x0, lo, z1), at(x1, lo, z1), at(x1, hi, z1), at(x0, hi, z1))
			}
		}
	}
	if len(m.Faces) == 0 {
		return mesh.Mesh{}, false
	}
	m.Material = mesh.Material{Color: engine.RGB(250, 252, 255), Opacity: 0.82}
	m.UpdateBounds() // so that glonk can skip the group when it is off screen
	return m, true
}

// nearest returns the one of the places v, v plus the width of the field, v less the
// width of the field and so on that is nearest to a point.
func nearest(v, to float64) float64 {
	return v - cloudField*math.Round((v-to)/cloudField)
}

// Advance moves the clouds on by dt seconds.
func (cl *Clouds) Advance(dt float64) {
	cl.drift = math.Mod(cl.drift+cloudSpeed*dt, cloudField)
}

// Draw draws the clouds as the camera sees them, over what is in the canvas, in the colors
// of the sky's time of day. Call it after the blocks and anything else that is solid: a
// cloud is tested against the depth that is there and stores none, so what is nearer
// hides it and nothing drawn later is hidden by it.
//
// The clouds are drawn three times as deep as the blocks, with a fog of their own that
// starts where the world's ends, so that the sky has clouds toward the horizon and they
// thin out into it: the camera's own Far and Fog are put aside for those.
func (cl *Clouds) Draw(c *engine.Canvas, cam mesh.Camera, sky *Sky) {
	cam.Far, cam.Fog = sky.cloudFog.End, &sky.cloudFog
	// Bright all over, and brighter still on the side the sun or the moon is on, so that
	// the top of a cloud is lightest and its underside a little gray.
	cl.lights = [2]mesh.Light{mesh.Ambient{Level: 0.72}, mesh.Directional{Direction: sky.giver().Neg()}}
	look := sky.cloud.engine()

	for i, home := range cl.homes {
		// The field of clouds repeats in every direction over a world with no end, so
		// each group is drawn at whichever of its places is nearest the camera.
		cl.groups[i].Transform.Position = geometry.Vector3{
			X: nearest(home.X+cl.drift, cam.Position.X),
			Y: home.Y,
			Z: nearest(home.Z, cam.Position.Z),
		}
	}

	// glonk does not sort what is seen through, and what is mixed in last shows most,
	// so the groups go in from far to near.
	slices.SortFunc(cl.order, func(a, b int) int {
		da := cl.groups[a].Transform.Position.Sub(cam.Position).LengthSq()
		db := cl.groups[b].Transform.Position.Sub(cam.Position).LengthSq()
		switch {
		case da > db:
			return -1
		case da < db:
			return 1
		}
		return a - b
	})
	for _, i := range cl.order {
		cl.groups[i].Material.Color = look
		cl.groups[i].DrawShadedPixels(c, cam, cl.lights[:]...)
	}
}

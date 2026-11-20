package render

import (
	"math"

	"glonk.dev/bitmap"
	"glonk.dev/engine"
	"glonk.dev/geometry"
	"glonk.dev/mesh"

	"github.com/kyrokohan/skyspear/internal/assets"
	"github.com/kyrokohan/skyspear/internal/world"
)

// Kind is a kind of beast.
type Kind uint8

const (
	Mossback  Kind = iota // a squat grazer with moss on its back
	Bugler                // a tall herd beast with a crest that sweeps back from its head
	Nipper                // a quick little thief the size of a hen, that goes in flocks
	Quickclaw             // a feathered hunter as tall as a person, that comes in the dark

	Shieldbrow  // a horned giant with a shield of bone behind its head, that guards its young
	Thornback   // a giant with plates along its back and spikes on its tail
	Boulderback // a low giant in armor, with a club at the end of its tail
	Cloudgrazer // a giant among giants: a long neck that feeds from the tops of trees
	Dread       // the top hunter: jaws as long as a person, on two legs like trees

	Snowtusk // a giant of the cold steppe in long hair, with a trunk and two great tusks
	Slowclaw // a slow shaggy beast of the steppe that stands up on its hind legs, with long claws

	Howler    // a grey hunter of the steppe night that goes in packs, and is heard before it comes
	Daggercat // a great cat of the steppe with two fangs like knives, that springs

	Kinds // how many kinds there are
)

var kindNames = [Kinds]string{
	Mossback: "mossback", Bugler: "bugler", Nipper: "nipper", Quickclaw: "quickclaw",
	Shieldbrow: "shieldbrow", Thornback: "thornback", Boulderback: "boulderback",
	Cloudgrazer: "cloudgrazer", Dread: "dread",
	Snowtusk: "snowtusk", Slowclaw: "slowclaw",
	Howler: "howler", Daggercat: "daggercat",
}

// String returns the name of a kind, as a save holds it.
func (k Kind) String() string {
	if k < Kinds {
		return kindNames[k]
	}
	return "beast"
}

// KindNamed returns the kind with a name, and whether there is one.
func KindNamed(name string) (Kind, bool) {
	for k, n := range kindNames {
		if n == name {
			return Kind(k), true
		}
	}
	return 0, false
}

// youngScale is how big a young beast is, of a grown one.
const youngScale = 0.5

// sizes are the boxes that blocks stop a grown beast of each kind in: how wide, and how
// high. A box is as deep as it is wide, though a beast is longer than that: a box cannot
// turn, and the beast does.
var sizes = [Kinds][2]float64{
	Mossback: {0.8, 0.95},
	Bugler:   {1.3, 3.0},
	Nipper:   {0.4, 0.6},

	Quickclaw: {0.6, 1.7},

	// A giant's box is narrower than the giant, so that it does not stick on the land.
	Shieldbrow:  {1.7, 2.4},
	Thornback:   {1.4, 2.4},
	Boulderback: {1.7, 1.3},

	// A cloudgrazer's is its legs and its trunk: its neck goes higher still.
	Cloudgrazer: {2.6, 7.3},

	// The dread is too wide for a gap of two blocks.
	Dread: {2.2, 4.6},

	Snowtusk: {2.0, 4.6},
	Slowclaw: {1.2, 1.9},

	Howler:    {0.6, 1.1},
	Daggercat: {0.8, 1.4},
}

// spans are how far behind and ahead of the point between its feet the body of a grown
// beast reaches, for a kind that is much longer than its box is wide: what is thrown or
// swung at it meets it all along that.
var spans = [Kinds][2]float64{
	Shieldbrow:  {-3.6, 2.9},
	Thornback:   {-3.9, 2.4},
	Boulderback: {-3.3, 2.2},
	Cloudgrazer: {-11.5, 8.5},
	Dread:       {-5.4, 3.7},
	Snowtusk:    {-3.0, 2.9},
	Slowclaw:    {-1.5, 1.6},
	Daggercat:   {-1.2, 1.4},
}

// Span returns how far behind and ahead of the point between its feet a beast's body
// reaches, along the way it faces: half its box each way, for a kind that is no longer
// than that.
func Span(k Kind, young bool) (back, front float64) {
	width, _ := Size(k, young)
	if s := spans[k]; s != [2]float64{} {
		if young {
			return s[0] * youngScale, s[1] * youngScale
		}
		return s[0], s[1]
	}
	return -width / 2, width / 2
}

// Size returns the box that blocks stop a beast in: how wide, and how high.
func Size(k Kind, young bool) (width, height float64) {
	if young {
		return sizes[k][0] * youngScale, sizes[k][1] * youngScale
	}
	return sizes[k][0], sizes[k][1]
}

// Beast is where one beast is and how it stands: all that drawing it needs.
type Beast struct {
	Kind Kind

	// Pos is the middle of the bottom of its body: the point between its feet.
	Pos geometry.Vector3

	// Yaw is the way it faces, as a camera's Yaw is.
	Yaw float64

	// Stride is how far through a step its legs are, in radians: they swing as its sine.
	Stride float64

	// Light is how bright the light it stands in is, from 0 to 1: the Sky's Brightness of
	// the light in its cell.
	Light float64

	// Shorn is true while a mossback's moss grows back, Hurt while a beast shows a hit,
	// and Young for one that is not grown: the same shape, smaller.
	Shorn, Hurt, Young bool

	// Fall is how far a beast that has died has fallen, from 0 to 1: it sinks to the
	// ground and fades.
	Fall float64

	// Glow is true where it is dark enough for the eyes of a beast that has such eyes
	// to shine: they are then drawn lit in full, whatever the light.
	Glow bool

	// Lower is how far a beast that warns or charges has lowered its head, from 0 to 1,
	// and Lash how far its tail is swung to a side as it strikes, from -1 to 1.
	Lower, Lash float64
}

// limb is one box of a beast's shape.
type limb struct {
	mesh mesh.Mesh

	// bare takes the place of mesh on a shorn beast, where there is one.
	bare    mesh.Mesh
	hasBare bool

	// right, up and front are where the origin of the box is, from the point between
	// the beast's feet, and tilt how far its top is tipped back, as a camera's Pitch is.
	right, up, front, tilt float64

	// swing is how far the box swings forward and back as the beast steps, in radians:
	// a leg. sway is how far it swings from side to side: a tail, which swings lashMore
	// times as far when the beast strikes with it. nod is how far its front tips down
	// when the beast lowers its head.
	swing, sway, nod float64

	// eye is true for a box that is an eye: it has no picture, only a color, and it
	// shines in the dark.
	eye bool
}

// shape is a kind of beast as it is drawn.
type shape struct {
	stand float64 // how high its body is carried: it sinks by this as it falls dead
	parts []limb
	hurt  *bitmap.Bitmap // its picture, reddened, for a beast that was hit
}

// Beasts draws the beasts. They are glonk meshes: boxes with the beast's picture on
// their faces, lit by glonk and faded by the camera's fog, and drawn into the canvas
// after the blocks, whose depth hides them where a block is nearer.
type Beasts struct {
	shapes [Kinds]shape
	lights [2]mesh.Light
}

// NewBeasts builds the boxes every kind of beast is made of, with the pictures from set.
func NewBeasts(set *assets.Set) *Beasts {
	return &Beasts{shapes: [Kinds]shape{
		Mossback: mossbackShape(set),
		Bugler:   buglerShape(set),
		Nipper:   nipperShape(set),

		Quickclaw: quickclawShape(set),

		Shieldbrow:  shieldbrowShape(set),
		Thornback:   thornbackShape(set),
		Boulderback: boulderbackShape(set),
		Cloudgrazer: cloudgrazerShape(set),
		Dread:       dreadShape(set),

		Snowtusk: snowtuskShape(set),
		Slowclaw: slowclawShape(set),

		Howler:    howlerShape(set),
		Daggercat: daggercatShape(set),
	}}
}

// The parts of a mossback, in blocks, with its feet at 0 and its front toward +z.
const (
	legLength  = 0.35
	legWidth   = 0.2
	legAside   = 0.22 // from the middle to a leg
	legAhead   = 0.3  // from the middle to a front or back leg
	bodyWidth  = 0.7
	bodyHeight = 0.5
	bodyLength = 1.0
	headWidth  = 0.5
	headHeight = 0.45
	headLength = 0.4
	legSwing   = 0.5 // how far a leg swings each way, in radians
)

// The tiles of the mossback's picture, from the left, as cmd/genassets draws them.
const (
	tileBack = iota
	tileHide
	tileFace
	tileLeg
	tileBare     // the back once it is shorn
	tileBareHide // and its sides
)

// all6 is one tile on every face of a box.
func all6(t int) [6]int {
	return [6]int{t, t, t, t, t, t}
}

func mossbackShape(set *assets.Set) shape {
	look := mesh.Material{Color: engine.RGB(224, 208, 172), Texture: set.Mossback}

	// Each box says which tile goes on which of its faces, in the order of world.Face.
	hide := [6]int{tileHide, tileHide, tileHide, tileBack, tileHide, tileHide}
	bare := [6]int{tileBareHide, tileBareHide, tileBareHide, tileBare, tileBareHide, tileBareHide}
	face := hide
	face[world.FaceZPos] = tileFace

	const n = assets.MossbackTiles
	// A leg hangs from its own origin, so that turning it there swings it from the hip.
	// The legs at opposite corners swing together, as a walking animal's do.
	leg := box(-legWidth/2, -legLength, -legWidth/2, legWidth, legLength, legWidth, all6(tileLeg), n, look)
	return shape{
		stand: legLength,
		hurt:  reddened(set.Mossback),
		parts: []limb{
			{
				mesh:    box(-bodyWidth/2, legLength, -bodyLength/2, bodyWidth, bodyHeight, bodyLength, hide, n, look),
				bare:    box(-bodyWidth/2, legLength, -bodyLength/2, bodyWidth, bodyHeight, bodyLength, bare, n, look),
				hasBare: true,
			},
			{mesh: box(-headWidth/2, legLength+0.15, bodyLength/2, headWidth, headHeight, headLength, face, n, look)},
			{mesh: leg, right: -legAside, up: legLength, front: legAhead, swing: legSwing},
			{mesh: leg, right: legAside, up: legLength, front: legAhead, swing: -legSwing},
			{mesh: leg, right: -legAside, up: legLength, front: -legAhead, swing: -legSwing},
			{mesh: leg, right: legAside, up: legLength, front: -legAhead, swing: legSwing},
		},
	}
}

// The tiles of the bugler's picture, from the left, as cmd/genassets draws them.
const (
	buglerBack = iota
	buglerSide
	buglerBelly
	buglerHead // the side of its head, with the eye
	buglerBill
	buglerLeg
	buglerCrest
)

// buglerShape is a bugler: it walks on two great hind legs with its body level, a heavy
// tail out behind and its neck up in front, and a hollow crest sweeps back from its head.
func buglerShape(set *assets.Set) shape {
	look := mesh.Material{Color: engine.RGB(118, 128, 72), Texture: set.Bugler}
	const n = assets.BuglerTiles
	const hip = 1.3 // how long its hind legs are

	// A box of hide: its back on top, its belly under it, and its side all round.
	hide := [6]int{buglerSide, buglerSide, buglerBelly, buglerBack, buglerSide, buglerSide}
	head := hide
	head[world.FaceXNeg], head[world.FaceXPos], head[world.FaceZPos] = buglerHead, buglerHead, buglerBill

	leg := box(-0.17, -hip, -0.17, 0.34, hip, 0.34, all6(buglerLeg), n, look)
	arm := box(-0.09, -0.75, -0.09, 0.18, 0.75, 0.18, all6(buglerLeg), n, look)
	return shape{
		stand: hip,
		hurt:  reddened(set.Bugler),
		parts: []limb{
			{mesh: box(-0.5, hip-0.1, -1.1, 1.0, 1.0, 2.1, hide, n, look)},
			// The tail is two boxes, each with its origin at its root. The far one sways.
			{mesh: box(-0.3, -0.3, -1.3, 0.6, 0.6, 1.3, hide, n, look), up: hip + 0.45, front: -1.1},
			{mesh: box(-0.17, -0.17, -1.2, 0.34, 0.34, 1.2, hide, n, look), up: hip + 0.45, front: -2.4, sway: 0.25},
			{mesh: box(-0.225, hip+0.7, 0.75, 0.45, 1.0, 0.5, hide, n, look)},
			{mesh: box(-0.25, hip+1.45, 0.8, 0.5, 0.45, 0.95, head, n, look)},
			// The crest runs back from over the eyes and rises as it goes.
			{mesh: box(-0.08, 0, -1.0, 0.16, 0.22, 1.0, all6(buglerCrest), n, look), up: hip + 1.8, front: 1.05, tilt: -0.35},
			{mesh: leg, right: -0.32, up: hip, front: -0.2, swing: 0.45},
			{mesh: leg, right: 0.32, up: hip, front: -0.2, swing: -0.45},
			{mesh: arm, right: -0.4, up: hip + 0.05, front: 0.75, swing: -0.3},
			{mesh: arm, right: 0.4, up: hip + 0.05, front: 0.75, swing: 0.3},
		},
	}
}

// The tiles of the nipper's picture, from the left, as cmd/genassets draws them.
const (
	nipperBack = iota
	nipperSide
	nipperHead
	nipperLeg
)

// nipperShape is a nipper: a little runner on two thin legs, its body level, with a
// long neck up in front and a longer tail out behind to balance it.
func nipperShape(set *assets.Set) shape {
	look := mesh.Material{Color: engine.RGB(64, 132, 84), Texture: set.Nipper}
	const n = assets.NipperTiles
	const hip = 0.3 // how long its legs are

	hide := [6]int{nipperSide, nipperSide, nipperSide, nipperBack, nipperSide, nipperSide}
	head := [6]int{nipperHead, nipperHead, nipperSide, nipperBack, nipperSide, nipperSide}
	leg := box(-0.035, -hip, -0.035, 0.07, hip, 0.07, all6(nipperLeg), n, look)
	return shape{
		stand: hip,
		hurt:  reddened(set.Nipper),
		parts: []limb{
			{mesh: box(-0.11, hip-0.02, -0.2, 0.22, 0.2, 0.42, hide, n, look)},
			{mesh: box(-0.05, -0.05, -0.5, 0.1, 0.1, 0.5, hide, n, look), up: hip + 0.1, front: -0.2, sway: 0.35},
			{mesh: box(-0.05, hip+0.1, 0.16, 0.1, 0.24, 0.1, hide, n, look)},
			{mesh: box(-0.08, hip+0.28, 0.14, 0.16, 0.15, 0.26, head, n, look)},
			{mesh: leg, right: -0.08, up: hip, swing: 0.7},
			{mesh: leg, right: 0.08, up: hip, swing: -0.7},
		},
	}
}

// lashMore is how many times as far as it sways in a walk a tail swings in a blow.
const lashMore = 3.5

// The tiles of the picture of an armed plant-eater, from the left, as cmd/genassets
// draws them: the last two are what it is armed with.
const (
	giantBack = iota
	giantSide
	giantBelly
	giantHead
	giantFace
	giantLeg
	giantArmA // a shieldbrow's frill, a thornback's plate, a boulderback's club
	giantArmB // a horn or a spike, and a boulderback's tail
)

// giantHide returns the tiles of a box of a giant's hide, and of its head.
func giantHide() (hide, head [6]int) {
	hide = [6]int{giantSide, giantSide, giantBelly, giantBack, giantSide, giantSide}
	head = hide
	head[world.FaceXNeg], head[world.FaceXPos], head[world.FaceZPos] = giantHead, giantHead, giantFace
	return hide, head
}

// fourLegs returns the legs of a beast that walks on four: each hangs from its hip, and
// those at opposite corners swing together.
func fourLegs(leg mesh.Mesh, hip, aside, ahead, behind, swing float64) []limb {
	return []limb{
		{mesh: leg, right: -aside, up: hip, front: ahead, swing: swing},
		{mesh: leg, right: aside, up: hip, front: ahead, swing: -swing},
		{mesh: leg, right: -aside, up: hip, front: behind, swing: -swing},
		{mesh: leg, right: aside, up: hip, front: behind, swing: swing},
	}
}

// shieldbrowShape is a shieldbrow: a barrel of a body on four legs like trees, a heavy
// tail, and a great head it carries low, with a beak, a horn on its nose, two long ones
// over its eyes, and a shield of bone standing up behind them. Its head is one piece,
// hung from its neck, so that it lowers whole.
func shieldbrowShape(set *assets.Set) shape {
	look := mesh.Material{Color: engine.RGB(134, 114, 90), Texture: set.Shieldbrow}
	const n = assets.ShieldbrowTiles
	const hip = 1.0
	hide, head := giantHide()
	frill := hide
	frill[world.FaceZPos], frill[world.FaceZNeg] = giantArmA, giantArmA

	// The head's boxes are all placed from the neck, and nod together.
	neck := limb{up: hip + 0.75, front: 1.5, nod: 0.45}
	part := func(m mesh.Mesh) limb {
		l := neck
		l.mesh = m
		return l
	}
	parts := []limb{
		{mesh: box(-0.95, hip-0.1, -1.6, 1.9, 1.5, 3.2, hide, n, look)},
		{mesh: box(-0.4, -0.4, -1.0, 0.8, 0.8, 1.0, hide, n, look), up: hip + 0.7, front: -1.6},
		{mesh: box(-0.22, -0.22, -1.1, 0.44, 0.44, 1.1, hide, n, look), up: hip + 0.6, front: -2.5, sway: 0.2},
		part(box(-0.5, -0.55, 0, 1.0, 1.0, 1.4, head, n, look)),
		part(box(-1.05, -0.1, -0.12, 2.1, 1.55, 0.24, frill, n, look)),
		part(box(-0.38, 0.32, 0.5, 0.15, 0.15, 1.2, all6(giantArmB), n, look)),
		part(box(0.23, 0.32, 0.5, 0.15, 0.15, 1.2, all6(giantArmB), n, look)),
		part(box(-0.09, 0.4, 1.05, 0.18, 0.4, 0.18, all6(giantArmB), n, look)),
	}
	leg := box(-0.27, -hip, -0.27, 0.54, hip, 0.54, all6(giantLeg), n, look)
	return shape{stand: hip, hurt: reddened(set.Shieldbrow), parts: append(parts, fourLegs(leg, hip, 0.62, 1.1, -1.1, 0.3)...)}
}

// thornbackShape is a thornback: a high arched body on four legs, a small head carried
// low, a row of plates standing along its back, the tallest in the middle, and a tail
// that ends in four spikes.
func thornbackShape(set *assets.Set) shape {
	look := mesh.Material{Color: engine.RGB(98, 118, 72), Texture: set.Thornback}
	const n = assets.ThornbackTiles
	const hip = 0.9
	hide, head := giantHide()
	plate := all6(giantArmA)

	// The end of the tail and its spikes are placed from one point, and swing together.
	end := limb{up: hip + 0.75, front: -2.5, sway: 0.3}
	part := func(m mesh.Mesh) limb {
		l := end
		l.mesh = m
		return l
	}
	parts := []limb{
		{mesh: box(-0.7, hip-0.05, -1.5, 1.4, 1.45, 3.0, hide, n, look)},
		{mesh: box(-0.35, -0.35, -1.0, 0.7, 0.7, 1.0, hide, n, look), up: hip + 0.8, front: -1.5},
		part(box(-0.2, -0.2, -1.4, 0.4, 0.4, 1.4, hide, n, look)),
		part(box(0.15, 0, -1.3, 0.65, 0.1, 0.1, all6(giantArmB), n, look)),
		part(box(-0.8, 0, -1.3, 0.65, 0.1, 0.1, all6(giantArmB), n, look)),
		part(box(0.15, 0.05, -0.85, 0.6, 0.1, 0.1, all6(giantArmB), n, look)),
		part(box(-0.75, 0.05, -0.85, 0.6, 0.1, 0.1, all6(giantArmB), n, look)),
		{mesh: box(-0.22, hip+0.15, 1.5, 0.44, 0.5, 0.55, hide, n, look)},
		{mesh: box(-0.26, hip-0.05, 1.95, 0.52, 0.5, 0.75, head, n, look)},
	}
	// Seven plates, in two rows a little to each side of the spine.
	for i := range 7 {
		high := 0.55 + 0.55*math.Sin(math.Pi*float64(i+1)/8)
		parts = append(parts, limb{
			mesh:  box(-0.05, 0, -0.26, 0.1, high, 0.52, plate, n, look),
			right: 0.16 * float64(1-2*(i%2)), up: hip + 1.38, front: -1.3 + 0.43*float64(i),
		})
	}
	leg := box(-0.22, -hip, -0.22, 0.44, hip, 0.44, all6(giantLeg), n, look)
	return shape{stand: hip, hurt: reddened(set.Thornback), parts: append(parts, fourLegs(leg, hip, 0.48, 1.0, -1.0, 0.3)...)}
}

// boulderbackShape is a boulderback: a wide flat body close to the ground under a roof
// of bone, a flat head, and a stiff tail with a club of bone at its end.
func boulderbackShape(set *assets.Set) shape {
	look := mesh.Material{Color: engine.RGB(152, 114, 82), Texture: set.Boulderback}
	const n = assets.BoulderbackTiles
	const hip = 0.5
	hide, head := giantHide()
	tail := [6]int{giantArmB, giantArmB, giantBelly, giantArmB, giantSide, giantSide}

	// The end of the tail and its club are placed from one point, and swing together.
	end := limb{up: hip + 0.4, front: -2.4, sway: 0.25}
	part := func(m mesh.Mesh) limb {
		l := end
		l.mesh = m
		return l
	}
	parts := []limb{
		{mesh: box(-1.0, hip-0.1, -1.4, 2.0, 0.95, 2.8, hide, n, look)},
		{mesh: box(-0.3, -0.25, -1.0, 0.6, 0.5, 1.0, tail, n, look), up: hip + 0.45, front: -1.4},
		part(box(-0.15, -0.15, -1.0, 0.3, 0.3, 1.0, tail, n, look)),
		part(box(-0.42, -0.28, -1.65, 0.84, 0.56, 0.7, all6(giantArmA), n, look)),
		{mesh: box(-0.42, hip, 1.4, 0.84, 0.62, 0.8, head, n, look)},
	}
	leg := box(-0.22, -hip, -0.22, 0.44, hip, 0.44, all6(giantLeg), n, look)
	return shape{stand: hip, hurt: reddened(set.Boulderback), parts: append(parts, fourLegs(leg, hip, 0.7, 0.95, -0.95, 0.35)...)}
}

// cloudgrazerShape is a cloudgrazer: a trunk as big as a hut on four legs like the
// boles of trees, a neck that rises ahead of it to twice that height with a small head
// at its end, and a tail as long again that sinks behind it.
func cloudgrazerShape(set *assets.Set) shape {
	look := mesh.Material{Color: engine.RGB(106, 124, 142), Texture: set.Cloudgrazer}
	const n = assets.CloudgrazerTiles
	const hip = 4.6
	hide, head := giantHide()
	neck := [6]int{giantArmA, giantArmA, giantBelly, giantArmA, giantSide, giantSide}
	end := [6]int{giantArmB, giantArmB, giantBelly, giantArmB, giantArmB, giantArmB}

	parts := []limb{
		{mesh: box(-1.5, hip-0.3, -3.4, 3.0, 3.0, 6.8, hide, n, look)},
		// The neck is two lengths, each from its root and tipped up, and the head
		// level at the top of them.
		{mesh: box(-0.65, -0.65, 0, 1.3, 1.3, 4.4, neck, n, look), up: hip + 1.9, front: 2.9, tilt: 0.9},
		{mesh: box(-0.5, -0.5, 0, 1.0, 1.0, 3.7, neck, n, look), up: hip + 5.1, front: 5.4, tilt: 1.1},
		{mesh: box(-0.5, hip+7.9, 6.7, 1.0, 0.95, 1.9, head, n, look)},
		// The tail is three lengths, each from its root and sinking, and the last two
		// sway.
		{mesh: box(-0.8, -0.8, -3.1, 1.6, 1.6, 3.1, hide, n, look), up: hip + 1.6, front: -3.3, tilt: 0.3},
		{mesh: box(-0.5, -0.5, -3.3, 1.0, 1.0, 3.3, hide, n, look), up: hip + 0.75, front: -6.1, tilt: 0.42, sway: 0.1},
		{mesh: box(-0.25, -0.25, -3.1, 0.5, 0.5, 3.1, end, n, look), up: hip - 0.55, front: -9.0, tilt: 0.5, sway: 0.2},
	}
	leg := box(-0.45, -hip, -0.45, 0.9, hip, 0.9, all6(giantLeg), n, look)
	return shape{stand: hip, hurt: reddened(set.Cloudgrazer), parts: append(parts, fourLegs(leg, hip, 1.0, 2.3, -2.3, 0.28)...)}
}

// snowtuskShape is a snowtusk: a high body in long hair on four legs like pillars, its
// back sloping down behind, a domed head carried high, a trunk that hangs from it and
// sways, and two tusks that sweep out ahead and up. Its head is one piece, hung from
// its neck, so that it lowers whole.
func snowtuskShape(set *assets.Set) shape {
	look := mesh.Material{Color: engine.RGB(122, 82, 52), Texture: set.Snowtusk}
	const n = assets.SnowtuskTiles
	const hip = 2.3
	hide, head := giantHide()
	trunk := all6(giantArmA)

	// The head's boxes are all placed from the neck, and nod together.
	neck := limb{up: hip + 1.5, front: 0.3, nod: 0.4}
	part := func(m mesh.Mesh) limb {
		l := neck
		l.mesh = m
		return l
	}
	tusk := all6(giantArmB)
	parts := []limb{
		{mesh: box(-1.1, hip-0.2, -1.9, 2.2, 2.3, 2.3, hide, n, look)},                                          // the shoulders, high
		{mesh: box(-1.0, hip-0.2, -3.0, 2.0, 1.9, 1.2, hide, n, look)},                                          // the back, lower
		{mesh: box(-0.12, -0.9, -0.12, 0.24, 0.9, 0.24, hide, n, look), up: hip + 1.5, front: -3.0, sway: 0.25}, // its tail
		part(box(-0.75, -0.6, 0, 1.5, 1.7, 1.4, head, n, look)),
		part(box(-0.3, -1.9, 1.25, 0.6, 1.5, 0.6, trunk, n, look)),
		part(box(-0.22, -3.0, 1.35, 0.44, 1.2, 0.44, trunk, n, look)),
		// The tusks: out of the face and forward, low, and then up at their ends.
		part(box(-0.78, -0.95, 1.1, 0.26, 0.26, 1.5, tusk, n, look)),
		part(box(0.52, -0.95, 1.1, 0.26, 0.26, 1.5, tusk, n, look)),
		part(box(-0.76, -0.75, 2.38, 0.22, 0.75, 0.22, tusk, n, look)),
		part(box(0.54, -0.75, 2.38, 0.22, 0.75, 0.22, tusk, n, look)),
	}
	leg := box(-0.36, -hip, -0.36, 0.72, hip, 0.72, all6(giantLeg), n, look)
	return shape{stand: hip, hurt: reddened(set.Snowtusk), parts: append(parts, fourLegs(leg, hip, 0.7, 0.7, -2.2, 0.25)...)}
}

// slowclawShape is a slowclaw: a heavy body in shaggy fur that slopes up from short hind
// legs, a small head on a thick neck with a long muzzle, a tail as thick as a leg that
// rests on the ground, and long arms that end in three claws. All of it above the hips
// is placed from them and tips up together: so it stands up, to feed and to strike.
func slowclawShape(set *assets.Set) shape {
	look := mesh.Material{Color: engine.RGB(168, 140, 96), Texture: set.Slowclaw}
	const n = assets.SlowclawTiles
	const hip = 0.8
	hide, head := giantHide()

	// What rears: it is hung from the hips, and its front goes up as the beast stands.
	hips := limb{up: hip + 0.3, front: -0.7, nod: -1.05}
	part := func(m mesh.Mesh) limb {
		l := hips
		l.mesh = m
		return l
	}
	claw := all6(giantArmB)
	arm := func(side float64) []limb {
		at := side*0.62 - 0.14
		l := []limb{part(box(at, -1.25, 1.35, 0.28, 1.3, 0.3, all6(giantArmA), n, look))}
		for i := range 3 {
			l = append(l, part(box(at+0.1*float64(i), -1.6, 1.42, 0.07, 0.4, 0.1, claw, n, look)))
		}
		return l
	}
	parts := []limb{
		part(box(-0.6, -0.5, 0, 1.2, 1.15, 2.0, hide, n, look)),   // the body
		part(box(-0.3, 0.05, 1.9, 0.6, 0.6, 0.45, hide, n, look)), // the neck
		part(box(-0.25, 0.05, 2.25, 0.5, 0.48, 0.75, head, n, look)),
		{mesh: box(-0.25, 0, -1.1, 0.5, 0.45, 1.1, hide, n, look), front: -0.7}, // the tail, on the ground
	}
	parts = append(append(parts, arm(-1)...), arm(1)...)
	leg := box(-0.24, -hip, -0.26, 0.48, hip, 0.52, all6(giantLeg), n, look)
	return shape{stand: hip, hurt: reddened(set.Slowclaw), parts: append(parts,
		limb{mesh: leg, right: -0.42, up: hip, front: -0.6, swing: 0.35},
		limb{mesh: leg, right: 0.42, up: hip, front: -0.6, swing: -0.35},
	)}
}

// howlerEyes and catEyes are the colors of the eyes of the hunters of the steppe.
var (
	howlerEyes = engine.RGB(176, 222, 255)
	catEyes    = engine.RGB(255, 190, 60)
)

// eyes returns the two eyes of a beast whose eyes shine, on the sides of its head: each
// so far to a side, so high and so far ahead of the point between its feet.
func eyes(shine engine.Color, n int, aside, up, front float64) []limb {
	eye := box(-0.02, -0.04, -0.05, 0.04, 0.08, 0.1, all6(0), n, mesh.Material{Color: shine})
	return []limb{
		{mesh: eye, right: -aside, up: up, front: front, eye: true},
		{mesh: eye, right: aside, up: up, front: front, eye: true},
	}
}

// howlerShape is a howler: a lean body on four long legs, a deep chest, a head carried
// level with a long muzzle and two ears that stand, and a brush of a tail that hangs.
// Its eyes shine.
func howlerShape(set *assets.Set) shape {
	look := mesh.Material{Color: engine.RGB(142, 146, 152), Texture: set.Howler}
	const n = assets.HowlerTiles
	const hip = 0.6
	hide, head := giantHide()
	ear := all6(giantArmB)
	parts := []limb{
		{mesh: box(-0.24, hip-0.05, -0.65, 0.48, 0.5, 1.3, hide, n, look)},
		{mesh: box(-0.17, hip+0.25, 0.6, 0.34, 0.36, 0.3, hide, n, look)}, // the neck
		{mesh: box(-0.19, hip+0.32, 0.8, 0.38, 0.36, 0.38, head, n, look)},
		{mesh: box(-0.1, hip+0.32, 1.16, 0.2, 0.2, 0.3, head, n, look)}, // the muzzle
		{mesh: box(-0.19, hip+0.66, 0.84, 0.1, 0.16, 0.08, ear, n, look)},
		{mesh: box(0.09, hip+0.66, 0.84, 0.1, 0.16, 0.08, ear, n, look)},
		{mesh: box(-0.08, -0.6, -0.08, 0.16, 0.6, 0.16, all6(giantArmA), n, look), up: hip + 0.4, front: -0.65, tilt: -0.5, sway: 0.3},
	}
	parts = append(parts, eyes(howlerEyes, n, 0.2, hip+0.56, 1.02)...)
	leg := box(-0.08, -hip, -0.08, 0.16, hip, 0.16, all6(giantLeg), n, look)
	return shape{stand: hip, hurt: reddened(set.Howler), parts: append(parts, fourLegs(leg, hip, 0.17, 0.5, -0.5, 0.6)...)}
}

// daggercatShape is a daggercat: a heavy body, higher at the shoulders than behind, on
// four thick legs, a short tail, and a broad head with a short muzzle, from under which
// two fangs hang as long as a hand. Its eyes shine.
func daggercatShape(set *assets.Set) shape {
	look := mesh.Material{Color: engine.RGB(196, 156, 96), Texture: set.Daggercat}
	const n = assets.DaggercatTiles
	const hip = 0.7
	hide, head := giantHide()
	fang := all6(giantArmB)
	parts := []limb{
		{mesh: box(-0.36, hip, -0.2, 0.72, 0.72, 1.1, hide, n, look)},       // the shoulders
		{mesh: box(-0.32, hip-0.05, -1.1, 0.64, 0.62, 0.95, hide, n, look)}, // the haunches, lower
		{mesh: box(-0.28, hip+0.25, 0.85, 0.56, 0.52, 0.5, head, n, look)},
		{mesh: box(-0.17, hip+0.25, 1.3, 0.34, 0.3, 0.26, head, n, look)}, // the muzzle
		{mesh: box(-0.16, hip-0.22, 1.38, 0.09, 0.48, 0.1, fang, n, look)},
		{mesh: box(0.07, hip-0.22, 1.38, 0.09, 0.48, 0.1, fang, n, look)},
		{mesh: box(-0.07, -0.45, -0.07, 0.14, 0.45, 0.14, all6(giantArmA), n, look), up: hip + 0.45, front: -1.1, tilt: -0.7, sway: 0.25},
	}
	parts = append(parts, eyes(catEyes, n, 0.29, hip+0.6, 1.2)...)
	leg := box(-0.12, -hip, -0.12, 0.24, hip, 0.24, all6(giantLeg), n, look)
	return shape{stand: hip, hurt: reddened(set.Daggercat), parts: append(parts, fourLegs(leg, hip, 0.24, 0.6, -0.8, 0.55)...)}
}

// dreadEyes is the color of the dread's eyes.
var dreadEyes = engine.RGB(255, 96, 40)

// dreadShape is the dread: a body as big as a hut carried level on two legs like trees,
// a tail as heavy as the rest of it behind, and in front a head as long as a person with
// a mouth it holds a little open. Its arms are small. Its eyes shine.
func dreadShape(set *assets.Set) shape {
	look := mesh.Material{Color: engine.RGB(66, 58, 56), Texture: set.Dread}
	const n = assets.DreadTiles
	const hip = 2.7
	hide, head := giantHide()
	jaw := [6]int{giantHead, giantHead, giantBelly, giantArmB, giantSide, giantFace}

	leg := box(-0.4, -hip, -0.45, 0.8, hip, 0.9, all6(giantLeg), n, look)
	arm := box(-0.1, -0.7, -0.12, 0.2, 0.7, 0.24, all6(giantArmA), n, look)
	bright := mesh.Material{Color: dreadEyes}
	eye := box(-0.03, -0.09, -0.12, 0.06, 0.18, 0.24, all6(0), n, bright)
	front := box(-0.1, -0.09, -0.02, 0.2, 0.18, 0.04, all6(0), n, bright)
	return shape{
		stand: hip,
		hurt:  reddened(set.Dread),
		parts: []limb{
			{mesh: box(-0.95, hip-0.3, -1.6, 1.9, 1.9, 3.3, hide, n, look)},
			// The tail is three lengths, each from its root: the last two sway.
			{mesh: box(-0.6, -0.6, -1.5, 1.2, 1.2, 1.5, hide, n, look), up: hip + 0.85, front: -1.6},
			{mesh: box(-0.4, -0.4, -1.4, 0.8, 0.8, 1.4, hide, n, look), up: hip + 0.8, front: -3.0, sway: 0.15},
			{mesh: box(-0.22, -0.22, -1.2, 0.44, 0.44, 1.2, hide, n, look), up: hip + 0.75, front: -4.3, sway: 0.28},
			// The neck, then the head: the skull above, and the jaw hung open under it.
			{mesh: box(-0.6, hip+0.9, 1.4, 1.2, 1.3, 0.9, hide, n, look)},
			{mesh: box(-0.65, hip+1.35, 1.9, 1.3, 0.85, 1.8, head, n, look)},
			{mesh: box(-0.55, -0.4, 0, 1.1, 0.4, 1.55, jaw, n, look), up: hip + 1.3, front: 1.95, tilt: -0.22},
			{mesh: eye, right: -0.67, up: hip + 1.9, front: 2.5, eye: true},
			{mesh: eye, right: 0.67, up: hip + 1.9, front: 2.5, eye: true},
			{mesh: front, right: -0.38, up: hip + 1.9, front: 3.71, eye: true},
			{mesh: front, right: 0.38, up: hip + 1.9, front: 3.71, eye: true},
			{mesh: leg, right: -0.62, up: hip, front: -0.1, swing: 0.5},
			{mesh: leg, right: 0.62, up: hip, front: -0.1, swing: -0.5},
			{mesh: arm, right: -0.8, up: hip + 0.5, front: 1.35, tilt: 0.5, swing: -0.15},
			{mesh: arm, right: 0.8, up: hip + 0.5, front: 1.35, tilt: 0.5, swing: 0.15},
		},
	}
}

// shining is the light an eye that shines is drawn in: all of it, from every side.
var shining = []mesh.Light{mesh.Ambient{Level: 1}}

// clawEyes is the color of a quickclaw's eyes.
var clawEyes = engine.RGB(255, 214, 64)

// The tiles of the quickclaw's picture, from the left, as cmd/genassets draws them.
const (
	clawBack = iota
	clawSide
	clawHead
	clawLeg
	clawArm
)

// quickclawShape is a quickclaw: a hunter on two long legs, as tall as a person, its
// body level, with a stiff tail behind, a head held high, and two eyes that shine.
func quickclawShape(set *assets.Set) shape {
	look := mesh.Material{Color: engine.RGB(58, 62, 74), Texture: set.Quickclaw}
	const n = assets.QuickclawTiles
	const hip = 0.9 // how long its legs are

	hide := [6]int{clawSide, clawSide, clawSide, clawBack, clawSide, clawSide}
	head := [6]int{clawHead, clawHead, clawSide, clawBack, clawSide, clawSide}
	leg := box(-0.08, -hip, -0.08, 0.16, hip, 0.16, all6(clawLeg), n, look)
	arm := box(-0.06, -0.4, -0.1, 0.12, 0.4, 0.2, all6(clawArm), n, look)
	// An eye is a small flat box of one color. It is seen from the side of the head and
	// from in front of it, so each eye is two of them.
	bright := mesh.Material{Color: clawEyes}
	eye := box(-0.02, -0.04, -0.05, 0.04, 0.08, 0.1, all6(0), n, bright)
	front := box(-0.04, -0.04, -0.01, 0.08, 0.08, 0.02, all6(0), n, bright)
	return shape{
		stand: hip,
		hurt:  reddened(set.Quickclaw),
		parts: []limb{
			{mesh: box(-0.25, hip-0.05, -0.55, 0.5, 0.55, 1.1, hide, n, look)},
			{mesh: box(-0.11, -0.11, -1.3, 0.22, 0.22, 1.3, hide, n, look), up: hip + 0.3, front: -0.55, sway: 0.3},
			{mesh: box(-0.13, hip+0.3, 0.3, 0.26, 0.55, 0.26, hide, n, look)},
			{mesh: box(-0.16, hip+0.62, 0.3, 0.32, 0.34, 0.62, head, n, look)},
			{mesh: eye, right: -0.17, up: hip + 0.84, front: 0.62, eye: true},
			{mesh: eye, right: 0.17, up: hip + 0.84, front: 0.62, eye: true},
			{mesh: front, right: -0.09, up: hip + 0.84, front: 0.93, eye: true},
			{mesh: front, right: 0.09, up: hip + 0.84, front: 0.93, eye: true},
			{mesh: leg, right: -0.17, up: hip, front: -0.05, swing: 0.6},
			{mesh: leg, right: 0.17, up: hip, front: -0.05, swing: -0.6},
			{mesh: arm, right: -0.29, up: hip + 0.35, front: 0.35, swing: -0.25},
			{mesh: arm, right: 0.29, up: hip + 0.35, front: 0.35, swing: 0.25},
		},
	}
}

// hurtRed is the color a creature that was hit is flushed with, and hurtPart how much.
var hurtRed = engine.RGB(220, 30, 30)

const hurtPart = 0.45

// reddened returns a copy of a picture flushed with red, as a creature that is hit is
// drawn for a moment: a glonk mesh takes the colors of its picture as they are, and
// has no tint of its own.
func reddened(b *bitmap.Bitmap) *bitmap.Bitmap {
	red := bitmap.New(b.Width(), b.Height())
	for y := range b.Height() {
		for x := range b.Width() {
			if c := b.At(x, y); !c.IsDefault() {
				red.Set(x, y, engine.Mix(c, hurtRed, hurtPart))
			}
		}
	}
	return red
}

// box returns a mesh of a box with one corner at (x, y, z) and the given size, with a
// tile of the picture on each face.
func box(x, y, z, w, h, d float64, tiles [6]int, count int, look mesh.Material) mesh.Mesh {
	m := mesh.Mesh{Material: look}
	// Corner i is on the far side along x, y and z when bit 1, 2 and 4 of i is set, as
	// in glonk's Cube, whose faces these are: each wound so that it faces out.
	for i := range 8 {
		m.Vertices = append(m.Vertices, geometry.Vector3{
			X: x + w*float64(i&1), Y: y + h*float64(i>>1&1), Z: z + d*float64(i>>2&1),
		})
	}
	quads := [6][4]int{
		world.FaceXNeg: {0, 4, 6, 2}, world.FaceXPos: {1, 3, 7, 5},
		world.FaceYNeg: {0, 1, 5, 4}, world.FaceYPos: {2, 6, 7, 3},
		world.FaceZNeg: {0, 2, 3, 1}, world.FaceZPos: {4, 5, 7, 6},
	}

	// place says where on the picture a corner of a face is: across the face along one
	// of its two axes and down it along the other, inside the face's tile. It stays a
	// little inside the tile, so that a pixel on the edge of a face is not taken from
	// the tile beside it.
	place := func(side world.Face, corner int) geometry.Vector2 {
		v := m.Vertices[corner]
		fx, fy, fz := (v.X-x)/w, (v.Y-y)/h, (v.Z-z)/d
		var across, down float64
		switch side >> 1 {
		case 0:
			across, down = fz, 1-fy
		case 1:
			across, down = fx, fz
		default:
			across, down = fx, 1-fy
		}
		inside := func(f float64) float64 { return 0.02 + 0.96*f }
		return geometry.Vector2{X: (float64(tiles[side]) + inside(across)) / float64(count), Y: inside(down)}
	}
	for side, q := range quads {
		side := world.Face(side)
		m.Faces = append(m.Faces, [3]int{q[0], q[1], q[2]}, [3]int{q[0], q[2], q[3]})
		m.FaceTexCoords = append(m.FaceTexCoords,
			[3]geometry.Vector2{place(side, q[0]), place(side, q[1]), place(side, q[2])},
			[3]geometry.Vector2{place(side, q[0]), place(side, q[2]), place(side, q[3])},
		)
	}
	m.UpdateBounds() // so that glonk can skip it when it is off screen
	return m
}

// lightsIn returns the lights a glonk mesh is drawn with where the light is a brightness
// from 0 to 1. The ambient light lights every face, and the other fills in limb of what
// that leaves dark: so a face toward the sun or the moon is as bright as the light, and
// one turned away from it seven tenths of that.
func lightsIn(brightness float64, sky *Sky) [2]mesh.Light {
	lit := min(max(brightness, 0.01), 1)
	return [2]mesh.Light{
		mesh.Ambient{Level: 0.7 * lit},
		mesh.Directional{Direction: sky.giver().Neg(), Intensity: 0.3 * lit / (1 - 0.7*lit)},
	}
}

// Draw draws the beasts as the camera sees them, over what is in the canvas. Call it
// after the blocks, and before anything that is seen through.
//
// A beast is lit about as the blocks round it are: its top as bright as the light it
// stands in, and its side away from the sun or the moon less.
func (b *Beasts) Draw(c *engine.Canvas, cam mesh.Camera, all []Beast, sky *Sky) {
	for _, one := range all {
		if one.Kind >= Kinds || one.Fall >= 1 {
			continue
		}
		s := &b.shapes[one.Kind]
		b.lights = lightsIn(one.Light, sky)
		turned := mesh.Camera{Yaw: one.Yaw}
		ahead, aside := turned.Forward(), turned.Right()

		// A young beast is the grown one made smaller, and a dead one sinks and fades.
		size := 1.0
		if one.Young {
			size = youngScale
		}
		step := math.Sin(one.Stride)
		var opacity float64 // solid, to glonk
		if one.Fall > 0 {
			step, opacity = 0, 1-one.Fall
		}
		for i := range s.parts {
			p := &s.parts[i]
			m := p.mesh
			if one.Shorn && p.hasBare {
				m = p.bare
			}
			at := one.Pos.Add(aside.Scale(p.right * size)).Add(ahead.Scale(p.front * size))
			at.Y += (p.up - one.Fall*s.stand) * size
			m.Transform = mesh.Transform{
				Position: at,
				Yaw:      one.Yaw + p.sway*(step+lashMore*one.Lash),
				Pitch:    p.tilt + p.swing*step - p.nod*one.Lower,
			}
			if one.Young {
				m.Transform.Scale = size
			}
			m.Material.Opacity = opacity
			switch {
			case p.eye && one.Glow && one.Fall == 0:
				m.DrawShadedPixels(c, cam, shining...)
				continue
			case one.Hurt && !p.eye:
				m.Material.Texture = s.hurt
			}
			m.DrawShadedPixels(c, cam, b.lights[:]...)
		}
	}
}

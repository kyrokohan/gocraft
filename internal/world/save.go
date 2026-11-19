package world

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"slices"

	"glonk.dev/geometry"
)

// A saved world is a folder: world.json, which says what the world is and where the
// player was, and chunks, a folder with one file for each chunk the player changed. A
// chunk that was never changed is not saved: the seed makes it again.

// Meta is what a saved world holds beside its blocks.
type Meta struct {
	// Version is the layout of the save. This game writes and reads 2.
	Version int `json:"version"`

	// Seed is the number the terrain is made from.
	Seed int64 `json:"seed"`

	// Pos is the point between the player's feet, and Yaw and Pitch where it looks.
	Pos    geometry.Vector3 `json:"pos"`
	Yaw    float64          `json:"yaw"`
	Pitch  float64          `json:"pitch"`
	Flying bool             `json:"flying"`

	// Slot is the hotbar slot in use, from 0.
	Slot int `json:"slot"`

	// Days is how long the world has been played, in days of the game: the clock, from
	// the morning the world began. A save from before there was a clock has none, and
	// so begins that morning again.
	Days float64 `json:"days"`

	// Mode is how the world is played: Survival or Creative. A save from before there
	// were modes has none, and is played as Creative.
	Mode string `json:"mode,omitempty"`

	// Items is what the player carries, slot by slot: the hotbar and then the pack. A save
	// from before there were items has none.
	Items []Stack `json:"items,omitempty"`

	// Health, Food and Air are how the player's body is, in survival.
	Health float64 `json:"health,omitempty"`
	Food   float64 `json:"food,omitempty"`
	Air    float64 `json:"air,omitempty"`

	// Drops are the things lying on the ground to be picked up.
	Drops []Drop `json:"drops,omitempty"`

	// Mossbacks are the animals round the player. A save from before they were kept has
	// none, and a herd is found round the player again.
	Mossbacks []Mossback `json:"mossbacks,omitempty"`

	// Crates and Kilns are what the crates and the kilns of the world hold, wherever
	// they are.
	Crates []CrateStore `json:"crates,omitempty"`
	Kilns  []KilnStore  `json:"kilns,omitempty"`

	// Bed is the bedroll the player last lay on, which it wakes at; nil for none.
	Bed *Cell `json:"bed,omitempty"`
}

// Cell is the place of a block in the world.
type Cell struct {
	X int `json:"x"`
	Y int `json:"y"`
	Z int `json:"z"`
}

// valid reports whether the cell is one a world has.
func (c Cell) valid() bool {
	return c.Y >= 0 && c.Y < Height && abs(c.X) <= 1e9 && abs(c.Z) <= 1e9
}

// CrateSlots is how many slots a crate has.
const CrateSlots = 18

// CrateStore is what a crate holds, slot by slot.
type CrateStore struct {
	Cell
	Items []Stack `json:"items"`
}

// KilnStore is what a kiln holds, and how far it has got with it.
type KilnStore struct {
	Cell
	In   Stack `json:"in"`   // what is to be cooked
	Fuel Stack `json:"fuel"` // what is to be burned
	Out  Stack `json:"out"`  // what has been made

	// Burn is how many seconds the fuel burning has left, of Burns in all, and Cooked how
	// many seconds what is in has cooked.
	Burn   float64 `json:"burn,omitempty"`
	Burns  float64 `json:"burns,omitempty"`
	Cooked float64 `json:"cooked,omitempty"`
}

// The ways a world is played, as Meta.Mode names them.
const (
	// Survival is played with what the player finds, and the player can be hurt.
	Survival = "survival"

	// Creative has every block without end, and nothing hurts.
	Creative = "creative"
)

// Drop is a stack lying on the ground, to be picked up.
type Drop struct {
	Stack
	Pos geometry.Vector3 `json:"pos"`

	// Age is how many seconds it has lain there.
	Age float64 `json:"age"`
}

// Mossback is a mossback as a save holds it.
type Mossback struct {
	Pos geometry.Vector3 `json:"pos"`
	Yaw float64          `json:"yaw"`

	// Regrow is how many seconds its moss takes to grow back after it was shorn, and 0
	// while it has its moss.
	Regrow float64 `json:"regrow,omitempty"`
}

// The most a save may hold of what is carried, of what lies on the ground, and of
// animals: far more than a game makes, and few enough that a file cannot make the game
// hold a lot.
const (
	maxSlots     = 256
	maxDrops     = 1 << 14
	maxMossbacks = 1 << 10
	maxStores    = 1 << 12 // crates, and kilns
)

// check returns what is wrong with what a save says beside its blocks, or "".
func (m Meta) check() string {
	switch {
	case m.Version != saveVersion:
		return fmt.Sprintf("it is version %d, and this game reads version %d", m.Version, saveVersion)
	case !finite(m.Pos.X, m.Pos.Y, m.Pos.Z, m.Yaw, m.Pitch):
		return "its player is nowhere"
	case !finite(m.Days):
		return "its clock does not tell a time"
	case math.Abs(m.Pos.X) > 1e9 || math.Abs(m.Pos.Z) > 1e9:
		// A place so far out that it cannot be counted in blocks is no place either.
		return "its player is too far away to be anywhere"
	case m.Mode != "" && m.Mode != Survival && m.Mode != Creative:
		return fmt.Sprintf("it is played in a way this game does not have: %q", m.Mode)
	case !finite(m.Health, m.Food, m.Air):
		return "its player's body is not one"
	case len(m.Items) > maxSlots || len(m.Drops) > maxDrops || len(m.Mossbacks) > maxMossbacks ||
		len(m.Crates) > maxStores || len(m.Kilns) > maxStores:
		return "it holds more things than a game can make"
	case m.Bed != nil && !m.Bed.valid():
		return "its player's bed is nowhere"
	}
	for _, c := range m.Crates {
		if !c.valid() || len(c.Items) > CrateSlots || slices.ContainsFunc(c.Items, func(s Stack) bool { return !s.valid() }) {
			return "one of its crates is not one this game can make"
		}
	}
	for _, k := range m.Kilns {
		if !k.valid() || !k.In.valid() || !k.Fuel.valid() || !k.Out.valid() ||
			!finite(k.Burn, k.Burns, k.Cooked) || k.Burn < 0 || k.Burns < 0 || k.Cooked < 0 {
			return "one of its kilns is not one this game can make"
		}
	}
	for _, s := range m.Items {
		if !s.valid() {
			return fmt.Sprintf("its player carries %d of a kind of item %d, which a slot cannot hold", s.Count, s.Item)
		}
	}
	for _, d := range m.Drops {
		if !d.valid() || d.Count == 0 || !finite(d.Pos.X, d.Pos.Y, d.Pos.Z, d.Age) {
			return "something lying on its ground is not a thing this game has"
		}
	}
	for _, a := range m.Mossbacks {
		if !finite(a.Pos.X, a.Pos.Y, a.Pos.Z, a.Yaw, a.Regrow) || a.Regrow < 0 {
			return "one of its mossbacks is nowhere"
		}
	}
	return ""
}

const (
	saveVersion = 3
	metaName    = "world.json"
	chunkFolder = "chunks"
	chunkMagic  = "GCCHUNK1" // opens every chunk file; its last character is the version of the layout

	// A chunk packs to a few thousand bytes. A file far larger than a chunk can be is
	// not read at all.
	maxChunkFile = 1 << 20
)

// errBadSave is wrapped by every error for a file that is not a save this game can use.
var errBadSave = errors.New("not a usable save")

// ErrOlder is wrapped by the error for a world that an older version of the game saved.
// Such a world is whole, but its blocks and things are not the ones this game has.
var ErrOlder = errors.New("it was saved by an older version of the game")

func isNotExist(err error) bool {
	return errors.Is(err, fs.ErrNotExist)
}

func finite(values ...float64) bool {
	for _, v := range values {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	return true
}

func chunkPath(dir string, k key) string {
	return filepath.Join(dir, chunkFolder, fmt.Sprintf("c.%d.%d", k.x, k.z))
}

// pack turns the blocks of a chunk into the bytes of a chunk file.
func pack(blocks []Block) []byte {
	raw := make([]byte, len(blocks))
	for i, b := range blocks {
		raw[i] = byte(b)
	}
	var out bytes.Buffer
	out.WriteString(chunkMagic)
	zw := gzip.NewWriter(&out)
	zw.Write(raw) // writing into memory cannot fail
	zw.Close()
	return out.Bytes()
}

// unpack reads the bytes of a chunk file into the blocks of a chunk. What the bytes
// hold is checked before it is used: they come from a file, which anything may have
// written.
func unpack(data []byte, blocks []Block) error {
	if len(data) < len(chunkMagic) || string(data[:len(chunkMagic)]) != chunkMagic {
		return fmt.Errorf("%w: it does not start as a chunk of this version does", errBadSave)
	}
	zr, err := gzip.NewReader(bytes.NewReader(data[len(chunkMagic):]))
	if err != nil {
		return fmt.Errorf("%w: its blocks cannot be read", errBadSave)
	}
	// The blocks are read into a list the size of one chunk, so a file can never make
	// the game hold more, however much it claims to unpack to.
	raw := make([]byte, len(blocks))
	if _, err := io.ReadFull(zr, raw); err != nil {
		return fmt.Errorf("%w: its blocks are cut short", errBadSave)
	}
	// The end of the blocks must be the end of the file. Reading on to it is also what
	// makes gzip check the blocks against the checksum it keeps after them.
	if n, err := zr.Read(make([]byte, 1)); n != 0 || err != io.EOF {
		return fmt.Errorf("%w: its blocks are damaged, or there is more after them", errBadSave)
	}
	for i, b := range raw {
		if !Block(b).Known() {
			return fmt.Errorf("%w: it holds a block of kind %d, which this game does not have", errBadSave, b)
		}
		blocks[i] = Block(b)
	}
	return nil
}

// readChunk reads the file of a chunk from a saved world into blocks. A chunk that was
// never saved gives an error that wraps fs.ErrNotExist.
func readChunk(dir string, k key, blocks []Block) error {
	path := chunkPath(dir, k)
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxChunkFile+1))
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if len(data) > maxChunkFile {
		return fmt.Errorf("%s: %w: it is far larger than a chunk can be", path, errBadSave)
	}
	if err := unpack(data, blocks); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

// writeFile writes data at path. The data is written to another file beside it first
// and then put in its place, so a write that fails part way, or a machine that stops,
// leaves the file that was there before.
func writeFile(path string, data []byte) (err error) {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".saving-*")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
		}
	}()
	if _, err = tmp.Write(data); err != nil {
		return err
	}
	if err = tmp.Sync(); err != nil {
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// clearChunks removes the chunk files of a saved world, and nothing else: a file in the
// folder that is not named as a chunk is left where it is.
func clearChunks(dir string) error {
	entries, err := os.ReadDir(filepath.Join(dir, chunkFolder))
	if err != nil {
		return err
	}
	for _, e := range entries {
		var x, z int
		var rest string
		if n, _ := fmt.Sscanf(e.Name(), "c.%d.%d%s", &x, &z, &rest); n != 2 || !e.Type().IsRegular() {
			continue
		}
		if err := os.Remove(filepath.Join(dir, chunkFolder, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

// Save writes the world into a folder, making it if it is not there: every chunk that
// was changed since the last save, and then what m says of the world and the player.
// The seed and the version in m are set from the world. A chunk whose file could not be
// read when it was loaded is not written, so that the file stays as it was.
//
// A world saved into the folder of another world takes its place: the other world's
// chunks are removed first.
func (w *World) Save(dir string, m Meta) error {
	if !finite(m.Pos.X, m.Pos.Y, m.Pos.Z, m.Yaw, m.Pitch) {
		return errors.New("save: the player is nowhere a save can hold")
	}
	if err := os.MkdirAll(filepath.Join(dir, chunkFolder), 0o755); err != nil {
		return err
	}
	if w.dir != dir {
		// This world was not opened from the folder, so whatever chunks are in it are
		// another world's, which this one takes the place of. Left there, they would
		// be read as this world's own.
		if err := clearChunks(dir); err != nil {
			return err
		}
	}

	for k, packed := range w.stash {
		if err := writeFile(chunkPath(dir, k), packed); err != nil {
			return err
		}
		delete(w.stash, k) // it is in the folder now, and is read from there
	}
	for at := range w.loaded {
		if !w.loaded[at] || !w.dirty[at] || w.held[at] {
			continue
		}
		k := w.owner[at]
		if err := writeFile(chunkPath(dir, k), pack(w.copyOut(k))); err != nil {
			return err
		}
		w.dirty[at] = false
	}

	m.Version, m.Seed = saveVersion, w.Seed
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err := writeFile(filepath.Join(dir, metaName), append(data, '\n')); err != nil {
		return err
	}
	w.dir = dir
	return nil
}

// Open reads a saved world from its folder, with the chunks round the player loaded. A
// folder with no world in it gives an error that wraps fs.ErrNotExist. A world.json that
// is not one, or is of another version, or holds something this game cannot use, gives
// an error and never a panic.
func Open(dir string) (*World, Meta, error) {
	m, err := Peek(dir)
	if err != nil {
		return nil, Meta{}, err
	}
	return newWorld(m.Seed, Terrain(m.Seed), dir, m.Pos.X, m.Pos.Z), m, nil
}

// Peek reads what a saved world holds beside its blocks, as Open does, without making
// the world: to list the worlds there are.
func Peek(dir string) (Meta, error) {
	path := filepath.Join(dir, metaName)
	data, err := os.ReadFile(path)
	if err != nil {
		return Meta{}, err
	}
	bad := func(why string) (Meta, error) {
		return Meta{}, fmt.Errorf("%s: %w: %s", path, errBadSave, why)
	}

	var m Meta
	if err := json.Unmarshal(data, &m); err != nil {
		return bad("it cannot be read: " + err.Error())
	}
	if m.Version < saveVersion {
		return Meta{}, fmt.Errorf("%s: %w: %w", path, errBadSave, ErrOlder)
	}
	if why := m.check(); why != "" {
		return bad(why)
	}
	return m, nil
}

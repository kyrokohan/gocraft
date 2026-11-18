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
}

const (
	saveVersion = 2
	metaName    = "world.json"
	chunkFolder = "chunks"
	chunkMagic  = "GCCHUNK1" // opens every chunk file; its last character is the version of the layout

	// A chunk packs to a few thousand bytes. A file far larger than a chunk can be is
	// not read at all.
	maxChunkFile = 1 << 20
)

// errBadSave is wrapped by every error for a file that is not a save this game can use.
var errBadSave = errors.New("not a usable save")

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
		return fmt.Errorf("%w: it does not start as a gocraft chunk of this version does", errBadSave)
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
	path := filepath.Join(dir, metaName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, Meta{}, err
	}
	bad := func(why string) (*World, Meta, error) {
		return nil, Meta{}, fmt.Errorf("%s: %w: %s", path, errBadSave, why)
	}

	var m Meta
	if err := json.Unmarshal(data, &m); err != nil {
		return bad("it cannot be read: " + err.Error())
	}
	if m.Version != saveVersion {
		return bad(fmt.Sprintf("it is version %d, and this game reads version %d", m.Version, saveVersion))
	}
	if !finite(m.Pos.X, m.Pos.Y, m.Pos.Z, m.Yaw, m.Pitch) {
		return bad("its player is nowhere")
	}
	if !finite(m.Days) {
		return bad("its clock does not tell a time")
	}
	// A place so far out that it cannot be counted in blocks is no place either.
	if math.Abs(m.Pos.X) > 1e9 || math.Abs(m.Pos.Z) > 1e9 {
		return bad("its player is too far away to be anywhere")
	}

	return newWorld(m.Seed, Terrain(m.Seed), dir, m.Pos.X, m.Pos.Z), m, nil
}

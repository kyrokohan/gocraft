package game

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
)

// Settings are how the player likes the game, kept between runs in a file of their own,
// apart from any world.
type Settings struct {
	// MouseSpeed is how far a drag turns the view, as a multiple of the angle the pointer
	// moved through: at 1 the world would stay under the pointer.
	MouseSpeed float64 `json:"mouseSpeed"`

	// ViewDistance is how far the world is drawn, in blocks.
	ViewDistance int `json:"viewDistance"`

	// FieldOfView is how much of the world the view takes in, from its top to its bottom,
	// in degrees.
	FieldOfView int `json:"fieldOfView"`

	// Volume is how loud the sounds are, from 0, none, to 100, as they were made.
	Volume int `json:"volume"`

	// View is which view the world is seen in: viewEyes or viewBehind. V changes it.
	View string `json:"view"`
}

// The values each setting can take, and the ones a player starts with.
var (
	mouseSpeeds   = []float64{0.5, 0.75, 1, 1.25, 1.5, 2, 2.5, 3}
	viewDistances = []int{32, 48, 64}
	fieldsOfView  = []int{50, 60, 70, 80, 90}
	volumes       = []int{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100}

	defaultSettings = Settings{MouseSpeed: 1.5, ViewDistance: 64, FieldOfView: 60, Volume: 70, View: viewEyes}
)

// nearest returns the value of a list nearest v.
func nearest[T int | float64](list []T, v T) T {
	best := list[0]
	for _, x := range list {
		if math.Abs(float64(x-v)) < math.Abs(float64(best-v)) {
			best = x
		}
	}
	return best
}

// fit returns the settings with each one the value of its list nearest what it is, so
// that a file written by hand, or by another version, gives settings the game has.
func (s Settings) fit() Settings {
	if math.IsNaN(s.MouseSpeed) {
		s.MouseSpeed = defaultSettings.MouseSpeed
	}
	s.MouseSpeed = nearest(mouseSpeeds, s.MouseSpeed)
	s.ViewDistance = nearest(viewDistances, s.ViewDistance)
	s.FieldOfView = nearest(fieldsOfView, s.FieldOfView)
	s.Volume = nearest(volumes, s.Volume)
	if s.View != viewBehind {
		s.View = viewEyes
	}
	return s
}

// volume is the volume from 0 to 1, as the mixer takes it.
func (s Settings) volume() float64 {
	return float64(s.Volume) / 100
}

// fov is the field of view in radians, as a camera takes it.
func (s Settings) fov() float64 {
	return float64(s.FieldOfView) * math.Pi / 180
}

// loadSettings reads the settings from a file. A file that is not there, or cannot be
// read, gives the settings a player starts with: settings are not worth stopping for.
func loadSettings(path string) Settings {
	s := defaultSettings
	if path == "" {
		return s
	}
	if data, err := os.ReadFile(path); err == nil {
		if json.Unmarshal(data, &s) != nil {
			s = defaultSettings
		}
	}
	return s.fit()
}

// save writes the settings to a file, beside the old one and then in its place.
func (s Settings) save(path string) error {
	if path == "" {
		return nil
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// step moves the setting on line i of the settings screen by one value along its list,
// either way, round from the end to the start.
func (s *Settings) step(i, by int) {
	turn := func(n, at int) int { return ((at+by)%n + n) % n }
	switch i {
	case 0:
		s.MouseSpeed = mouseSpeeds[turn(len(mouseSpeeds), slices.Index(mouseSpeeds, s.MouseSpeed))]
	case 1:
		s.ViewDistance = viewDistances[turn(len(viewDistances), slices.Index(viewDistances, s.ViewDistance))]
	case 2:
		s.FieldOfView = fieldsOfView[turn(len(fieldsOfView), slices.Index(fieldsOfView, s.FieldOfView))]
	case 3:
		s.Volume = volumes[turn(len(volumes), slices.Index(volumes, s.Volume))]
	}
}

// settingLines is how many lines of the settings screen are settings: the rest go back.
const settingLines = 4

// lines are the lines of the settings screen.
func (s Settings) lines() []string {
	line := func(name, value string) string { return fmt.Sprintf("%-15s ◀ %-9s ▶", name, value) }
	return []string{
		line("Mouse speed", strconv.FormatFloat(s.MouseSpeed, 'f', -1, 64)),
		line("View distance", strconv.Itoa(s.ViewDistance)+" blocks"),
		line("Field of view", strconv.Itoa(s.FieldOfView)+"°"),
		line("Volume", strconv.Itoa(s.Volume)+"%"),
		itemBack,
	}
}

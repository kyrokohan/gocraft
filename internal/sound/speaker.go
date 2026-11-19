package sound

import (
	"errors"
	"time"

	"github.com/ebitengine/oto/v3"
)

// Speaker plays a mixer on the machine's sound device, until it is closed.
type Speaker struct {
	player *oto.Player // kept, as oto stops a player nothing holds
}

// Open starts the sound device playing what a mixer makes. It can be opened once in a
// run of the program. A machine with no sound device gives an error, and the game is
// played without sound.
func Open(m *Mixer) (*Speaker, error) {
	ctx, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:      Rate,
		ChannelCount:    2,
		Format:          oto.FormatFloat32LE,
		BufferSize:      60 * time.Millisecond, // little enough that a sound comes with what makes it
		ApplicationName: "skyspear",
	})
	if err != nil {
		return nil, err
	}
	select {
	case <-ready:
	case <-time.After(3 * time.Second):
		return nil, errors.New("the sound device did not start")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p := ctx.NewPlayer(m)
	p.SetBufferSize(Rate / 16 * 8) // oto reads half a second ahead unless told, and a sound would come that late
	p.Play()
	return &Speaker{player: p}, nil
}

// Close stops the sound.
func (s *Speaker) Close() error {
	return s.player.Close()
}

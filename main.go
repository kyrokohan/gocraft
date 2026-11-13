// Command gocraft is a block-building sandbox that runs in the terminal.
package main

import (
	"context"
	"flag"
	"fmt"
	"image/png"
	"os"

	"glonk.dev/engine"

	"github.com/kyrokohan/gocraft/internal/game"
	"github.com/kyrokohan/gocraft/internal/render"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	seed := flag.Int64("seed", 1, "the number the world is made from")
	shot := flag.String("shot", "", "draw one frame into this PNG file and quit, instead of playing")
	flag.Parse()

	g, err := game.New(*seed)
	if err != nil {
		return err
	}
	if *shot != "" {
		return writeShot(g, *shot)
	}
	return engine.Run(context.Background(), g, engine.Config{FPS: 30})
}

// writeShot draws one frame of the game as a terminal of 240 by 67 cells would show it
// and saves its pixels, each four across, so the picture can be looked at without one.
// Text is not in it: it is drawn in cells, not pixels.
func writeShot(g engine.Game, path string) error {
	fb, err := engine.NewFrameBuffer(240, 67)
	if err != nil {
		return err
	}
	g.Draw(fb)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, render.Picture(fb.Canvas(), 4)); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

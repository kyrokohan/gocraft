// Command gocraft is a block-building sandbox that runs in the terminal.
package main

import (
	"context"
	"fmt"
	"os"

	"glonk.dev/engine"

	"github.com/kyrokohan/gocraft/internal/game"
)

func main() {
	if err := engine.Run(context.Background(), game.New(), engine.Config{FPS: 30}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

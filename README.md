# gocraft

A block-building sandbox that runs in the terminal, built on [glonk](https://glonk.dev).

Walk a world with no end, made from a seed: seas and shores, plains in flower, forest, desert, snowy tundra and bare crags, with caves and ores under them, drifting clouds over them and mossbacks grazing on the grass. The sun crosses the sky and sets, the stars come out, and the caves are dark until you bring a torch. Break blocks and place your own, with the keys or the mouse, look round by dragging, and find the world as you left it the next time. The art is original and drawn by code in this repo.

> Creative play only so far. See [Not in yet](#not-in-yet).

## Run

```
go run .
```

Requires Go 1.26 or newer.

- **Use a terminal that reports key releases:** Ghostty, kitty, WezTerm, foot, iTerm2 or Alacritty. There you can walk and turn at once. In other terminals (Apple's Terminal, for one) only one key stays held at a time, which is playable but clumsy, and the game says so on screen. This is how glonk reads keys; see its README.
- **A bigger window is a sharper picture.** Every cell is two pixels, one above the other, so 200 by 60 cells is a picture of 200 by 120 pixels.
- **True color looks best.** Inside tmux or over ssh, `export COLORTERM=truecolor` if your terminal can show it.

## Keys and mouse

| Keys | Action |
|---|---|
| W A S D | walk, strafe |
| Arrows | look |
| Space | jump; swim up; rise when flying |
| X | sink when flying |
| F | flying on and off |
| Q / E | break / place the block aimed at (hold to repeat) |
| 1 to 9, Tab | pick a block; next block |
| T | move the clock on six hours |
| F3 | frame numbers: where the time of a frame goes and how much is sent, the seed, the day and the hour |
| Esc | pause menu, with these keys on it |

| Mouse | Action |
|---|---|
| Move | aim at the block under the pointer |
| Hold left and drag | turn the view |
| Left click / right click | break / place (hold still to repeat) |
| Wheel | pick a block |
| Click a hotbar slot | pick that block |

- **Whichever you used last aims.** Move or click the mouse and the pointer aims. Press an arrow key and the middle of the view aims again, with its crosshair.
- **Drag to look.** A terminal reports the pointer in cells and cannot lock it, so the view cannot follow the mouse as it does in a window. Holding the left button and dragging turns it, a cell at a time; a press let go where it is stays a click. The arrows turn it too.
- While the game runs, the terminal cannot select text with the mouse. `-mouse=false` gives the pointer back to it.
- Letters work in either case, and Ctrl-C quits (and saves).
- A day lasts ten minutes of play, and the clock stands still in a pause. A new world starts in the morning.

## Saving

The world is saved when you choose **Save and quit**, and also on Ctrl-C or when the window closes. Next time the title offers **Continue**. **New world** asks before it takes the place of a saved one.

- A world is a folder: `gocraft/worlds/world` in your config folder (`~/Library/Application Support` on macOS, `~/.config` on Linux), or the one named with `-world`. It holds `world.json` (the seed, the day and the hour, and where you were) and one small file for each chunk you changed. The rest of the world is made again from the seed.
- Every file is written beside the old one and then put in its place, so a write that fails leaves the old file.
- A `world.json` that cannot be read stops the game with an error instead of being written over. Move the folder away, or name another with `-world`.
- A chunk file that cannot be read does not stop the game. The chunk is made again from the seed, a line on screen says so, and the file is never written over, so what you build in that chunk is not kept.
- **The one world of the first version** (`gocraft/world.sav`) cannot be played by this one. It is left where it is, untouched.
- The mossbacks are not saved yet: a herd is found round you each time a world loads.

## Flags

| Flag | Meaning |
|---|---|
| `-seed n` | the number a new world is made from; the same number makes the same world. 0, the default, takes one from the clock |
| `-world folder` | where the world is saved |
| `-mouse=false` | leave the pointer to the terminal |
| `-shot file.png` | draw one frame of play into a PNG and quit, to look at it without a terminal. A letter is a mark in its color where it stands |

## How it is drawn

Two things draw into one frame, and glonk holds them together.

- **The blocks are rays.** Every pixel of glonk's half-block `Canvas` sends a ray through the grid of blocks and takes the color of the tile it lands on. That is quick, it lets a far block be drawn from a smaller copy of its tile so that it does not flicker, and it does water exactly: you see the bottom through it. A plant is two pictures that cross in its cell, and the rays meet those too. The rays are the ones `Camera.RayPixels` gives, they end at the camera's `Far`, they fade by `Fog.Apply`, and they store glonk's depth with `Canvas.SetDepth`.
- **The things are glonk meshes.** The clouds are see-through slabs (`Material.Opacity`) and a mossback is six boxes with a picture on their faces (`Material.Texture`). glonk lights them, fades them by the same fog, and hides them behind any block that is nearer, through the depth the rays stored. The outline round the block you aim at is a cube's edges drawn with `DrawOnSurfacePixels`.
- **Light is worked out once, not per pixel.** Each cell holds a level from 0 to 15 of light from the sky and one of light from torches and glowing crystal, and a face is drawn as bright as the cell in front of it. The hour then says how bright and what color each level looks: white by day, warm at sunset, dim blue by night, and always warm by a torch. The mossbacks are lit by the cell they stand in, and the clouds by the hour.

On one core of an Apple M5 Pro, at 240 by 134 pixels, the blocks take 2.5 to 4 ms, the clouds 0.3 ms and twelve mossbacks 0.1 ms; a whole frame with the HUD 5 ms.

## A world with no end

The rays look up millions of blocks a frame, so a block has to be one index into one list. The world is therefore not a map of pieces: only the part round you is held, in a window of 256 by 256 columns that slides with you. A block at `x, z` lives at `x & 255, z & 255`, so the list wraps round and nothing is copied when the window moves.

The world is made a chunk at a time, 16 by 16 columns and 128 high, each from the seed and its place alone. So a chunk is the same whenever it is made, a tree on the line between two chunks is built half by each, and only the chunks you changed need saving. A chunk takes 0.2 ms to make and they come in two a frame, the nearest first, which keeps ahead of flying flat out.

Four slow noises say how high the land stands, how rough it is, how warm and how wet, and from those each column is sea, shore, plains, forest, desert, tundra or crags. Caves and ores come from noise in three dimensions, worked out on a coarse grid that is the same for the whole world.

## Light

Light from the sky falls straight down to the first block that stops it, dimmed a level by each leaf or water block on the way. From there light, and the light of a torch, loses a level for each step it takes, so it spreads under an overhang, round a corner and down a shaft, and is gone 15 steps from where it was full. This is worked out by a flood fill over the window, which holds one byte of light beside each block:

- **When a chunk comes in**, its own light spreads into the chunks beside it and theirs into it. Light only goes up then, so the order chunks come in does not matter. It takes 0.05 ms beside the 0.2 ms of making the chunk.
- **When you change a block**, the light is worked out again in the box round it that the change can reach: 14 blocks every way, and down the column under it, whose sky it may open or shut. It takes 0.2 ms.
- Light is not saved. It is worked out again from the blocks when the world loads.

Tests check the light against a plain, slow flood fill: in chunks of jumbled blocks that come in in any order, after random changes, and across the edges of chunks.

glonk also runs the loop and the terminal, reads the keys and the mouse, draws the title picture (`bitmap`), the menus and the HUD (`widget`), puts the hotbar's pictures down over its panel (`Canvas.ComposeRect`), reports the frame numbers (`Stats`), draws the `-shot` picture (`FrameBuffer.Image`), and runs every test without a terminal.

## Pictures

`internal/assets/img` holds the three pictures the game uses: a strip of 40 block tiles, the title, and the mossback's four tiles. All are drawn by `cmd/genassets` from a fixed hash, so they come out the same every time:

```
make assets                                   # draw them again
go run ./cmd/genassets -preview sheet.png     # and an enlarged sheet to look at
```

A test fails if the pictures in the repo are not what the generator draws.

## Development

```
make check    # formatting, vet, and tests with the race detector
make bench    # the time to draw the blocks, the clouds, the creatures and a whole frame
```

The race detector needs cgo. Where there is no C compiler, run `make check RACE=`.

| Package | Purpose |
|---|---|
| `internal/world` | the blocks round the player in a sliding window and their light, the land a seed makes, rays through the grid, saved worlds |
| `internal/render` | the blocks drawn by rays into a glonk `Canvas`, the sky at each hour, and the clouds and creatures drawn into it as glonk meshes |
| `internal/game` | the game glonk runs: the player, the creatures, building, the mouse, the HUD, the title and pause menus |
| `internal/assets` | the pictures, read once at start-up, each block tile in five sizes |
| `internal/tile` | the names of the tiles in the strip |
| `cmd/genassets` | draws the pictures |

gocraft uses glonk v0.12.0.

## Not in yet

- Flowing water, and shadows cast to the side by the sun: its light falls straight down.
- Torches on walls: a torch stands on the ground, as a plant does.
- A way to place the new kinds of block: the hotbar holds nine, with a torch in place of leaves.
- More kinds of creature, creatures in the save, anything to do to them.
- Survival: crafting, counted blocks, tools, health.
- A view that follows the mouse without a button held, and sound.

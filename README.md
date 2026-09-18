# gopherttype

A terminal typing game with word-count and timed rounds, plus a **Vim
challenge** mode that tests how well
you know your motions and operators, all watched by a gopher that follows your
input and reacts to how you play.

![gopherttype demo: choose a mode, type with the gopher, and see your results](docs/demo/showcase.gif)

## Install

### With Go

```sh
go install github.com/mkhamat/gopherttype/cmd/gopherttype@latest
gopherttype
```

Requires Go 1.27.1 or newer.

### Without Go

Download your platform's archive from [Releases](https://github.com/whoisnjoguu/gopherttype/releases).
Choose `arm64` for Apple Silicon or ARM64 machines, and `amd64` for Intel/AMD 64-bit machines.

- **macOS / Linux:** extract the `.tar.gz` and run `./gopherttype` in a terminal.
  Move the binary into a directory on your `PATH` to run it from anywhere.
- **Windows:** extract the `.zip` and run `.\gopherttype.exe` in PowerShell.

Each release includes `checksums.txt` for verifying downloads. macOS and Windows
binaries are unsigned, so your operating system may prompt before running them.

## Run from source

Requires Go 1.27.1

```sh
go run ./cmd/gopherttype
```

or build

```sh
go build ./cmd/gopherttype
./gopherttype
```

## Modes

On the home screen, ↑↓ pick a mode and ←→ pick its option:

- **time** / **words** — the classic typing game (option sets the duration or
  word count).
- **vim** — a round of procedurally generated Vim challenges (option
  sets the difficulty: easy, medium, hard, or mixed).

## Vim challenges

Each challenge shows a small buffer with the cursor's starting position and a
task such as *"delete the word under the cursor"*, *"change 'counter' to
'total'"*, *"search for 'needle'"* or *"replace every 'foo' on this line with
'bar'"*. Challenges are generated fresh from a word list each round, so no two
rounds are the same.

Your keystrokes drive **real, embedded Neovim** when the `nvim` binary is
available on your `PATH` — so the full set of Vim keys works: every motion,
operator, text object, register, count, search and `:substitute`, exactly as
Neovim implements them. When `nvim` isn't installed, a built-in emulator
covers the common motions and operators so the mode still works out of the box.
Either way you edit the buffer live on screen, mode label and all, and text
selected in visual mode is highlighted so you can see the span you're operating
on.

A challenge auto-completes the moment the buffer (and, for motion tasks, the
cursor) reaches the goal state — however you get there. A keystroke the editor
can't act on (a "beep", like `h` at column 0) is counted as a wrong key and the
gopher flinches, but it never blocks you.

Controls during a round:

- `ctrl+n` skip the current challenge, `ctrl+g` reveal a hint (the par
  solution), `ctrl+b` return home. `backspace` and `u` behave like real Vim.

A round is scored per challenge and summed:

- **solve base** — a flat reward for solving the challenge at all.
- **efficiency** — how close your keystroke count is to *par* (the authored
  optimal solution), so knowing the terse command pays off.
- **speed** — how fast you solved it, against an expert budget of ~0.9s per par
  keystroke.

The results screen shows your total score, challenges solved, overall
efficiency, keystrokes-vs-par, wrong keys, and time — and the gopher's closing
expression reflects how you did.

## Scoring

Scoring follows Monkeytype's mechanics.
WPM counts characters in correctly completed words (including spaces), divided
by five and elapsed minutes. Timed rounds also credit a correct unfinished word.
Accuracy measures correct keystrokes; mistakes still count even after correction.

## Words

Each word is picked independently at random from the bundled
[English word list](internal/words/english.txt), so repeats are possible.
The list is embedded at build time.

## Native mascot preview (developer tool)

Interactive (needs a terminal at least 32×14), held pose or animated demo:

```sh
go run ./cmd/mascot-preview
go run ./cmd/mascot-preview --animate
```

## Runtime

The gopher is a tiny 3D model built from 18 shapes: ellipsoids for its body and
face, rounded plates for its teeth. A CPU ray caster shades a 128×96 sample grid
and packs it into 32×12 terminal cells using Unicode blocks and ANSI colors.
Rotation, blinking, and expressions animate the model - not prerecorded frames.
The same renderer drives every screen on a 120 Hz target clock.

## Built with

[Bubble Tea](https://github.com/charmbracelet/bubbletea),
and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

## Attribution

All typing mechanics are based on [Monkeytype](https://monkeytype.com/).

Go gopher original design by Renee French; adapted here into a shaded terminal
mascot. [Original](https://go.dev/blog/gopher) · [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/).

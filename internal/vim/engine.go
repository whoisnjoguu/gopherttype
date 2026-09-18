package vim

import "errors"

// Mode is the current editor mode.
type Mode int

const (
	Normal Mode = iota
	Insert
	Visual
	VisualLine
	CommandLine
)

// Pos is a cursor position: zero-based row and column (in runes).
type Pos struct {
	Row, Col int
}

// Span is an inclusive selection range for highlighting visual mode.
type Span struct {
	Start    Pos
	End      Pos
	Linewise bool
}

// Engine is a Vim editing engine over an in-memory buffer
type Engine interface {
	Feed(token string) bool
	Pending() bool
	Settle()
	Lines() []string
	Cursor() Pos
	Mode() Mode
	ModeLabel() string
	PendingKeys() string
	CommandLine() (kind rune, text string)
	VisualSpan() (Span, bool)
	Matches(goal Goal) bool
	Close()
}

// goalReached is the shared solve test used by every engine
func goalReached(lines []string, cur Pos, normal bool, goal Goal) bool {
	if !normal {
		return false
	}
	if !equalLines(lines, goal.Lines) {
		return false
	}
	if goal.Cursor != nil && cur != *goal.Cursor {
		return false
	}
	return true
}

// less reports whether a comes before b in reading order.
func less(a, b Pos) bool {
	if a.Row != b.Row {
		return a.Row < b.Row
	}
	return a.Col < b.Col
}

func equalLines(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ErrNvimRequired is returned by NewEngine when a real embedded Neovim could not be launched
var ErrNvimRequired = errors.New(`vim: neovim ("nvim") is required to play but is not available`)

// NewEngine builds the live engine seeded with a start state
func NewEngine(lines []string, cur Pos) (Engine, error) {
	engine, err := newNvimEngine(lines, cur)
	if err != nil {
		return nil, ErrNvimRequired
	}
	return engine, nil
}


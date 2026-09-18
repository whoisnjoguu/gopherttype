package vim

import (
	"errors"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/neovim/go-client/nvim"
)

var errNvimUnavailable = errors.New("vim: nvim binary not available")

// Timings for the asynchronous read model
const (
	nvimFeedWait  = 45 * time.Millisecond
	nvimSettleCap = 500 * time.Millisecond
)

// nvimAvailable reports whether the "nvim" binary is on PATH
func nvimAvailable() bool {
	_, err := exec.LookPath("nvim")
	return err == nil
}

// NvimAvailable reports whether the "nvim" binary is on PATH
func NvimAvailable() bool { return nvimAvailable() }

// nvimEngine drives a real, headless Neovim child process over msgpack-RPC
type nvimEngine struct {
	v   *nvim.Nvim
	buf nvim.Buffer
	win nvim.Window

	mu      sync.Mutex
	gen     int
	pendGen int
	pendCh  chan struct{}
	pending bool

	lines   []string
	cur     Pos
	modeStr string
	span    Span
	visual  bool

	cmdKind rune
	cmdText string

	beeped bool
}

// newNvimEngine spawns a headless Neovim seeds it with the start state
func newNvimEngine(lines []string, cur Pos) (*nvimEngine, error) { // returns a ready engine
	if !nvimAvailable() {
		return nil, errNvimUnavailable
	}
	v, err := nvim.NewChildProcess(
		nvim.ChildProcessArgs("--clean", "-n", "--embed", "--headless"),
		nvim.ChildProcessLogf(func(string, ...interface{}) {}),
	)
	if err != nil {
		return nil, err
	}
	buf, err := v.CurrentBuffer()
	if err != nil {
		v.Close()
		return nil, err
	}
	win, err := v.CurrentWindow()
	if err != nil {
		v.Close()
		return nil, err
	}
	// keep Neovim quiet and non-interactive so RPC reads never block on prompts
	_ = v.Command("set noswapfile noundofile nomore noshowcmd noruler noshowmode shortmess+=aoOtTFIcC")

	e := &nvimEngine{v: v, buf: buf, win: win, cur: cur, lines: append([]string(nil), lines...)}
	if err := e.seed(lines, cur); err != nil {
		v.Close()
		return nil, err
	}
	return e, nil
}

func (e *nvimEngine) seed(lines []string, cur Pos) error {
	rows := make([][]byte, len(lines))
	for i, l := range lines {
		rows[i] = []byte(l)
	}
	if len(rows) == 0 {
		rows = [][]byte{{}}
	}
	if err := e.v.SetBufferLines(e.buf, 0, -1, true, rows); err != nil {
		return err
	}
	if err := e.v.SetWindowCursor(e.win, [2]int{cur.Row + 1, cur.Col}); err != nil {
		return err
	}
	e.readNow()
	return nil
}

// Feed sends one keystroke token to Neovim and refreshes the shadow state
func (e *nvimEngine) Feed(token string) bool {
	e.mu.Lock()
	prevLines, prevCur, prevMode := e.lines, e.cur, e.modeStr
	e.gen++
	myGen := e.gen
	e.pending = true
	ch := make(chan struct{})
	e.pendCh, e.pendGen = ch, myGen
	e.mu.Unlock()

	e.beeped = false
	_, _ = e.v.Input(token)

	// interim mode is available immediately
	if m, err := e.v.Mode(); err == nil {
		e.mu.Lock()
		e.modeStr = m.Mode
		e.updateCmdLine(token, m.Mode)
		e.mu.Unlock()
	}

	go e.readAsync(myGen, ch)

	select {
	case <-ch:
	case <-time.After(nvimFeedWait):
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	if e.pending {
		return true // command still in flight; not a beep
	}
	unchanged := equalLines(e.lines, prevLines) && e.cur == prevCur && e.modeStr == prevMode
	if unchanged && token != "<Esc>" {
		e.beeped = true
		return false
	}
	return true
}

// readAsync reads the buffer, cursor, mode and selection
func (e *nvimEngine) readAsync(myGen int, ch chan struct{}) {
	lines, err1 := e.v.BufferLines(e.buf, 0, -1, true)
	pos, err2 := e.v.WindowCursor(e.win)
	mode, err3 := e.v.Mode()

	e.mu.Lock()
	if myGen == e.gen && err1 == nil && err2 == nil {
		e.lines = bytesToLines(lines)
		e.cur = Pos{Row: pos[0] - 1, Col: pos[1]}
		if err3 == nil {
			e.modeStr = mode.Mode
		}
		e.readVisualLocked()
		e.pending = false
	}
	e.mu.Unlock()
	close(ch)
}

// readNow performs a synchronous read used at seed time, when Neovim is at rest
func (e *nvimEngine) readNow() {
	lines, err1 := e.v.BufferLines(e.buf, 0, -1, true)
	pos, err2 := e.v.WindowCursor(e.win)
	mode, _ := e.v.Mode()
	e.mu.Lock()
	if err1 == nil {
		e.lines = bytesToLines(lines)
	}
	if err2 == nil {
		e.cur = Pos{Row: pos[0] - 1, Col: pos[1]}
	}
	if mode != nil {
		e.modeStr = mode.Mode
	}
	e.readVisualLocked()
	e.pending = false
	e.mu.Unlock()
}

// readVisualLocked refreshes the visual selection span. The caller holds e.mu.
func (e *nvimEngine) readVisualLocked() {
	if !strings.HasPrefix(e.modeStr, "v") && !strings.HasPrefix(e.modeStr, "V") && !strings.HasPrefix(e.modeStr, "\x16") {
		e.visual = false
		return
	}
	var anchor []int
	if err := e.v.Call("getpos", &anchor, "v"); err != nil || len(anchor) < 3 {
		e.visual = false
		return
	}
	start := Pos{Row: anchor[1] - 1, Col: anchor[2] - 1}
	end := e.cur
	if less(end, start) {
		start, end = end, start
	}
	e.span = Span{Start: start, End: end, Linewise: strings.HasPrefix(e.modeStr, "V")}
	e.visual = true
}

// updateCmdLine maintains a shadow of the command-line being typed
func (e *nvimEngine) updateCmdLine(token, mode string) {
	if !strings.HasPrefix(mode, "c") {
		e.cmdKind, e.cmdText = 0, ""
		return
	}
	if e.cmdKind == 0 {
		if token == ":" || token == "/" || token == "?" {
			e.cmdKind = rune(token[0])
			e.cmdText = ""
		}
		return
	}
	switch token {
	case "<BS>":
		if r := []rune(e.cmdText); len(r) > 0 {
			e.cmdText = string(r[:len(r)-1])
		}
	case "<CR>", "<Esc>":
		// handled by the mode flip on the next read
	default:
		if len(token) > 0 && token[0] >= 0x20 {
			e.cmdText += token
		}
	}
}

// Pending reports whether the last keystroke's effect is still catching up
func (e *nvimEngine) Pending() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.pending
}

// Settle blocks until the latest keystroke's read completes
func (e *nvimEngine) Settle() {
	e.mu.Lock()
	ch, pending := e.pendCh, e.pending
	e.mu.Unlock()
	if !pending || ch == nil {
		return
	}
	select {
	case <-ch:
	case <-time.After(nvimSettleCap):
	}
}

func (e *nvimEngine) Lines() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]string(nil), e.lines...)
}

func (e *nvimEngine) Cursor() Pos {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.cur
}

func (e *nvimEngine) Mode() Mode {
	e.mu.Lock()
	defer e.mu.Unlock()
	return nvimMode(e.modeStr)
}

func (e *nvimEngine) ModeLabel() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	switch nvimMode(e.modeStr) {
	case Insert:
		if strings.HasPrefix(e.modeStr, "R") {
			return "REPLACE"
		}
		return "INSERT"
	case Visual:
		if strings.HasPrefix(e.modeStr, "\x16") {
			return "V-BLOCK"
		}
		return "VISUAL"
	case VisualLine:
		return "V-LINE"
	case CommandLine:
		return "COMMAND"
	default:
		if e.modeStr == "n" {
			return "NORMAL"
		}
		return "PENDING"
	}
}

// PendingKeys is not reconstructed for the Neovim backend
func (e *nvimEngine) PendingKeys() string { return "" }

func (e *nvimEngine) CommandLine() (rune, string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.cmdKind, e.cmdText
}

func (e *nvimEngine) VisualSpan() (Span, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.span, e.visual
}

func (e *nvimEngine) Matches(goal Goal) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return goalReached(e.lines, e.cur, e.modeStr == "n", goal)
}

// Close terminates the Neovim child process
func (e *nvimEngine) Close() {
	v := e.v
	go func() { _ = v.Close() }()
}

func nvimMode(mode string) Mode {
	switch {
	case mode == "":
		return Normal
	case strings.HasPrefix(mode, "i"), strings.HasPrefix(mode, "R"):
		return Insert
	case strings.HasPrefix(mode, "V"):
		return VisualLine
	case strings.HasPrefix(mode, "\x16"):
		return Visual
	case strings.HasPrefix(mode, "v"):
		return Visual
	case strings.HasPrefix(mode, "c"):
		return CommandLine
	default:
		return Normal
	}
}

func bytesToLines(rows [][]byte) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = string(r)
	}
	return out
}

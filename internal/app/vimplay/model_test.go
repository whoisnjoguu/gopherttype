package vimplay

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/mkhamat/gopherttype/internal/app/ui"
	"github.com/mkhamat/gopherttype/internal/vim"
)

func letter(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: r, Text: string(r)})
}

func ctrl(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Mod: tea.ModCtrl, Code: r})
}

// pressKey feeds a key and, if the real nvim engine reports Pending once,
// settles it before returning the resulting command
func pressKey(t *testing.T, m *Model, key tea.KeyPressMsg) tea.Cmd {
	t.Helper()
	cmd := m.Update(key)
	if m.session != nil && m.session.Pending() {
		m.session.WaitSettle()
		cmd = m.settle(time.Now())
	}
	return cmd
}

func round() []vim.Challenge {
	return []vim.Challenge{
		vim.NewChallenge("delete line", []string{"a", "b"}, vim.Pos{Row: 0, Col: 0}, vim.Easy,
			vim.Goal{Lines: []string{"b"}}, "dd"),
		vim.NewChallenge("delete char", []string{"cat"}, vim.Pos{Row: 0, Col: 0}, vim.Easy,
			vim.Goal{Lines: []string{"at"}}, "x"),
	}
}

func TestVimPlaySolvesRound(t *testing.T) {
	if !vim.NvimAvailable() {
		t.Skip("nvim not installed")
	}
	m := New(vim.Easy, round())
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	pressKey(t, m, letter('d'))
	if cmd := pressKey(t, m, letter('d')); cmd != nil {
		if _, finished := cmd().(FinishedMsg); finished {
			t.Fatal("round finished after only the first challenge")
		}
	}
	// Second challenge: a single 'x' completes the round.
	cmd := pressKey(t, m, letter('x'))
	if cmd == nil {
		t.Fatal("completing the last challenge must return a command")
	}
	msg, ok := cmd().(FinishedMsg)
	if !ok {
		t.Fatalf("final command = %T, want FinishedMsg", cmd())
	}
	if msg.Report.Solved != 2 || msg.Report.Total != 2 {
		t.Fatalf("report solved/total = %d/%d, want 2/2", msg.Report.Solved, msg.Report.Total)
	}
}

func TestVimPlayRejectsWrongKey(t *testing.T) {
	if !vim.NvimAvailable() {
		t.Skip("nvim not installed")
	}
	m := New(vim.Easy, round())
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	// 'z' is not a valid first key for "dd"; it must not advance or solve.
	if cmd := m.Update(letter('z')); cmd != nil {
		if _, finished := cmd().(FinishedMsg); finished {
			t.Fatal("a rejected key must not finish the round")
		}
	}
	if !strings.Contains(ansiStrip(m.View()), "1/2") {
		t.Error("still expected to be on the first challenge")
	}
}

func TestVimPlayHomeControl(t *testing.T) {
	if !vim.NvimAvailable() {
		t.Skip("nvim not installed")
	}
	m := New(vim.Easy, round())
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	cmd := m.Update(ctrl('b'))
	if cmd == nil {
		t.Fatal("ctrl+b must return a command")
	}
	if _, ok := cmd().(HomeMsg); !ok {
		t.Fatalf("ctrl+b = %T, want HomeMsg", cmd())
	}
}

func TestVimPlaySkipAdvances(t *testing.T) {
	if !vim.NvimAvailable() {
		t.Skip("nvim not installed")
	}
	m := New(vim.Easy, round())
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.Update(ctrl('n')) // skip challenge one
	if !strings.Contains(ansiStrip(m.View()), "2/2") {
		t.Error("ctrl+n should advance to the second challenge")
	}
}

func TestVimPlayResizeBelowMinimum(t *testing.T) {
	if !vim.NvimAvailable() {
		t.Skip("nvim not installed")
	}
	m := New(vim.Easy, round())
	m.Update(tea.WindowSizeMsg{Width: 20, Height: 8})
	if m.View() != ui.ResizeView(20, 8, "Ctrl+B home · Ctrl+C quit") {
		t.Error("below minimum must show the resize prompt")
	}
}

// ansiStrip removes styling so tests can assert on plain text.
func ansiStrip(s string) string {
	var b strings.Builder
	inEscape := false
	for _, r := range s {
		switch {
		case r == '\x1b':
			inEscape = true
		case inEscape && (r == 'm'):
			inEscape = false
		case !inEscape:
			b.WriteRune(r)
		}
	}
	return b.String()
}

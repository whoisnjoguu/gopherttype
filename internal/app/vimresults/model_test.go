package vimresults

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/mkhamat/gopherttype/internal/app/ui"
	"github.com/mkhamat/gopherttype/internal/vim"
)

func strongReport() vim.Report {
	results := []vim.ChallengeResult{
		{Task: "a", Solved: true, Par: 3, Used: 3, Elapsed: time.Second},
		{Task: "b", Solved: true, Par: 1, Used: 1, Elapsed: time.Second},
	}
	report := vim.Report{
		Total: 2, Solved: 2, TotalPar: 4, TotalUsed: 4,
		Duration: 2 * time.Second, Results: results,
	}
	for _, r := range results {
		report.Score += r.Score()
	}
	return report
}

func TestVimResultsRendersScoreboard(t *testing.T) {
	m := New(strongReport())
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	view := m.View()
	for _, want := range []string{"pts", "2/2", "100%"} {
		if !strings.Contains(stripANSI(view), want) {
			t.Errorf("scoreboard missing %q", want)
		}
	}
}

func TestVimResultsKeys(t *testing.T) {
	m := New(strongReport())
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})); cmd == nil {
		t.Fatal("enter must return a command")
	} else if _, ok := cmd().(RetryMsg); !ok {
		t.Fatalf("enter = %T, want RetryMsg", cmd())
	}
	if cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab})); cmd == nil {
		t.Fatal("tab must return a command")
	} else if _, ok := cmd().(HomeMsg); !ok {
		t.Fatalf("tab = %T, want HomeMsg", cmd())
	}
}

func TestVimResultsResizeBelowMinimum(t *testing.T) {
	m := New(strongReport())
	m.Update(tea.WindowSizeMsg{Width: 20, Height: 8})
	if m.View() != ui.ResizeView(20, 8, "q / Ctrl+C quit") {
		t.Error("below minimum must show the resize prompt")
	}
}

func stripANSI(s string) string {
	var b strings.Builder
	inEscape := false
	for _, r := range s {
		switch {
		case r == '\x1b':
			inEscape = true
		case inEscape && r == 'm':
			inEscape = false
		case !inEscape:
			b.WriteRune(r)
		}
	}
	return b.String()
}
